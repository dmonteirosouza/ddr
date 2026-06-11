package app

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func printSortedSizeTable(entries []sizeEntry, limit int) {
	sizes := make([]folderSize, 0, len(entries))
	for _, entry := range entries {
		size := duSize(expand(entry.path))
		size.label = entry.label
		size.path = entry.path
		sizes = append(sizes, size)
	}

	sort.Slice(sizes, func(i, j int) bool {
		return sizes[i].bytes > sizes[j].bytes
	})

	w := newTable()
	fmt.Fprintln(w, "STATUS\tTAMANHO\tITEM\tCAMINHO")
	reviewCount := 0
	for index, size := range sizes {
		if limit > 0 && index >= limit {
			break
		}
		status := "OK"
		if !size.ok {
			status = "PULAR"
		} else if size.bytes >= 20*1024*1024*1024 {
			status = "REVISAR"
			reviewCount++
		} else if size.bytes >= 5*1024*1024*1024 {
			status = "ATENCAO"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", statusBadge(status), size.size, size.label, size.path)
	}
	w.Flush()

	if reviewCount > 0 {
		tip("Comece pelos itens REVISAR. Em geral Docker, Downloads e caches de ferramentas liberam mais espaco.")
	} else {
		tip("Nenhuma pasta comum passou de 20 GB nesta lista.")
	}
}

func duSize(fullPath string) folderSize {
	if _, err := os.Stat(fullPath); err != nil {
		return folderSize{size: "-", ok: false}
	}

	result := runCommand("du", "-sk", fullPath)
	if !result.ok && strings.TrimSpace(result.output) == "" {
		return folderSize{size: "blocked", ok: false}
	}

	fields := strings.Fields(result.output)
	if len(fields) == 0 {
		return folderSize{size: "-", ok: false}
	}

	kib, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return folderSize{size: fields[0], ok: false}
	}

	bytes := kib * 1024
	return folderSize{size: formatBytes(bytes), bytes: bytes, ok: true}
}

func printTopProcesses(limit int) {
	result := runCommand("top", "-l", "1", "-o", "mem", "-stats", "pid,command,mem")
	if !result.ok {
		fmt.Println(strings.TrimSpace(result.output))
		return
	}

	lines := strings.Split(result.output, "\n")
	headerIndex := -1
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "PID") {
			headerIndex = index
			break
		}
	}

	if headerIndex == -1 {
		fmt.Println(truncateLines(result.output, limit+12))
		return
	}

	fmt.Println(strings.Join(lines[:headerIndex+1], "\n"))
	end := headerIndex + 1 + limit
	if end > len(lines) {
		end = len(lines)
	}
	fmt.Println(strings.Join(lines[headerIndex+1:end], "\n"))
}

func printSizeTable(entries []sizeEntry) {
	for _, entry := range entries {
		size := du(expand(entry.path))
		fmt.Printf("%-28s %8s  %s\n", entry.label, size, entry.path)
	}
}

func du(fullPath string) string {
	if _, err := os.Stat(fullPath); err != nil {
		return "-"
	}

	result := runCommand("du", "-sh", fullPath)
	if !result.ok {
		return "blocked"
	}

	fields := strings.Fields(result.output)
	if len(fields) == 0 {
		return "-"
	}
	return fields[0]
}
