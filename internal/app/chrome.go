package app

import "fmt"

func chrome() {
	title("ddr chrome")

	section("Checklist do Chrome")
	fmt.Println("Abra: chrome://settings/performance")
	fmt.Println("- Ative o Memory Saver.")
	fmt.Println("- Use o modo mais agressivo, se aparecer para voce.")
	fmt.Println("- Deixe sempre ativos apenas sites realmente criticos.")
	fmt.Println("\nAbra: chrome://settings/system")
	fmt.Println("- Desative apps em segundo plano depois que o Chrome fechar.")
	fmt.Println("- Mantenha aceleracao de hardware ligada, a menos que o grafico fique instavel.")
	fmt.Println("\nAbra: chrome://extensions")
	fmt.Println("- Desative extensoes sem uso, principalmente IA, carteira, captura de tela e produtividade.")
	fmt.Println("\nUse Shift+Esc dentro do Chrome para ordenar abas/extensoes por memoria.")
}
