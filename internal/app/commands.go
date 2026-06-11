package app

import "fmt"

func scan() error {
	title("ddr - painel do Mac")
	fmt.Println("Leitura rapida para entender disco, memoria, Docker, VS Code e caches pesados.")
	fmt.Println("Estados: OK = tranquilo, ATENCAO = vale cuidar, CRITICO = agir agora.")

	section("Disco")
	printDiskSummary()

	section("Memoria")
	printMemorySummary()

	section("Quem esta usando memoria")
	printProcessSummary(12)

	section("Docker")
	printDockerSummary(false)

	section("Pastas que costumam pesar")
	printSortedSizeTable(candidatePaths, 12)

	section("Proximos passos")
	fmt.Println("  ddr clean                 mostra o plano de limpeza, sem apagar nada")
	fmt.Println("  ddr clean --safe --yes    executa limpeza conservadora")
	fmt.Println("  ddr vscode --apply        aplica ajustes leves no VS Code com backup")
	fmt.Println("  ddr chrome                checklist para quem usa muitas abas")
	return nil
}

func memory() error {
	title("ddr memory")

	section("Memoria")
	printMemorySummary()

	section("Quem esta usando memoria")
	printProcessSummary(25)
	return nil
}

func dockerReport() error {
	if !commandExists("docker") {
		fmt.Println("Docker CLI nao encontrado.")
		return nil
	}

	title("ddr docker")

	section("Uso do Docker")
	printDockerSummary(true)

	fmt.Println("\nLimpeza:")
	fmt.Println("  ddr clean --safe --yes      limpa cache de build, npm e Gradle")
	fmt.Println("  ddr clean --all-safe --yes  tambem limpa containers parados e imagens sem uso")
	fmt.Println("\nVolumes do Docker sao preservados de proposito.")
	return nil
}

func vscode(flags map[string]bool) error {
	title("ddr vscode")

	section("Armazenamento do VS Code")
	printSortedSizeTable([]sizeEntry{
		{"workspaceStorage", "~/Library/Application Support/Code/User/workspaceStorage"},
		{"globalStorage", "~/Library/Application Support/Code/User/globalStorage"},
		{"History", "~/Library/Application Support/Code/User/History"},
		{"extensions", "~/.vscode/extensions"},
	}, 10)

	section("Ajustes")
	printVscodeSettingsStatus()

	section("Extensoes instaladas")
	if commandExists("code") {
		printExtensionList()
	} else {
		fmt.Println("CLI 'code' nao encontrado. No VS Code, rode: Shell Command: Install 'code' command in PATH")
	}

	if !flags["--apply"] {
		fmt.Println("\nPara aplicar ajustes mais leves no VS Code/Codex:")
		fmt.Println("  ddr vscode --apply")
		return nil
	}

	return applyVscodeSettings()
}
