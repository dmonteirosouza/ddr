package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

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
