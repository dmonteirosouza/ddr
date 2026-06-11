package app

import (
	"fmt"
	"strconv"
	"strings"
)

func printMemorySummary() {
	info := readMemoryInfo()

	if info.ramBytes > 0 {
		fmt.Printf("%-14s %s\n", "RAM", formatBytes(info.ramBytes))
	}

	memoryStatus := "OK"
	if info.freePctOK {
		if info.freePct < 10 {
			memoryStatus = "CRITICO"
		} else if info.freePct < 20 {
			memoryStatus = "ATENCAO"
		}
		usedPct := 100 - float64(info.freePct)
		fmt.Printf("%-14s %s (%d%% livre)\n", "Pressao", statusBadge(memoryStatus), info.freePct)
		fmt.Printf("%-14s %s %.0f%%\n", "Medidor", meter(usedPct, 24), usedPct)
	}

	if info.compressorSize != "" {
		fmt.Printf("%-14s %s\n", "Comprimido", info.compressorSize)
	}

	swapStatus := "OK"
	if info.swapOK {
		swapPct := 0.0
		if info.swap.totalMB > 0 {
			swapPct = info.swap.usedMB / info.swap.totalMB * 100
		}
		if swapPct >= 85 {
			swapStatus = "CRITICO"
		} else if swapPct >= 60 {
			swapStatus = "ATENCAO"
		}

		fmt.Printf("%-14s %s (%s / %s)\n", "Swap", statusBadge(swapStatus), formatMegabytes(info.swap.usedMB), formatMegabytes(info.swap.totalMB))
		fmt.Printf("%-14s %s %.0f%%\n", "Swap medidor", meter(swapPct, 24), swapPct)
	}

	if !info.freePctOK && !info.swapOK {
		fmt.Println("Nao consegui ler o resumo de memoria. Tente rodar pelo Terminal.")
		return
	}

	if memoryStatus == "CRITICO" || swapStatus == "CRITICO" {
		tip("Memoria muito pressionada. Reduza Docker, feche abas pesadas e reinicie VS Code/Codex se estiver acumulando processos.")
	} else if memoryStatus == "ATENCAO" || swapStatus == "ATENCAO" {
		tip("Memoria com alguma pressao. Vale fechar abas/app de projetos antigos antes de abrir builds grandes.")
	} else {
		tip("Memoria em estado confortavel agora.")
	}
}

func readMemoryInfo() memoryInfo {
	info := memoryInfo{}

	if result := runCommand("sysctl", "hw.memsize"); result.ok {
		fields := strings.Fields(result.output)
		if len(fields) >= 2 {
			if value, err := strconv.ParseUint(fields[len(fields)-1], 10, 64); err == nil {
				info.ramBytes = value
			}
		}
	}

	if result := runCommand("sysctl", "vm.swapusage"); result.ok {
		if match := swapRegexp.FindStringSubmatch(result.output); len(match) == 4 {
			total, _ := strconv.ParseFloat(match[1], 64)
			used, _ := strconv.ParseFloat(match[2], 64)
			free, _ := strconv.ParseFloat(match[3], 64)
			info.swap = swapUsage{totalMB: total, usedMB: used, freeMB: free}
			info.swapOK = true
		}
	}

	if result := runCommand("memory_pressure"); result.ok {
		if match := memoryFreeRegexp.FindStringSubmatch(result.output); len(match) == 2 {
			if value, err := strconv.Atoi(match[1]); err == nil {
				info.freePct = value
				info.freePctOK = true
			}
		}
		if match := compressorRegexp.FindStringSubmatch(result.output); len(match) == 2 {
			if pages, err := strconv.ParseUint(match[1], 10, 64); err == nil {
				info.compressorSize = formatBytes(pages * 16384)
			}
		}
	}

	return info
}
