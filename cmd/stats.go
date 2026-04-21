package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var statsDays int

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "view your activity stats 📊",
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
	fmt.Println(ui.Title.Render(fmt.Sprintf("📊  Your dev pulse  ·  last %d days", statsDays)))
	fmt.Println()
	fmt.Println(renderOverview(stats))

	if len(topCmds) > 0 {
		fmt.Println(renderBarSection(fmt.Sprintf("💻  top %d commands", len(topCmds)), topCmds, false))
		fmt.Println("  " + ui.Muted.Render("run `pulse history` to see every command in full"))
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
	cmdValue := ui.FormatNumber(s.TotalCommands)
	if s.NoiseCommands > 0 {
		cmdValue += ui.Muted.Render(fmt.Sprintf("  ·  +%s noise", ui.FormatNumber(s.NoiseCommands)))
	}
	rows := []string{
		statRow("🔥  streak", streak),
		statRowRaw("⚡  commands", cmdValue),
		statRow("⏰  grind time", ui.FormatDuration(s.TotalTimeMS)),
		statRow("✅  success rate", fmt.Sprintf("%.1f%%", s.SuccessRate)),
	}
	return ui.Box.Render(strings.Join(rows, "\n"))
}

func statRow(label, value string) string {
	return ui.Label.Render(label) + ui.Value.Render(value)
}

// statRowRaw renders a row where value is already styled.
func statRowRaw(label, value string) string {
	return ui.Label.Render(label) + value
}

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
