package app

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func printProcessSummary(limit int) {
	processes, ok := readTopProcesses()
	if !ok {
		fmt.Println("Nao consegui ler os processos. Tente rodar pelo Terminal.")
		return
	}

	printProcessFamilies(processes)

	fmt.Println()
	fmt.Println("Processos mais pesados")
	w := newTable()
	fmt.Fprintln(w, "PID\tMEM\tCOMANDO")
	for index, process := range processes {
		if index >= limit {
			break
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", process.pid, process.memRaw, process.command)
	}
	w.Flush()
}

func printProcessFamilies(processes []processInfo) {
	type familyTotal struct {
		name  string
		bytes uint64
	}

	totals := map[string]uint64{}
	for _, process := range processes {
		totals[processFamily(process.command)] += process.bytes
	}

	families := make([]familyTotal, 0, len(totals))
	for name, bytes := range totals {
		if bytes == 0 {
			continue
		}
		families = append(families, familyTotal{name: name, bytes: bytes})
	}
	sort.Slice(families, func(i, j int) bool {
		return families[i].bytes > families[j].bytes
	})

	fmt.Println("Familias de processos")
	w := newTable()
	fmt.Fprintln(w, "FAMILIA\tMEM")
	for index, family := range families {
		if index >= 8 {
			break
		}
		fmt.Fprintf(w, "%s\t%s\n", family.name, formatBytes(family.bytes))
	}
	w.Flush()

	if len(families) == 0 {
		return
	}

	top := families[0]
	switch {
	case top.bytes >= 6*1024*1024*1024:
		tip(fmt.Sprintf("%s esta muito pesado. Fechar/reiniciar essa familia deve aliviar a memoria rapidamente.", top.name))
	case top.bytes >= 3*1024*1024*1024:
		tip(fmt.Sprintf("%s e o maior consumidor agora. Se o Mac travar, comece por ele.", top.name))
	default:
		tip("Nenhuma familia parece absurdamente pesada neste momento.")
	}
}

func readTopProcesses() ([]processInfo, bool) {
	result := runCommand("top", "-l", "1", "-o", "mem", "-stats", "pid,command,mem")
	if !result.ok {
		return readPSProcesses()
	}

	lines := nonEmptyLines(result.output)
	headerIndex := -1
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "PID") {
			headerIndex = index
			break
		}
	}
	if headerIndex == -1 {
		return nil, false
	}

	processes := []processInfo{}
	for _, line := range lines[headerIndex+1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		memRaw := fields[len(fields)-1]
		processes = append(processes, processInfo{
			pid:     fields[0],
			command: strings.Join(fields[1:len(fields)-1], " "),
			memRaw:  memRaw,
			bytes:   parseMemoryToBytes(memRaw),
		})
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].bytes > processes[j].bytes
	})

	return processes, len(processes) > 0
}

func readPSProcesses() ([]processInfo, bool) {
	result := runCommand("ps", "axo", "pid=,comm=,rss=")
	if !result.ok {
		return nil, false
	}

	processes := []processInfo{}
	for _, line := range nonEmptyLines(result.output) {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		rssKB, err := strconv.ParseUint(fields[len(fields)-1], 10, 64)
		if err != nil {
			continue
		}

		bytes := rssKB * 1024
		processes = append(processes, processInfo{
			pid:     fields[0],
			command: strings.Join(fields[1:len(fields)-1], " "),
			memRaw:  formatBytes(bytes),
			bytes:   bytes,
		})
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].bytes > processes[j].bytes
	})

	return processes, len(processes) > 0
}

func processFamily(command string) string {
	lower := strings.ToLower(command)
	switch {
	case strings.Contains(lower, "google chrome"):
		return "Chrome"
	case strings.Contains(lower, "codex"):
		return "Codex"
	case strings.Contains(lower, "code helper") || lower == "code":
		return "VS Code"
	case strings.Contains(lower, "docker") || strings.Contains(lower, "com.docker"):
		return "Docker"
	case strings.Contains(lower, "qemu") || strings.Contains(lower, "emulator") || strings.Contains(lower, "com.apple.virtua"):
		return "VM/Emulator"
	case strings.Contains(lower, "java"):
		return "Java"
	case lower == "node" || strings.Contains(lower, "node "):
		return "Node"
	case strings.Contains(lower, "postgres") || strings.Contains(lower, "mysql"):
		return "Databases"
	case strings.Contains(lower, "dart") || strings.Contains(lower, "flutter"):
		return "Dart/Flutter"
	case strings.Contains(lower, "windowserver") || strings.Contains(lower, "finder"):
		return "macOS UI"
	default:
		return "Other"
	}
}
