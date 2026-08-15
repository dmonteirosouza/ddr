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
  ddr clean --npm --gradle --yes
  ddr clean --go-build --pip --yes
  ddr clean --all-caches --yes
  ddr clean --all-safe --yes
  ddr clean --vscode-storage --yes
  ddr chrome
  ddr version

Limpeza:
  --docker-build        cache de build do Docker
  --docker-system       containers parados, redes e imagens Docker sem uso
  --npm                 cache do npm
  --yarn                cache do Yarn
  --pnpm                store do pnpm
  --gradle              caches do Gradle
  --go-build            cache de build do Go
  --go-mod              cache de modulos do Go
  --pip                 cache do pip
  --pub                 cache do Dart/Flutter Pub
  --cocoapods           cache do CocoaPods
  --xcode-derived-data  Xcode DerivedData
  --vscode-storage      remove workspaceStorage do VS Code; feche o VS Code antes
  --safe                Docker build + npm + Gradle
  --all-safe            --safe + containers parados, redes e imagens sem uso
  --all-caches          todos os caches de ferramentas, sem VS Code storage nem Docker system
  --yes                 necessario para apagar qualquer coisa

Notas:
  Volumes do Docker nunca sao apagados automaticamente.
  Mudancas no VS Code sempre criam backup antes de salvar.`)
}
