package app

import (
	"fmt"
	"strconv"
	"strings"
)

func printDiskSummary() {
	info, ok := readDiskInfo("/System/Volumes/Data")
	if !ok {
		info, ok = readDiskInfo("/")
	}
	if !ok {
		printCommand("df", []string{"-h"}, true, 80)
		return
	}

	status := "OK"
	if info.freeBytes < 20*1024*1024*1024 || info.usedPct >= 90 {
		status = "CRITICO"
	} else if info.freeBytes < 50*1024*1024*1024 || info.usedPct >= 80 {
		status = "ATENCAO"
	}

	fmt.Printf("%-14s %s\n", "Status", statusBadge(status))
	fmt.Printf("%-14s %s\n", "Volume", info.mount)
	fmt.Printf("%-14s %s / %s usados\n", "Uso", formatBytes(info.usedBytes), formatBytes(info.totalBytes))
	fmt.Printf("%-14s %s\n", "Livre", formatBytes(info.freeBytes))
	fmt.Printf("%-14s %s %.0f%%\n", "Medidor", meter(info.usedPct, 24), info.usedPct)

	switch status {
	case "CRITICO":
		tip("Pouco espaco livre. Rode 'ddr clean --safe --yes' e revise Docker, Downloads e caches grandes.")
	case "ATENCAO":
		tip("Espaco ficando apertado. Limpe caches antes de builds grandes ou projetos com Docker.")
	default:
		tip("Disco com folga para trabalhar.")
	}
}

func readDiskInfo(mount string) (diskInfo, bool) {
	result := runCommand("df", "-kP", mount)
	if !result.ok {
		return diskInfo{}, false
	}

	lines := nonEmptyLines(result.output)
	if len(lines) < 2 {
		return diskInfo{}, false
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 6 {
		return diskInfo{}, false
	}

	totalKB, err1 := strconv.ParseUint(fields[1], 10, 64)
	usedKB, err2 := strconv.ParseUint(fields[2], 10, 64)
	freeKB, err3 := strconv.ParseUint(fields[3], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || totalKB == 0 {
		return diskInfo{}, false
	}

	mountedOn := strings.Join(fields[5:], " ")
	usedPct := float64(usedKB) / float64(totalKB) * 100
	return diskInfo{
		mount:      mountedOn,
		totalBytes: totalKB * 1024,
		usedBytes:  usedKB * 1024,
		freeBytes:  freeKB * 1024,
		usedPct:    usedPct,
	}, true
}
