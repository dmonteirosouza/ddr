package app

import (
	"fmt"
	"strings"
)

func printDockerSummary(includeAdvice bool) {
	if !commandExists("docker") {
		fmt.Println("Docker CLI not found.")
		return
	}

	result := runCommand("docker", "system", "df")
	if !result.ok {
		fmt.Printf("%-14s %s\n", "Status", statusBadge("ATENCAO"))
		fmt.Println("Nao consegui acessar o Docker agora.")
		if output := strings.TrimSpace(result.output); output != "" {
			fmt.Println(output)
		}
		tip("Abra o Docker Desktop ou rode o comando no Terminal com acesso ao socket do Docker.")
		return
	}

	rows := parseDockerSystemDF(result.output)
	if len(rows) == 0 {
		fmt.Println(strings.TrimSpace(result.output))
		return
	}

	totalReclaim := uint64(0)
	for _, row := range rows {
		totalReclaim += row.reclaimBytes
	}

	status := "OK"
	if totalReclaim >= 30*1024*1024*1024 {
		status = "ATENCAO"
	}
	if totalReclaim >= 60*1024*1024*1024 {
		status = "CRITICO"
	}

	fmt.Printf("%-14s %s\n", "Status", statusBadge(status))
	fmt.Printf("%-14s %s que o Docker diz poder recuperar\n", "Recuperavel", formatBytes(totalReclaim))

	w := newTable()
	fmt.Fprintln(w, "TIPO\tTOTAL\tATIVO\tTAMANHO\tRECUPERAVEL")
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.kind, row.total, row.active, row.size, row.reclaimable)
	}
	w.Flush()

	switch status {
	case "CRITICO":
		tip("Docker esta segurando muito espaco recuperavel. Rode 'ddr clean --all-safe --yes' se nao precisar de imagens antigas.")
	case "ATENCAO":
		tip("Ha uma boa quantidade de cache/imagens recuperaveis. 'ddr clean --safe --yes' ja ajuda sem tocar em volumes.")
	default:
		tip("Docker nao parece ser o maior problema agora.")
	}

	if includeAdvice {
		fmt.Println("\nVolumes sao preservados pelo ddr. Revise manualmente antes de apagar bancos ou dados locais.")
	}
}

func parseDockerSystemDF(output string) []dockerRow {
	lines := nonEmptyLines(output)
	rows := []dockerRow{}

	for _, line := range lines {
		if strings.HasPrefix(line, "TYPE ") || strings.HasPrefix(line, "TYPE\t") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		kind := fields[0]
		offset := 1
		if len(fields) >= 6 && fields[0] == "Local" && fields[1] == "Volumes" {
			kind = "Local Volumes"
			offset = 2
		}
		if len(fields) >= 6 && fields[0] == "Build" && fields[1] == "Cache" {
			kind = "Build Cache"
			offset = 2
		}
		if len(fields) < offset+4 {
			continue
		}

		reclaimFieldIndex := offset + 3
		reclaimable := fields[reclaimFieldIndex]
		reclaimPctRaw := ""
		if len(fields) > reclaimFieldIndex+1 {
			reclaimable = strings.Join(fields[reclaimFieldIndex:], " ")
			reclaimPctRaw = fields[len(fields)-1]
		}

		rows = append(rows, dockerRow{
			kind:          kind,
			total:         fields[offset],
			active:        fields[offset+1],
			size:          fields[offset+2],
			reclaimable:   reclaimable,
			reclaimBytes:  parseHumanSize(fields[reclaimFieldIndex]),
			reclaimPctRaw: reclaimPctRaw,
		})
	}

	return rows
}
