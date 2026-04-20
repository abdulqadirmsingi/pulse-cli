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
	Short: "check ur dev stats no cap 📊",
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

	// Fetch all the data we need in one pass.
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

	// Render everything.
	fmt.Println()
	fmt.Println(renderStatsHeader(statsDays))
	fmt.Println(renderOverview(stats))
	if len(topCmds) > 0 {
		fmt.Println(renderTopCmds(topCmds))
	}
	if len(topProjects) > 0 {
		fmt.Println(renderTopProjects(topProjects))
	}
	fmt.Println()
	return nil
}

// renderStatsHeader prints the title line with the time window.
func renderStatsHeader(days int) string {
	return ui.Title.Render(fmt.Sprintf("📊  ur dev pulse  ·  last %d days", days))
}

// renderOverview renders the four key metrics in a box.
//
// 🧠 Go Lesson #30: strings.Join(slice, separator) is the idiomatic way to
// build a multi-line string from parts. It's cleaner than repeated +=.
func renderOverview(s *db.Stats) string {
	streakLabel := fmt.Sprintf("%d day streak", s.StreakDays)
	switch {
	case s.StreakDays == 0:
		streakLabel = "no streak yet 💀"
	case s.StreakDays >= 30:
		streakLabel += " 🏆 unreal"
	case s.StreakDays >= 7:
		streakLabel += " 🔥 on fire"
	}

	rows := []string{
		statRow("🔥  streak", streakLabel),
		statRow("⚡  commands", ui.FormatNumber(s.TotalCommands)),
		statRow("⏰  grind time", ui.FormatDuration(s.TotalTimeMS)),
		statRow("✅  success rate", fmt.Sprintf("%.1f%%", s.SuccessRate)),
	}
	return ui.Box.Render(strings.Join(rows, "\n"))
}

func statRow(label, value string) string {
	return ui.Label.Render(label) + ui.Value.Render(value)
}

// renderTopCmds renders a bar chart of the most-used commands.
func renderTopCmds(entries []db.TopEntry) string {
	max := float64(entries[0].Count)
	lines := []string{ui.Accent.Render("💻  top commands (no cap)")}
	lines = append(lines, "")

	for _, e := range entries {
		name := lipgloss.NewStyle().Width(14).Render(e.Name)
		bar := ui.ProgressBar(float64(e.Count), max, 14)
		count := ui.Muted.Render(fmt.Sprintf("  %s runs", ui.FormatNumber(e.Count)))
		lines = append(lines, name+bar+count)
	}
	return ui.Box.Render(strings.Join(lines, "\n"))
}

// renderTopProjects renders a bar chart of projects by time spent.
func renderTopProjects(entries []db.TopEntry) string {
	max := float64(entries[0].MS)
	lines := []string{ui.Accent.Render("📁  top projects")}
	lines = append(lines, "")

	for _, e := range entries {
		name := lipgloss.NewStyle().Width(18).Render(e.Name)
		bar := ui.ProgressBar(float64(e.MS), max, 14)
		dur := ui.Muted.Render("  " + ui.FormatDuration(e.MS))
		lines = append(lines, name+bar+dur)
	}
	return ui.Box.Render(strings.Join(lines, "\n"))
}
