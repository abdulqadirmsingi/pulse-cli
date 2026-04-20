package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/tui"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "see what you worked on today 📅",
	Long:  "Hourly heatmap + command and project breakdown for the current day.",
	RunE:  runToday,
}

func init() {
	rootCmd.AddCommand(todayCmd)
}

func runToday(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	now := time.Now()

	stats, err := database.GetTodayStats()
	if err != nil {
		return err
	}
	topCmds, err := database.GetTopCommands(1, 5)
	if err != nil {
		return err
	}
	topProjects, err := database.GetTopProjects(1, 4)
	if err != nil {
		return err
	}
	hourly, err := database.GetHourlyStats(now.Format("2006-01-02"))
	if err != nil {
		return err
	}

	dayLabel := strings.ToLower(now.Format("Monday, January 2"))
	fmt.Println()
	fmt.Println(ui.Title.Render("📅  today — " + dayLabel))
	fmt.Println()

	rows := []string{
		ui.Label.Render("⚡  commands") + ui.Value.Render(ui.FormatNumber(stats.TotalCommands)),
		ui.Label.Render("⏰  time") + ui.Value.Render(ui.FormatDuration(stats.TotalTimeMS)),
		ui.Label.Render("✅  success") + ui.Value.Render(fmt.Sprintf("%.1f%%", stats.SuccessRate)),
	}
	fmt.Println(ui.Box.Render(strings.Join(rows, "\n")))
	fmt.Println()

	fmt.Println(ui.Accent.Render("  hourly activity"))
	fmt.Println()
	for _, line := range tui.HourlyChart(hourly, "  ") {
		fmt.Println(line)
	}
	fmt.Println()

	if len(topCmds) > 0 {
		printBarSection("top commands today", topCmds, false)
	}

	if len(topProjects) > 0 {
		printBarSection("top projects today", topProjects, true)
	}

	return nil
}

func printBarSection(title string, entries []db.TopEntry, byTime bool) {
	fmt.Println(ui.Accent.Render("  " + title))
	fmt.Println()
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
	for _, e := range entries {
		var val float64
		var suffix string
		if byTime {
			val = float64(e.MS)
			suffix = ui.FormatDuration(e.MS)
		} else {
			val = float64(e.Count)
			suffix = fmt.Sprintf("%d runs", e.Count)
		}
		name := lipgloss.NewStyle().Width(16).Render(ui.Truncate(e.Name, 15))
		bar := ui.ProgressBar(val, maxVal, 16)
		fmt.Println("  " + name + bar + ui.Muted.Render("  "+suffix))
	}
	fmt.Println()
}
