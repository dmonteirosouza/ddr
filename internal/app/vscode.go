package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func printVscodeSettingsStatus() {
	text, err := os.ReadFile(vscodeSettingsPath)
	if err != nil {
		fmt.Println("Nao encontrei settings.json do VS Code.")
		return
	}

	var settings map[string]any
	if err := json.Unmarshal(stripJSONComments(text), &settings); err != nil {
		fmt.Println("Nao consegui ler o settings.json do VS Code.")
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
	fmt.Fprintln(w, "AJUSTE\tSTATUS\tALVO")
	pending := 0
	for _, row := range rows {
		if row[1] != "OK" {
			pending++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", row[0], statusBadge(row[1]), row[2])
	}
	w.Flush()

	if pending > 0 {
		tip("'ddr vscode --apply' aplica os ajustes pendentes e cria backup antes.")
	} else {
		tip("VS Code ja esta com os ajustes principais para reduzir consumo.")
	}
}

func boolStatus(value any, target bool) string {
	boolValue, ok := value.(bool)
	if !ok {
		return "PENDENTE"
	}
	if boolValue == target {
		return "OK"
	}
	return "AJUSTAR"
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

	fmt.Printf("%-14s %d instaladas\n", "Extensoes", len(extensions))
	w := newTable()
	fmt.Fprintln(w, "EXTENSAO\tVERSAO")
	for _, extension := range extensions {
		name, version, _ := strings.Cut(extension, "@")
		fmt.Fprintf(w, "%s\t%s\n", name, version)
	}
	w.Flush()
}
