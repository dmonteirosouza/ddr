package app

import (
	"text/tabwriter"

	"ddr/internal/ui"
)

func title(text string) {
	ui.Title(text)
}

func section(text string) {
	ui.Section(text)
}

func newTable() *tabwriter.Writer {
	return ui.NewTable()
}

func meter(percent float64, width int) string {
	return ui.Meter(percent, width)
}

func formatBytes(bytes uint64) string {
	return ui.FormatBytes(bytes)
}

func formatMegabytes(value float64) string {
	return ui.FormatMegabytes(value)
}

func truncateLines(text string, maxLines int) string {
	return ui.TruncateLines(text, maxLines)
}

func statusBadge(status string) string {
	return ui.Status(status)
}

func tip(text string) {
	ui.Tip(text)
}
