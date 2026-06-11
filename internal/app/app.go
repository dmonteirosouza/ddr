package app

import (
	"fmt"
	"strings"
)

var Version = "dev"

// Run executes the CLI command selected by args.
func Run(args []string) error {
	command, flags := parseArgs(args)

	switch command {
	case "", "scan":
		return scan()
	case "help", "--help", "-h":
		printHelp()
		return nil
	case "version", "--version", "-v":
		fmt.Printf("ddr %s\n", Version)
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
	fmt.Println(`ddr - doctor de desenvolvimento para macOS

Uso:
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
  ddr version

Limpeza:
  --safe           cache de build do Docker + npm + Gradle
  --all-safe       --safe + containers parados, redes e imagens sem uso
  --vscode-storage remove workspaceStorage do VS Code; feche o VS Code antes
  --yes            necessario para apagar qualquer coisa

Notas:
  Volumes do Docker nunca sao apagados automaticamente.
  Mudancas no VS Code sempre criam backup antes de salvar.`)
}
