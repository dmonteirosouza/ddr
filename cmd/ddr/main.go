package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sizeEntry struct {
	label string
	path  string
}

type commandResult struct {
	ok     bool
	output string
	err    error
}

var (
	homeDir = mustHomeDir()

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
	section("Disk")
	printCommand("df", []string{"-h"}, false, 80)

	section("Memory")
	printCommand("sysctl", []string{"hw.memsize"}, true, 20)
	printCommand("sysctl", []string{"vm.swapusage"}, true, 20)
	printCommand("memory_pressure", nil, true, 80)

	section("Top Memory Processes")
	printTopProcesses(18)

	section("Docker")
	if commandExists("docker") {
		printCommand("docker", []string{"system", "df"}, true, 80)
	} else {
		fmt.Println("Docker CLI not found.")
	}

	section("Common Heavy Folders")
	printSizeTable(candidatePaths)

	fmt.Println("\nNext:")
	fmt.Println("  ddr clean                 show safe cleanup plan")
	fmt.Println("  ddr clean --safe --yes    run conservative cleanup")
	fmt.Println("  ddr vscode --apply        apply lighter VS Code settings with backup")
	return nil
}

func memory() error {
	section("Memory")
	printCommand("sysctl", []string{"hw.memsize"}, true, 20)
	printCommand("sysctl", []string{"vm.swapusage"}, true, 20)
	printCommand("vm_stat", nil, true, 80)
	printCommand("memory_pressure", nil, true, 80)

	section("Top Memory Processes")
	printTopProcesses(25)
	return nil
}

func dockerReport() error {
	if !commandExists("docker") {
		fmt.Println("Docker CLI not found.")
		return nil
	}

	section("Docker Usage")
	printCommand("docker", []string{"system", "df"}, true, 80)

	section("Docker Detail")
	printCommand("docker", []string{"system", "df", "-v"}, true, 120)

	fmt.Println("\nCleanup:")
	fmt.Println("  ddr clean --safe --yes      prune build cache only, plus npm/Gradle caches")
	fmt.Println("  ddr clean --all-safe --yes  also prune stopped containers and unused images")
	fmt.Println("\nDocker volumes are intentionally preserved.")
	return nil
}

func vscode(flags map[string]bool) error {
	section("VS Code Storage")
	printSizeTable([]sizeEntry{
		{"workspaceStorage", "~/Library/Application Support/Code/User/workspaceStorage"},
		{"globalStorage", "~/Library/Application Support/Code/User/globalStorage"},
		{"History", "~/Library/Application Support/Code/User/History"},
		{"extensions", "~/.vscode/extensions"},
	})

	section("Installed Extensions")
	if commandExists("code") {
		printCommand("code", []string{"--list-extensions", "--show-versions"}, true, 100)
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

	for _, action := range actions {
		fmt.Printf("- %s: %s\n", action[0], action[1])
	}

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

func section(title string) {
	fmt.Printf("\n== %s ==\n", title)
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
