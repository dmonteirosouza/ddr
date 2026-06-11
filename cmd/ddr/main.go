package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type sizeEntry struct {
	label string
	path  string
}

type folderSize struct {
	label string
	path  string
	size  string
	bytes uint64
	ok    bool
}

type commandResult struct {
	ok     bool
	output string
	err    error
}

type diskInfo struct {
	mount      string
	totalBytes uint64
	usedBytes  uint64
	freeBytes  uint64
	usedPct    float64
}

type swapUsage struct {
	totalMB float64
	usedMB  float64
	freeMB  float64
}

type memoryInfo struct {
	ramBytes       uint64
	swap           swapUsage
	swapOK         bool
	freePct        int
	freePctOK      bool
	compressorSize string
}

type processInfo struct {
	pid     string
	command string
	memRaw  string
	bytes   uint64
}

type dockerRow struct {
	kind          string
	total         string
	active        string
	size          string
	reclaimable   string
	reclaimBytes  uint64
	reclaimPctRaw string
}

var (
	homeDir = mustHomeDir()

	swapRegexp       = regexp.MustCompile(`total = ([0-9.]+)M\s+used = ([0-9.]+)M\s+free = ([0-9.]+)M`)
	memoryFreeRegexp = regexp.MustCompile(`System-wide memory free percentage:\s+([0-9]+)%`)
	compressorRegexp = regexp.MustCompile(`Pages used by compressor:\s+([0-9]+)`)

	vscodeSettingsPath         = expand("~/Library/Application Support/Code/User/settings.json")
	vscodeWorkspaceStoragePath = expand("~/Library/Application Support/Code/User/workspaceStorage")

	candidatePaths = []sizeEntry{
		{"Docker Desktop", "~/Library/Containers/com.docker.docker"},
		{"VS Code workspaceStorage", "~/Library/Application Support/Code/User/workspaceStorage"},
		{"VS Code extensions", "~/.vscode/extensions"},
		{"npm cache", "~/.npm"},
		{"Gradle", "~/.gradle"},
		{"Pub cache", "~/.pub-cache"},
		{"User caches", "~/Library/Caches"},
		{"Downloads", "~/Downloads"},
		{"CoreSimulator", "~/Library/Developer/CoreSimulator"},
		{"Google app data", "~/Library/Application Support/Google"},
		{"Claude app data", "~/Library/Application Support/Claude"},
		{"VS Code app data", "~/Library/Application Support/Code"},
	}

	watcherExcludes = map[string]any{
		"**/.git/objects/**":       true,
		"**/.git/subtree-cache/**": true,
		"**/node_modules/**":       true,
		"**/.next/**":              true,
		"**/.nuxt/**":              true,
		"**/dist/**":               true,
		"**/build/**":              true,
		"**/coverage/**":           true,
		"**/.turbo/**":             true,
		"**/.cache/**":             true,
		"**/.dart_tool/**":         true,
		"**/.gradle/**":            true,
		"**/Pods/**":               true,
		"**/vendor/**":             true,
	}
)

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ddr: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(args []string) error {
	command, flags := parseArgs(args)

	switch command {
	case "", "scan":
		return scan()
	case "help", "--help", "-h":
		printHelp()
		return nil
	case "memory", "mem":
		return memory()
	case "docker":
		return dockerReport()
	case "vscode", "code":
		return vscode(flags)
	case "clean":
		return clean(flags)
	case "chrome":
		chrome()
		return nil
	default:
		return fmt.Errorf("unknown command %q. Run \"ddr help\"", command)
	}
}

func parseArgs(args []string) (string, map[string]bool) {
	command := ""
	flags := map[string]bool{}

	for _, arg := range args {
		if command == "" && !strings.HasPrefix(arg, "-") {
			command = arg
			continue
		}
		flags[arg] = true
	}

	return command, flags
}

func printHelp() {
	fmt.Println(`ddr - small macOS dev doctor

Usage:
  ddr scan
  ddr memory
  ddr docker
  ddr vscode
  ddr vscode --apply
  ddr clean
  ddr clean --safe --yes
  ddr clean --all-safe --yes
  ddr clean --vscode-storage --yes
  ddr chrome

Cleaning flags:
  --safe           Docker build cache + npm cache + Gradle caches
  --all-safe       --safe plus stopped containers, unused networks, unused images
  --vscode-storage remove VS Code workspaceStorage; use with VS Code closed
  --yes            required to actually delete anything

Notes:
  Docker volumes are never deleted automatically.
  VS Code settings are backed up before writes.`)
}

func scan() error {
	title("ddr scan")

	section("Disk")
	printDiskSummary()

	section("Memory")
	printMemorySummary()

	section("Top Memory Processes")
	printProcessSummary(12)

	section("Docker")
	printDockerSummary(false)

	section("Common Heavy Folders")
	printSortedSizeTable(candidatePaths, 12)

	fmt.Println("\nNext:")
	fmt.Println("  ddr clean                 show safe cleanup plan")
	fmt.Println("  ddr clean --safe --yes    run conservative cleanup")
	fmt.Println("  ddr vscode --apply        apply lighter VS Code settings with backup")
	return nil
}

func memory() error {
	title("ddr memory")

	section("Memory")
	printMemorySummary()

	section("Top Memory Processes")
	printProcessSummary(25)
	return nil
}

func dockerReport() error {
	if !commandExists("docker") {
		fmt.Println("Docker CLI not found.")
		return nil
	}

	title("ddr docker")

	section("Docker Usage")
	printDockerSummary(true)

	fmt.Println("\nCleanup:")
	fmt.Println("  ddr clean --safe --yes      prune build cache only, plus npm/Gradle caches")
	fmt.Println("  ddr clean --all-safe --yes  also prune stopped containers and unused images")
	fmt.Println("\nDocker volumes are intentionally preserved.")
	return nil
}

func vscode(flags map[string]bool) error {
	title("ddr vscode")

	section("VS Code Storage")
	printSortedSizeTable([]sizeEntry{
		{"workspaceStorage", "~/Library/Application Support/Code/User/workspaceStorage"},
		{"globalStorage", "~/Library/Application Support/Code/User/globalStorage"},
		{"History", "~/Library/Application Support/Code/User/History"},
		{"extensions", "~/.vscode/extensions"},
	}, 10)

	section("Settings")
	printVscodeSettingsStatus()

	section("Installed Extensions")
	if commandExists("code") {
		printExtensionList()
	} else {
		fmt.Println("code CLI not found. In VS Code, run: Shell Command: Install 'code' command in PATH")
	}

	if !flags["--apply"] {
		fmt.Println("\nTo apply lighter VS Code/Codex settings:")
		fmt.Println("  ddr vscode --apply")
		return nil
	}

	return applyVscodeSettings()
}

func applyVscodeSettings() error {
	section("Applying VS Code Settings")

	text, err := os.ReadFile(vscodeSettingsPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(vscodeSettingsPath), 0o755); err != nil {
			return err
		}
		text = []byte("{}\n")
	} else if err != nil {
		return err
	}

	backupPath := fmt.Sprintf("%s.ddr-%s.bak", vscodeSettingsPath, timestamp())
	if err := os.WriteFile(backupPath, text, 0o644); err != nil {
		return err
	}

	var settings map[string]any
	if err := json.Unmarshal(stripJSONComments(text), &settings); err != nil {
		return fmt.Errorf("could not parse VS Code settings: %w", err)
	}
	if settings == nil {
		settings = map[string]any{}
	}

	settings["chatgpt.commentCodeLensEnabled"] = false
	settings["chatgpt.openOnStartup"] = false
	settings["gitlens.ai.enabled"] = false
	settings["git.autofetch"] = false
	settings["workbench.editor.limit.enabled"] = true
	settings["workbench.editor.limit.value"] = 8
	settings["workbench.editor.limit.perEditorGroup"] = false
	settings["js/ts.tsserver.maxMemory"] = float64(2048)
	settings["js/ts.preferences.includePackageJsonAutoImports"] = "off"

	settings["files.watcherExclude"] = mergeBoolMap(settings["files.watcherExclude"], watcherExcludes)
	settings["search.exclude"] = mergeBoolMap(settings["search.exclude"], watcherExcludes)

	updated, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return err
	}
	updated = append(updated, '\n')

	if err := os.WriteFile(vscodeSettingsPath, updated, 0o644); err != nil {
		return err
	}

	fmt.Printf("Updated: %s\n", vscodeSettingsPath)
	fmt.Printf("Backup:  %s\n", backupPath)
	fmt.Println("\nReload VS Code to apply: Cmd+Shift+P -> Developer: Reload Window")
	return nil
}

func clean(flags map[string]bool) error {
	safe := flags["--safe"] || flags["--all-safe"]
	allSafe := flags["--all-safe"]
	vscodeStorage := flags["--vscode-storage"]
	yes := flags["--yes"] || flags["-y"]

	title("ddr clean")

	section("Cleanup Plan")

	if !safe && !allSafe && !vscodeStorage {
		fmt.Println("No cleanup set selected.")
		fmt.Println("\nTry:")
		fmt.Println("  ddr clean --safe")
		fmt.Println("  ddr clean --safe --yes")
		fmt.Println("  ddr clean --all-safe --yes")
		fmt.Println("  ddr clean --vscode-storage --yes")
		return nil
	}

	actions := [][2]string{}
	if safe {
		actions = append(actions, [2]string{"Docker build cache", "docker builder prune -af"})
		actions = append(actions, [2]string{"npm cache", "npm cache clean --force"})
		actions = append(actions, [2]string{"Gradle caches", "remove ~/.gradle/caches and ~/.gradle/wrapper/dists"})
	}
	if allSafe {
		actions = append(actions, [2]string{"Docker unused objects", "docker system prune -af (preserves volumes)"})
	}
	if vscodeStorage {
		actions = append(actions, [2]string{"VS Code workspaceStorage", "remove cached workspace state; close VS Code first"})
	}

	w := newTable()
	fmt.Fprintln(w, "ITEM\tACTION")
	for _, action := range actions {
		fmt.Fprintf(w, "%s\t%s\n", action[0], action[1])
	}
	w.Flush()

	fmt.Println("\nDocker volumes are not removed.")

	if !yes {
		fmt.Println("\nDry run only. Re-run with --yes to execute.")
		return nil
	}

	if safe {
		if commandExists("docker") {
			section("Pruning Docker Build Cache")
			printCommand("docker", []string{"builder", "prune", "-af"}, true, 200)
		}

		if commandExists("npm") {
			section("Cleaning npm Cache")
			printCommand("npm", []string{"cache", "clean", "--force"}, true, 80)
		}

		section("Cleaning Gradle Caches")
		removeKnownPath(expand("~/.gradle/caches"))
		removeKnownPath(expand("~/.gradle/wrapper/dists"))
	}

	if allSafe && commandExists("docker") {
		section("Pruning Docker Unused Objects")
		printCommand("docker", []string{"system", "prune", "-af"}, true, 300)
	}

	if vscodeStorage {
		section("Cleaning VS Code workspaceStorage")
		removeKnownPath(vscodeWorkspaceStoragePath)
	}

	section("After")
	printCommand("df", []string{"-h"}, true, 80)
	return nil
}

func chrome() {
	title("ddr chrome")

	section("Chrome Checklist")
	fmt.Println("Open: chrome://settings/performance")
	fmt.Println("- Enable Memory Saver.")
	fmt.Println("- Use the most aggressive Memory Saver mode if available.")
	fmt.Println("- Keep only critical sites always active.")
	fmt.Println("\nOpen: chrome://settings/system")
	fmt.Println("- Disable background apps after Chrome closes.")
	fmt.Println("- Keep hardware acceleration enabled unless graphics get unstable.")
	fmt.Println("\nOpen: chrome://extensions")
	fmt.Println("- Disable unused extensions, especially AI, wallet, capture, and productivity extensions.")
	fmt.Println("\nUse Shift+Esc inside Chrome to sort tabs/extensions by memory.")
}

func printDiskSummary() {
	info, ok := readDiskInfo("/System/Volumes/Data")
	if !ok {
		info, ok = readDiskInfo("/")
	}
	if !ok {
		printCommand("df", []string{"-h"}, true, 80)
		return
	}

	status := "OK"
	if info.freeBytes < 20*1024*1024*1024 || info.usedPct >= 90 {
		status = "CRITICAL"
	} else if info.freeBytes < 50*1024*1024*1024 || info.usedPct >= 80 {
		status = "WARN"
	}

	fmt.Printf("%-14s %s\n", "Status", status)
	fmt.Printf("%-14s %s\n", "Mount", info.mount)
	fmt.Printf("%-14s %s / %s used\n", "Usage", formatBytes(info.usedBytes), formatBytes(info.totalBytes))
	fmt.Printf("%-14s %s\n", "Free", formatBytes(info.freeBytes))
	fmt.Printf("%-14s %s %.0f%%\n", "Meter", meter(info.usedPct, 24), info.usedPct)
}

func readDiskInfo(mount string) (diskInfo, bool) {
	result := runCommand("df", "-k", mount)
	if !result.ok {
		return diskInfo{}, false
	}

	lines := nonEmptyLines(result.output)
	if len(lines) < 2 {
		return diskInfo{}, false
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 9 {
		return diskInfo{}, false
	}

	totalKB, err1 := strconv.ParseUint(fields[1], 10, 64)
	usedKB, err2 := strconv.ParseUint(fields[2], 10, 64)
	freeKB, err3 := strconv.ParseUint(fields[3], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || totalKB == 0 {
		return diskInfo{}, false
	}

	mountedOn := strings.Join(fields[8:], " ")
	usedPct := float64(usedKB) / float64(totalKB) * 100
	return diskInfo{
		mount:      mountedOn,
		totalBytes: totalKB * 1024,
		usedBytes:  usedKB * 1024,
		freeBytes:  freeKB * 1024,
		usedPct:    usedPct,
	}, true
}

func printMemorySummary() {
	info := readMemoryInfo()

	if info.ramBytes > 0 {
		fmt.Printf("%-14s %s\n", "RAM", formatBytes(info.ramBytes))
	}

	if info.freePctOK {
		status := "OK"
		if info.freePct < 10 {
			status = "CRITICAL"
		} else if info.freePct < 20 {
			status = "WARN"
		}
		usedPct := 100 - float64(info.freePct)
		fmt.Printf("%-14s %s (%d%% free)\n", "Pressure", status, info.freePct)
		fmt.Printf("%-14s %s %.0f%%\n", "Meter", meter(usedPct, 24), usedPct)
	}

	if info.compressorSize != "" {
		fmt.Printf("%-14s %s\n", "Compressor", info.compressorSize)
	}

	if info.swapOK {
		status := "OK"
		swapPct := 0.0
		if info.swap.totalMB > 0 {
			swapPct = info.swap.usedMB / info.swap.totalMB * 100
		}
		if swapPct >= 85 {
			status = "CRITICAL"
		} else if swapPct >= 60 {
			status = "WARN"
		}

		fmt.Printf("%-14s %s (%s / %s)\n", "Swap", status, formatMegabytes(info.swap.usedMB), formatMegabytes(info.swap.totalMB))
		fmt.Printf("%-14s %s %.0f%%\n", "Swap meter", meter(swapPct, 24), swapPct)
	}

	if !info.freePctOK && !info.swapOK {
		fmt.Println("Could not read memory summary. Try running from Terminal outside a restricted session.")
	}
}

func readMemoryInfo() memoryInfo {
	info := memoryInfo{}

	if result := runCommand("sysctl", "hw.memsize"); result.ok {
		fields := strings.Fields(result.output)
		if len(fields) >= 2 {
			if value, err := strconv.ParseUint(fields[len(fields)-1], 10, 64); err == nil {
				info.ramBytes = value
			}
		}
	}

	if result := runCommand("sysctl", "vm.swapusage"); result.ok {
		if match := swapRegexp.FindStringSubmatch(result.output); len(match) == 4 {
			total, _ := strconv.ParseFloat(match[1], 64)
			used, _ := strconv.ParseFloat(match[2], 64)
			free, _ := strconv.ParseFloat(match[3], 64)
			info.swap = swapUsage{totalMB: total, usedMB: used, freeMB: free}
			info.swapOK = true
		}
	}

	if result := runCommand("memory_pressure"); result.ok {
		if match := memoryFreeRegexp.FindStringSubmatch(result.output); len(match) == 2 {
			if value, err := strconv.Atoi(match[1]); err == nil {
				info.freePct = value
				info.freePctOK = true
			}
		}
		if match := compressorRegexp.FindStringSubmatch(result.output); len(match) == 2 {
			if pages, err := strconv.ParseUint(match[1], 10, 64); err == nil {
				info.compressorSize = formatBytes(pages * 16384)
			}
		}
	}

	return info
}

func printProcessSummary(limit int) {
	processes, ok := readTopProcesses()
	if !ok {
		fmt.Println("Could not read processes. Try running from Terminal outside a restricted session.")
		return
	}

	printProcessFamilies(processes)

	fmt.Println()
	fmt.Println("Top processes")
	w := newTable()
	fmt.Fprintln(w, "PID\tMEM\tCOMMAND")
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

	fmt.Println("Process families")
	w := newTable()
	fmt.Fprintln(w, "FAMILY\tMEM")
	for index, family := range families {
		if index >= 8 {
			break
		}
		fmt.Fprintf(w, "%s\t%s\n", family.name, formatBytes(family.bytes))
	}
	w.Flush()
}

func readTopProcesses() ([]processInfo, bool) {
	result := runCommand("top", "-l", "1", "-o", "mem", "-stats", "pid,command,mem")
	if !result.ok {
		return nil, false
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

func printDockerSummary(includeAdvice bool) {
	if !commandExists("docker") {
		fmt.Println("Docker CLI not found.")
		return
	}

	result := runCommand("docker", "system", "df")
	if !result.ok {
		fmt.Println(strings.TrimSpace(result.output))
		return
	}

	rows := parseDockerSystemDF(result.output)
	if len(rows) == 0 {
		fmt.Println(strings.TrimSpace(result.output))
		return
	}

	totalReclaim := uint64(0)
	for _, row := range rows {
		totalReclaim += row.reclaimBytes
	}

	status := "OK"
	if totalReclaim >= 30*1024*1024*1024 {
		status = "WARN"
	}
	if totalReclaim >= 60*1024*1024*1024 {
		status = "CRITICAL"
	}

	fmt.Printf("%-14s %s\n", "Status", status)
	fmt.Printf("%-14s %s reclaimable reported by Docker\n", "Reclaimable", formatBytes(totalReclaim))

	w := newTable()
	fmt.Fprintln(w, "TYPE\tTOTAL\tACTIVE\tSIZE\tRECLAIMABLE")
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.kind, row.total, row.active, row.size, row.reclaimable)
	}
	w.Flush()

	if includeAdvice {
		fmt.Println("\nVolumes are preserved by ddr. Review them manually before deleting.")
	}
}

func parseDockerSystemDF(output string) []dockerRow {
	lines := nonEmptyLines(output)
	rows := []dockerRow{}

	for _, line := range lines {
		if strings.HasPrefix(line, "TYPE ") || strings.HasPrefix(line, "TYPE\t") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		kind := fields[0]
		offset := 1
		if len(fields) >= 6 && fields[0] == "Local" && fields[1] == "Volumes" {
			kind = "Local Volumes"
			offset = 2
		}
		if len(fields) >= 6 && fields[0] == "Build" && fields[1] == "Cache" {
			kind = "Build Cache"
			offset = 2
		}
		if len(fields) < offset+4 {
			continue
		}

		reclaimFieldIndex := offset + 3
		reclaimable := fields[reclaimFieldIndex]
		reclaimPctRaw := ""
		if len(fields) > reclaimFieldIndex+1 {
			reclaimable = strings.Join(fields[reclaimFieldIndex:], " ")
			reclaimPctRaw = fields[len(fields)-1]
		}

		rows = append(rows, dockerRow{
			kind:          kind,
			total:         fields[offset],
			active:        fields[offset+1],
			size:          fields[offset+2],
			reclaimable:   reclaimable,
			reclaimBytes:  parseHumanSize(fields[reclaimFieldIndex]),
			reclaimPctRaw: reclaimPctRaw,
		})
	}

	return rows
}

func printVscodeSettingsStatus() {
	text, err := os.ReadFile(vscodeSettingsPath)
	if err != nil {
		fmt.Println("No VS Code settings found.")
		return
	}

	var settings map[string]any
	if err := json.Unmarshal(stripJSONComments(text), &settings); err != nil {
		fmt.Println("Could not parse VS Code settings.")
		return
	}

	rows := [][3]string{
		{"Codex TODO CodeLens", boolStatus(settings["chatgpt.commentCodeLensEnabled"], false), "chatgpt.commentCodeLensEnabled=false"},
		{"Codex startup", boolStatus(settings["chatgpt.openOnStartup"], false), "chatgpt.openOnStartup=false"},
		{"Git autofetch", boolStatus(settings["git.autofetch"], false), "git.autofetch=false"},
		{"GitLens AI", boolStatus(settings["gitlens.ai.enabled"], false), "gitlens.ai.enabled=false"},
		{"Editor limit", boolStatus(settings["workbench.editor.limit.enabled"], true), "workbench.editor.limit.enabled=true"},
	}

	w := newTable()
	fmt.Fprintln(w, "SETTING\tSTATUS\tTARGET")
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\n", row[0], row[1], row[2])
	}
	w.Flush()
}

func boolStatus(value any, target bool) string {
	boolValue, ok := value.(bool)
	if !ok {
		return "MISSING"
	}
	if boolValue == target {
		return "OK"
	}
	return "TUNE"
}

func printExtensionList() {
	result := runCommand("code", "--list-extensions", "--show-versions")
	if !result.ok {
		fmt.Println(strings.TrimSpace(result.output))
		return
	}

	lines := nonEmptyLines(result.output)
	extensions := []string{}
	for _, line := range lines {
		if strings.Contains(line, "@") && !strings.Contains(line, "ERROR:") {
			extensions = append(extensions, line)
		}
	}

	fmt.Printf("%-14s %d installed\n", "Extensions", len(extensions))
	w := newTable()
	fmt.Fprintln(w, "EXTENSION\tVERSION")
	for _, extension := range extensions {
		name, version, _ := strings.Cut(extension, "@")
		fmt.Fprintf(w, "%s\t%s\n", name, version)
	}
	w.Flush()
}

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
	fmt.Fprintln(w, "STATUS\tSIZE\tITEM\tPATH")
	for index, size := range sizes {
		if limit > 0 && index >= limit {
			break
		}
		status := "OK"
		if !size.ok {
			status = "SKIP"
		} else if size.bytes >= 20*1024*1024*1024 {
			status = "REVIEW"
		} else if size.bytes >= 5*1024*1024*1024 {
			status = "WATCH"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", status, size.size, size.label, size.path)
	}
	w.Flush()
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

func printCommand(name string, args []string, allowFail bool, maxLines int) {
	result := runCommand(name, args...)
	if !result.ok && !allowFail {
		fmt.Println(strings.TrimSpace(result.output))
		return
	}

	output := strings.TrimSpace(result.output)
	if output == "" {
		return
	}
	fmt.Println(truncateLines(output, maxLines))
}

func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func runCommand(name string, args ...string) commandResult {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimRight(stdout.String()+stderr.String(), "\n")
	return commandResult{
		ok:     err == nil,
		output: output,
		err:    err,
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func removeKnownPath(fullPath string) {
	allowed := map[string]bool{
		expand("~/.gradle/caches"):        true,
		expand("~/.gradle/wrapper/dists"): true,
		vscodeWorkspaceStoragePath:        true,
	}

	if !allowed[fullPath] {
		fmt.Printf("Refusing to remove unknown path: %s\n", fullPath)
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		fmt.Printf("Could not remove %s: %s\n", fullPath, err)
		return
	}

	fmt.Printf("Removed: %s\n", fullPath)
}

func mergeBoolMap(existing any, additions map[string]any) map[string]any {
	merged := map[string]any{}

	if typed, ok := existing.(map[string]any); ok {
		for key, value := range typed {
			merged[key] = value
		}
	}

	keys := make([]string, 0, len(additions))
	for key := range additions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		merged[key] = additions[key]
	}
	return merged
}

func expand(rawPath string) string {
	if rawPath == "~" {
		return homeDir
	}
	if strings.HasPrefix(rawPath, "~/") {
		return filepath.Join(homeDir, rawPath[2:])
	}
	return rawPath
}

func title(text string) {
	fmt.Printf("\n%s\n%s\n", text, strings.Repeat("=", len(text)))
}

func section(title string) {
	fmt.Printf("\n== %s ==\n", title)
}

func nonEmptyLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func truncateLines(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

func timestamp() string {
	return time.Now().Format("20060102150405")
}

func meter(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(percent/100*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "]"
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	for _, suffix := range units {
		value /= unit
		if value < unit {
			if value >= 100 {
				return fmt.Sprintf("%.0f %s", value, suffix)
			}
			if value >= 10 {
				return fmt.Sprintf("%.1f %s", value, suffix)
			}
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f EB", value/unit)
}

func formatMegabytes(value float64) string {
	return formatBytes(uint64(value * 1024 * 1024))
}

func parseHumanSize(value string) uint64 {
	value = strings.TrimSpace(strings.Trim(value, "()"))
	if value == "" || value == "0B" || value == "0" {
		return 0
	}

	numberPart := ""
	unitPart := ""
	for _, char := range value {
		if (char >= '0' && char <= '9') || char == '.' {
			numberPart += string(char)
		} else {
			unitPart += string(char)
		}
	}

	number, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(strings.TrimSpace(unitPart)) {
	case "B", "":
		return uint64(number)
	case "K", "KB", "KIB":
		return uint64(number * 1024)
	case "M", "MB", "MIB":
		return uint64(number * 1024 * 1024)
	case "G", "GB", "GIB":
		return uint64(number * 1024 * 1024 * 1024)
	case "T", "TB", "TIB":
		return uint64(number * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func parseMemoryToBytes(value string) uint64 {
	return parseHumanSize(value)
}

func mustHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return dir
}

func stripJSONComments(input []byte) []byte {
	output := make([]byte, 0, len(input))
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i := 0; i < len(input); i++ {
		char := input[i]
		var next byte
		if i+1 < len(input) {
			next = input[i+1]
		}

		if inLineComment {
			if char == '\n' {
				inLineComment = false
				output = append(output, char)
			}
			continue
		}

		if inBlockComment {
			if char == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if !inString && char == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}

		if !inString && char == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}

		output = append(output, char)

		if char == '"' && !escaped {
			inString = !inString
		}

		escaped = char == '\\' && !escaped
		if char != '\\' {
			escaped = false
		}
	}

	return output
}
