package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func applyVscodeSettings() error {
	section("Aplicando ajustes no VS Code")

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
		return fmt.Errorf("nao consegui ler o settings.json do VS Code: %w", err)
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

	fmt.Printf("Atualizado: %s\n", vscodeSettingsPath)
	fmt.Printf("Backup:  %s\n", backupPath)
	fmt.Println("\nRecarregue o VS Code: Cmd+Shift+P -> Developer: Reload Window")
	return nil
}
