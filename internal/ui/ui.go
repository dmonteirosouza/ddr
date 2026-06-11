package ui

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"
	dim   = "\033[2m"

	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[90m"
)

func Title(text string) {
	line := strings.Repeat("=", len(text)+4)
	fmt.Printf("\n%s\n", paint(blue+bold, line))
	fmt.Printf("%s  %s  %s\n", paint(blue+bold, "="), paint(bold, text), paint(blue+bold, "="))
	fmt.Printf("%s\n", paint(blue+bold, line))
}

func Section(text string) {
	fmt.Printf("\n%s %s\n", paint(cyan+bold, "=="), paint(bold, text))
}

func Tip(text string) {
	fmt.Printf("%s %s\n", paint(magenta+bold, "Dica:"), text)
}

func Status(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "OK":
		return paint(green+bold, "OK")
	case "ATENCAO", "AVISO", "WARN", "WATCH", "REVIEW", "REVISAR", "OLHAR", "TUNE", "AJUSTAR", "MISSING", "PENDENTE":
		return paint(yellow+bold, normalized)
	case "CRITICO", "CRITICAL", "ALTO":
		return paint(red+bold, normalized)
	case "SKIP", "PULAR":
		return paint(gray, normalized)
	default:
		return paint(dim, normalized)
	}
}

func Meter(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent/100*float64(width) + 0.5)
	if filled > width {
		filled = width
	}

	color := green
	if percent >= 85 {
		color = red
	} else if percent >= 65 {
		color = yellow
	}

	bar := strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
	return "[" + paint(color, bar) + "]"
}

func NewTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	for _, suffix := range units {
		value /= unit
		if value < unit {
			if value >= 100 {
				return fmt.Sprintf("%.0f %s", value, suffix)
			}
			if value >= 10 {
				return fmt.Sprintf("%.1f %s", value, suffix)
			}
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f EB", value/unit)
}

func FormatMegabytes(value float64) string {
	return FormatBytes(uint64(value * 1024 * 1024))
}

func TruncateLines(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

func paint(color string, text string) string {
	if !colorsEnabled() {
		return text
	}
	return color + text + reset
}

func colorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}
