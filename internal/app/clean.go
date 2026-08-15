package app

import (
	"fmt"
	"os"
	"strings"
)

type cleanTarget struct {
	flag            string
	label           string
	action          string
	selectedBySafe  bool
	selectedByAll   bool
	selectedByCache bool
	estimate        func() cleanEstimate
	run             func()
}

type cleanEstimate struct {
	bytes uint64
	ok    bool
	note  string
}

var cleanTargets = []cleanTarget{
	{
		flag:            "--docker-build",
		label:           "Cache de build do Docker",
		action:          "docker builder prune -af",
		selectedBySafe:  true,
		selectedByAll:   true,
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimateDockerReclaimable("Build Cache")
		},
		run: func() {
			runCleanCommand("docker", []string{"builder", "prune", "-af"}, 200)
		},
	},
	{
		flag:          "--docker-system",
		label:         "Objetos Docker sem uso",
		action:        "docker system prune -af (preserva volumes)",
		selectedByAll: true,
		estimate:      estimateDockerSystemPrune,
		run: func() {
			runCleanCommand("docker", []string{"system", "prune", "-af"}, 300)
		},
	},
	{
		flag:            "--npm",
		label:           "Cache do npm",
		action:          "npm cache clean --force",
		selectedBySafe:  true,
		selectedByAll:   true,
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimateCommandPath("npm", "config", "get", "cache")
		},
		run: func() {
			runCleanCommand("npm", []string{"cache", "clean", "--force"}, 80)
		},
	},
	{
		flag:            "--yarn",
		label:           "Cache do Yarn",
		action:          "yarn cache clean",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimateCommandPath("yarn", "cache", "dir")
		},
		run: func() {
			runCleanCommand("yarn", []string{"cache", "clean"}, 80)
		},
	},
	{
		flag:            "--pnpm",
		label:           "Store do pnpm",
		action:          "pnpm store prune",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimateCommandPath("pnpm", "store", "path")
		},
		run: func() {
			runCleanCommand("pnpm", []string{"store", "prune"}, 80)
		},
	},
	{
		flag:            "--gradle",
		label:           "Caches do Gradle",
		action:          "remove ~/.gradle/caches e ~/.gradle/wrapper/dists",
		selectedBySafe:  true,
		selectedByAll:   true,
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimatePaths("~/.gradle/caches", "~/.gradle/wrapper/dists")
		},
		run: func() {
			removeKnownPath(expand("~/.gradle/caches"))
			removeKnownPath(expand("~/.gradle/wrapper/dists"))
		},
	},
	{
		flag:            "--go-build",
		label:           "Cache de build do Go",
		action:          "go clean -cache",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimateGoEnv("GOCACHE")
		},
		run: func() {
			runCleanCommand("go", []string{"clean", "-cache"}, 80)
		},
	},
	{
		flag:            "--go-mod",
		label:           "Cache de modulos do Go",
		action:          "go clean -modcache",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimateGoEnv("GOMODCACHE")
		},
		run: func() {
			runCleanCommand("go", []string{"clean", "-modcache"}, 80)
		},
	},
	{
		flag:            "--pip",
		label:           "Cache do pip",
		action:          "python3 -m pip cache purge",
		selectedByCache: true,
		estimate:        estimatePipCache,
		run:             cleanPipCache,
	},
	{
		flag:            "--pub",
		label:           "Cache do Dart/Flutter Pub",
		action:          "dart pub cache clean --force",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			if pubCache := os.Getenv("PUB_CACHE"); pubCache != "" {
				return estimatePaths(pubCache)
			}
			return estimatePaths("~/.pub-cache")
		},
		run: func() {
			runCleanCommand("dart", []string{"pub", "cache", "clean", "--force"}, 120)
		},
	},
	{
		flag:            "--cocoapods",
		label:           "Cache do CocoaPods",
		action:          "pod cache clean --all",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimatePaths("~/Library/Caches/CocoaPods")
		},
		run: func() {
			runCleanCommand("pod", []string{"cache", "clean", "--all"}, 120)
		},
	},
	{
		flag:            "--xcode-derived-data",
		label:           "Xcode DerivedData",
		action:          "remove ~/Library/Developer/Xcode/DerivedData",
		selectedByCache: true,
		estimate: func() cleanEstimate {
			return estimatePaths("~/Library/Developer/Xcode/DerivedData")
		},
		run: func() {
			removeKnownPath(expand("~/Library/Developer/Xcode/DerivedData"))
		},
	},
	{
		flag:   "--vscode-storage",
		label:  "VS Code workspaceStorage",
		action: "remove estado/cache do workspace; feche o VS Code antes",
		estimate: func() cleanEstimate {
			return estimatePaths(vscodeWorkspaceStoragePath)
		},
		run: func() {
			removeKnownPath(vscodeWorkspaceStoragePath)
		},
	},
}

func clean(flags map[string]bool) error {
	yes := flags["--yes"] || flags["-y"]

	title("ddr clean")

	section("Plano de limpeza")

	targets := selectedCleanTargets(flags)
	if len(targets) == 0 {
		fmt.Println("Nenhuma limpeza selecionada.")
		fmt.Println("\nTente:")
		fmt.Println("  ddr clean --safe")
		fmt.Println("  ddr clean --npm --gradle --yes")
		fmt.Println("  ddr clean --all-caches --yes")
		fmt.Println("  ddr clean --vscode-storage --yes")
		fmt.Println("\nOpcoes disponiveis:")
		printCleanOptions()
		return nil
	}

	totalKnown := uint64(0)
	w := newTable()
	fmt.Fprintln(w, "FLAG\tITEM\tESTIMADO\tACAO")
	for _, target := range targets {
		estimate := estimateCleanTarget(target)
		if estimate.ok {
			totalKnown += estimate.bytes
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", target.flag, target.label, estimate.String(), target.action)
	}
	w.Flush()

	fmt.Printf("\nEstimativa conhecida: %s\n", formatBytes(totalKnown))
	fmt.Println("Alguns comandos podem liberar menos que o tamanho do cache medido.")
	fmt.Println("\nVolumes do Docker nao sao removidos.")

	if !yes {
		fmt.Println("\nSimulacao apenas. Rode de novo com --yes para executar.")
		return nil
	}

	for _, target := range targets {
		section("Limpando " + target.label)
		target.run()
	}

	section("Depois")
	printCommand("df", []string{"-h"}, true, 80)
	return nil
}

func selectedCleanTargets(flags map[string]bool) []cleanTarget {
	safe := flags["--safe"] || flags["--all-safe"]
	allSafe := flags["--all-safe"]
	allCaches := flags["--all-caches"]

	selected := []cleanTarget{}
	seen := map[string]bool{}
	for _, target := range cleanTargets {
		if flags[target.flag] ||
			(safe && target.selectedBySafe) ||
			(allSafe && target.selectedByAll) ||
			(allCaches && target.selectedByCache) {
			if !seen[target.flag] {
				selected = append(selected, target)
				seen[target.flag] = true
			}
		}
	}
	return selected
}

func printCleanOptions() {
	w := newTable()
	fmt.Fprintln(w, "FLAG\tITEM")
	for _, target := range cleanTargets {
		fmt.Fprintf(w, "%s\t%s\n", target.flag, target.label)
	}
	fmt.Fprintln(w, "--safe\tDocker build + npm + Gradle")
	fmt.Fprintln(w, "--all-safe\t--safe + objetos Docker sem uso")
	fmt.Fprintln(w, "--all-caches\tTodos os caches de ferramentas listados, sem VS Code storage nem Docker system")
	w.Flush()
}

func runCleanCommand(name string, args []string, maxLines int) {
	if !commandExists(name) {
		fmt.Printf("Comando %q nao encontrado; pulando.\n", name)
		return
	}
	printCommand(name, args, true, maxLines)
}

func cleanPipCache() {
	if commandExists("python3") {
		runCleanCommand("python3", []string{"-m", "pip", "cache", "purge"}, 80)
		return
	}
	runCleanCommand("pip3", []string{"cache", "purge"}, 80)
}

func estimateCleanTarget(target cleanTarget) cleanEstimate {
	if target.estimate == nil {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}
	return target.estimate()
}

func (estimate cleanEstimate) String() string {
	if estimate.ok {
		return formatBytes(estimate.bytes)
	}
	if estimate.note != "" {
		return estimate.note
	}
	return "desconhecido"
}

func estimatePaths(paths ...string) cleanEstimate {
	total := uint64(0)
	for _, rawPath := range paths {
		fullPath := expand(rawPath)
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cleanEstimate{ok: false, note: "sem acesso"}
		}

		size := duSize(fullPath)
		if !size.ok {
			return cleanEstimate{ok: false, note: "desconhecido"}
		}
		total += size.bytes
	}
	return cleanEstimate{bytes: total, ok: true}
}

func estimateCommandPath(name string, args ...string) cleanEstimate {
	if !commandExists(name) {
		return cleanEstimate{ok: false, note: "indisponivel"}
	}

	result := runCommand(name, args...)
	if !result.ok {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}

	path := lastNonEmptyLine(result.output)
	if path == "" || strings.EqualFold(path, "undefined") || strings.EqualFold(path, "null") {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}
	return estimatePaths(path)
}

func estimateGoEnv(key string) cleanEstimate {
	if !commandExists("go") {
		return cleanEstimate{ok: false, note: "indisponivel"}
	}

	result := runCommand("go", "env", key)
	if !result.ok {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}

	path := lastNonEmptyLine(result.output)
	if path == "" {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}
	return estimatePaths(path)
}

func estimatePipCache() cleanEstimate {
	if commandExists("python3") {
		return estimateCommandPath("python3", "-m", "pip", "cache", "dir")
	}
	return estimateCommandPath("pip3", "cache", "dir")
}

func estimateDockerReclaimable(kind string) cleanEstimate {
	if !commandExists("docker") {
		return cleanEstimate{ok: false, note: "indisponivel"}
	}

	result := runCommand("docker", "system", "df")
	if !result.ok {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}

	for _, row := range parseDockerSystemDF(result.output) {
		if row.kind == kind {
			return cleanEstimate{bytes: row.reclaimBytes, ok: true}
		}
	}
	return cleanEstimate{bytes: 0, ok: true}
}

func estimateDockerSystemPrune() cleanEstimate {
	if !commandExists("docker") {
		return cleanEstimate{ok: false, note: "indisponivel"}
	}

	result := runCommand("docker", "system", "df")
	if !result.ok {
		return cleanEstimate{ok: false, note: "desconhecido"}
	}

	total := uint64(0)
	for _, row := range parseDockerSystemDF(result.output) {
		if row.kind == "Build Cache" || row.kind == "Local Volumes" {
			continue
		}
		total += row.reclaimBytes
	}
	return cleanEstimate{bytes: total, ok: true}
}

func lastNonEmptyLine(text string) string {
	lines := nonEmptyLines(text)
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
