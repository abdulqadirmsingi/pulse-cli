package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/insights"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var vibeDays int

var vibeCmd = &cobra.Command{
	Use:   "vibe",
	Short: "analyse your coding patterns and habits ✨",
	Long:  "Analyses your coding patterns and tells you what they mean — no fluff.",
	RunE:  runVibe,
}

func init() {
	rootCmd.AddCommand(vibeCmd)
	vibeCmd.Flags().IntVarP(&vibeDays, "days", "d", 7, "days to analyse")
}

func runVibe(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	stats, err := database.GetStats(vibeDays)
	if err != nil {
		return err
	}
	topCmds, err := database.GetTopCommands(vibeDays, 8)
	if err != nil {
		return err
	}
	topProjects, err := database.GetTopProjects(vibeDays, 8)
	if err != nil {
		return err
	}

	report := insights.Analyse(stats, topCmds, topProjects)

	fmt.Println()
	fmt.Println(ui.Title.Render(fmt.Sprintf("✨  Your vibe check  ·  last %d days", vibeDays)))
	fmt.Println()

	// Quick stats line
	statsLine := fmt.Sprintf("%s commands  ·  %s grind time  ·  %.1f%% success  ·  %d day streak",
		ui.FormatNumber(stats.TotalCommands),
		ui.FormatDuration(stats.TotalTimeMS),
		stats.SuccessRate,
		stats.StreakDays,
	)
	fmt.Println("  " + ui.Muted.Render(statsLine))
	fmt.Println()

	// Observations
	if len(report.Observations) > 0 {
		fmt.Println(ui.Accent.Render("  the vibe"))
		fmt.Println()
		for _, obs := range report.Observations {
			fmt.Println(renderInsight(obs))
		}
		fmt.Println()
	}

	// Tips — only shown if there's something actionable
	if len(report.Tips) > 0 {
		fmt.Println(ui.Accent.Render("  tips (no fluff)"))
		fmt.Println()
		for _, tip := range report.Tips {
			fmt.Println(renderInsight(tip))
		}
		fmt.Println()
	}

	// No data guard
	if len(report.Observations) == 0 {
		fmt.Println(ui.Muted.Render("  nothing to analyse yet — run some commands first"))
		fmt.Println()
	}

	return nil
}

// renderInsight formats a single insight line.
//
// 🧠 Go Lesson #48: lipgloss.NewStyle() can be called inline when you only
// need it once — you don't have to store every style as a package-level var.
// The style is stack-allocated and cheap to create.
func renderInsight(ins insights.Insight) string {
	icon := ins.Level.Icon()
	msg := ins.Message

	// Color the message based on level
	var styled string
	switch ins.Level {
	case insights.LevelFire:
		styled = lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(msg)
	case insights.LevelGood:
		styled = lipgloss.NewStyle().Foreground(ui.ColorCyan).Render(msg)
	case insights.LevelHeadsUp:
		styled = lipgloss.NewStyle().Foreground(ui.ColorGold).Render(msg)
	case insights.LevelRoast:
		styled = lipgloss.NewStyle().Foreground(ui.ColorRed).Render(msg)
	default: // tip
		styled = lipgloss.NewStyle().Foreground(ui.ColorPurple).Render(msg)
	}

	// Pad icon so messages align regardless of emoji width
	return "  " + icon + "  " + styled
}

