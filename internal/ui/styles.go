package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette — dark theme, neon accents. Looks sick in any modern terminal.
var (
	ColorCyan   = lipgloss.Color("#00D4FF")
	ColorPurple = lipgloss.Color("#9D4EDD")
	ColorGreen  = lipgloss.Color("#39FF14")
	ColorGold   = lipgloss.Color("#FFD700")
	ColorRed    = lipgloss.Color("#FF4757")
	ColorGray   = lipgloss.Color("#6B7280")
)

// Pre-built styles — create once, reuse everywhere.
var (
	Title = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)

	Label = lipgloss.NewStyle().Foreground(ColorGray).Width(22)
	Value = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	Bar   = lipgloss.NewStyle().Foreground(ColorPurple)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Padding(0, 2)

	Success = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	Err     = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	Muted   = lipgloss.NewStyle().Foreground(ColorGray)
	Accent  = lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)
)

// ProgressBar renders an ASCII progress bar for a given value/max ratio.
func ProgressBar(value, max float64, width int) string {
	if max == 0 || value == 0 {
		return Bar.Render(strings.Repeat("░", width))
	}
	ratio := value / max
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	return Bar.Render(strings.Repeat("█", filled) + strings.Repeat("░", width-filled))
}

// FormatDuration converts milliseconds into a human-readable "Xh Ym" string.
func FormatDuration(ms int64) string {
	if ms <= 0 {
		return "0s"
	}
	secs := ms / 1000
	mins := secs / 60
	hours := mins / 60

	switch {
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins%60)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs%60)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

// Truncate clips s to max visible characters, appending "…" if cut.
// Uses rune-safe slicing so multi-byte chars don't corrupt the output.
func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// FormatNumber adds comma separators to large integers (e.g. 1247 → "1,247").
func FormatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
