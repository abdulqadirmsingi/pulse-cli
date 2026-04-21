package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
)

// View renders the full dashboard to a string.
// Bubble Tea calls this after every Update to redraw the screen.
//
// 🧠 Go Lesson #37: View() must be pure — given the same Model it must always
// return the same string.  Never read from global state or produce side effects
// inside View. All data comes from the Model.
func (m Model) View() string {
	if !m.ready {
		return "\n  " + ui.Muted.Render("loading ur stats...") + "\n"
	}
	return m.header() + "\n\n" + m.body() + "\n" + m.footer()
}

func (m Model) header() string {
	var parts []string
	for i, label := range tabLabels {
		if tab(i) == m.activeTab {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(ui.ColorCyan).Bold(true).
				Underline(true).Padding(0, 2).Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(ui.ColorGray).Padding(0, 2).Render(label))
		}
	}
	return "\n  " + ui.Title.Render("⚡ pulse") + "    " + strings.Join(parts, "")
}

func (m Model) footer() string {
	t := time.Now().Format("15:04:05")
	hint := "[tab / ←→] switch  [1-4] jump  [r] refresh  [q] quit"
	return ui.Muted.Render("  "+hint+"  ·  "+t) + "\n"
}

func (m Model) body() string {
	switch m.activeTab {
	case tabOverview:
		return m.viewOverview()
	case tabCommands:
		return m.viewBarChart("💻  top commands", m.data.cmds, false)
	case tabProjects:
		return m.viewBarChart("📁  projects by time", m.data.projects, true)
	case tabToday:
		return m.viewToday()
	}
	return ""
}

func (m Model) viewOverview() string {
	if m.data.stats == nil {
		return ui.Muted.Render("  no data yet — run some commands first") + "\n"
	}
	s := m.data.stats
	streak := fmt.Sprintf("%d days", s.StreakDays)
	switch {
	case s.StreakDays == 0:
		streak = "none yet 💀"
	case s.StreakDays >= 30:
		streak += " 🏆"
	case s.StreakDays >= 7:
		streak += " 🔥"
	}
	rows := []string{
		sRow("🔥  streak", streak),
		sRow("⚡  commands", ui.FormatNumber(s.TotalCommands)),
		sRow("⏰  grind time", ui.FormatDuration(s.TotalTimeMS)),
		sRow("✅  success rate", fmt.Sprintf("%.1f%%", s.SuccessRate)),
	}
	return ui.Box.Render(strings.Join(rows, "\n")) + "\n"
}

// viewBarChart renders commands or projects as a horizontal bar chart.
// byTime=true uses MS for bar length; false uses Count.
func (m Model) viewBarChart(title string, entries []db.TopEntry, byTime bool) string {
	if len(entries) == 0 {
		return ui.Muted.Render("  no data yet") + "\n"
	}
	var maxVal float64
	for _, e := range entries {
		if byTime {
			if float64(e.MS) > maxVal {
				maxVal = float64(e.MS)
			}
		} else {
			if float64(e.Count) > maxVal {
				maxVal = float64(e.Count)
			}
		}
	}
	lines := []string{ui.Accent.Render("  " + title), ""}
	for _, e := range entries {
		var val float64
		var suffix string
		if byTime {
			val = float64(e.MS)
			suffix = ui.FormatDuration(e.MS)
		} else {
			val = float64(e.Count)
			suffix = ui.FormatNumber(e.Count) + " runs"
		}
		name := lipgloss.NewStyle().Width(18).Render(e.Name)
		bar := ui.ProgressBar(val, maxVal, 18)
		lines = append(lines, "  "+name+bar+ui.Muted.Render("  "+suffix))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) viewToday() string {
	lines := []string{ui.Accent.Render("  📅  today by hour"), ""}
	lines = append(lines, HourlyChart(m.data.hourly, "  ")...)
	return strings.Join(lines, "\n") + "\n"
}

// HourlyChart renders a 24-row bar chart from HourlyBucket data.
// Exported so cmd/today.go can reuse it without duplicating rendering logic.
//
// 🧠 Go Lesson #38: Exporting a function from an internal package is fine —
// the restriction is on who can IMPORT the package, not what it exports.
// Any package within this module can import internal/tui.
func HourlyChart(hourly []db.HourlyBucket, indent string) []string {
	hourMap := map[int]int64{}
	var maxCount int64
	for _, b := range hourly {
		hourMap[b.Hour] = b.Count
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}
	if maxCount == 0 {
		return []string{indent + ui.Muted.Render("no activity logged today yet")}
	}
	var lines []string
	for h := 0; h < 24; h++ {
		count := hourMap[h]
		bar := ui.ProgressBar(float64(count), float64(maxCount), 22)
		label := ui.Muted.Render(fmt.Sprintf("%02d:00", h))
		suffix := ""
		if count > 0 {
			suffix = ui.Value.Render(fmt.Sprintf("  %d", count))
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s%s", indent, label, bar, suffix))
	}
	return lines
}

func sRow(label, value string) string {
	return ui.Label.Render(label) + ui.Value.Render(value)
}
