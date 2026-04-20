package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var statsDays int

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "check ur dev stats 📊",
	Long:  "Shows your command count, grind time, streak, top commands, and top projects.",
	RunE:  runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVarP(&statsDays, "days", "d", 7, "how many days to look back")
}

func runStats(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	stats, err := database.GetStats(statsDays)
	if err != nil {
		return fmt.Errorf("loading stats: %w", err)
	}
	topCmds, err := database.GetTopCommands(statsDays, 6)
	if err != nil {
		return err
	}
	topProjects, err := database.GetTopProjects(statsDays, 5)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Title.Render(fmt.Sprintf("📊  ur dev pulse  ·  last %d days", statsDays)))
	fmt.Println()
	fmt.Println(renderOverview(stats))

	if len(topCmds) > 0 {
		fmt.Println(renderBarSection("💻  top commands", topCmds, false))
	}
	if len(topProjects) > 0 {
		fmt.Println(renderBarSection("📁  top projects", topProjects, true))
	}
	fmt.Println()
	return nil
}

func renderOverview(s *db.Stats) string {
	streak := fmt.Sprintf("%d day streak", s.StreakDays)
	switch {
	case s.StreakDays == 0:
		streak = "no streak yet 💀"
	case s.StreakDays >= 30:
		streak += " 🏆"
	case s.StreakDays >= 7:
		streak += " 🔥"
	}
	rows := []string{
		statRow("🔥  streak", streak),
		statRow("⚡  commands", ui.FormatNumber(s.TotalCommands)),
		statRow("⏰  grind time", ui.FormatDuration(s.TotalTimeMS)),
		statRow("✅  success rate", fmt.Sprintf("%.1f%%", s.SuccessRate)),
	}
	return ui.Box.Render(strings.Join(rows, "\n"))
}

func statRow(label, value string) string {
	return ui.Label.Render(label) + ui.Value.Render(value)
}

// renderBarSection renders a labelled list of bar chart rows WITHOUT a box wrapper.
// We avoid Box here because block characters (█ ░) can render as double-width in
// some terminals, making lipgloss miscalculate the box border position.
//
// 🧠 Go Lesson #49: When you hit a third-party rendering quirk, the pragmatic fix
// is often to remove the abstraction causing it rather than fighting the library.
func renderBarSection(title string, entries []db.TopEntry, byTime bool) string {
	var maxVal float64
	for _, e := range entries {
		v := float64(e.Count)
		if byTime {
			v = float64(e.MS)
		}
		if v > maxVal {
			maxVal = v
		}
	}

	lines := []string{
		ui.Accent.Render("  " + title),
		"",
	}

	nameW := lipgloss.NewStyle().Width(16)
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
		name := nameW.Render(ui.Truncate(e.Name, 15))
		bar := ui.ProgressBar(val, maxVal, 12)
		lines = append(lines, "  "+name+bar+ui.Muted.Render("  "+suffix))
	}
	return strings.Join(lines, "\n")
}
