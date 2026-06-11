package app

import "fmt"

func clean(flags map[string]bool) error {
	safe := flags["--safe"] || flags["--all-safe"]
	allSafe := flags["--all-safe"]
	vscodeStorage := flags["--vscode-storage"]
	yes := flags["--yes"] || flags["-y"]

	title("ddr clean")

	section("Plano de limpeza")

	if !safe && !allSafe && !vscodeStorage {
		fmt.Println("Nenhuma limpeza selecionada.")
		fmt.Println("\nTente:")
		fmt.Println("  ddr clean --safe")
		fmt.Println("  ddr clean --safe --yes")
		fmt.Println("  ddr clean --all-safe --yes")
		fmt.Println("  ddr clean --vscode-storage --yes")
		return nil
	}

	actions := [][2]string{}
	if safe {
		actions = append(actions, [2]string{"Cache de build do Docker", "docker builder prune -af"})
		actions = append(actions, [2]string{"Cache do npm", "npm cache clean --force"})
		actions = append(actions, [2]string{"Caches do Gradle", "remove ~/.gradle/caches e ~/.gradle/wrapper/dists"})
	}
	if allSafe {
		actions = append(actions, [2]string{"Objetos Docker sem uso", "docker system prune -af (preserva volumes)"})
	}
	if vscodeStorage {
		actions = append(actions, [2]string{"VS Code workspaceStorage", "remove estado/cache do workspace; feche o VS Code antes"})
	}

	w := newTable()
	fmt.Fprintln(w, "ITEM\tACAO")
	for _, action := range actions {
		fmt.Fprintf(w, "%s\t%s\n", action[0], action[1])
	}
	w.Flush()

	fmt.Println("\nVolumes do Docker nao sao removidos.")

	if !yes {
		fmt.Println("\nSimulacao apenas. Rode de novo com --yes para executar.")
		return nil
	}

	if safe {
		if commandExists("docker") {
			section("Limpando cache de build do Docker")
			printCommand("docker", []string{"builder", "prune", "-af"}, true, 200)
		}

		if commandExists("npm") {
			section("Limpando cache do npm")
			printCommand("npm", []string{"cache", "clean", "--force"}, true, 80)
		}

		section("Limpando caches do Gradle")
		removeKnownPath(expand("~/.gradle/caches"))
		removeKnownPath(expand("~/.gradle/wrapper/dists"))
	}

	if allSafe && commandExists("docker") {
		section("Limpando objetos Docker sem uso")
		printCommand("docker", []string{"system", "prune", "-af"}, true, 300)
	}

	if vscodeStorage {
		section("Limpando workspaceStorage do VS Code")
		removeKnownPath(vscodeWorkspaceStoragePath)
	}

	section("Depois")
	printCommand("df", []string{"-h"}, true, 80)
	return nil
}
