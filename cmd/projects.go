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

var projectsDays int

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "see time spent across your projects 📁",
	Long:  "Lists every detected project with time spent, command count, and success rate.",
	RunE:  runProjects,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.Flags().IntVarP(&projectsDays, "days", "d", 7, "days to look back")
}

func runProjects(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	projects, err := database.GetProjectList(projectsDays)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Title.Render(fmt.Sprintf("📁  projects  ·  last %d days", projectsDays)))
	fmt.Println()

	if len(projects) == 0 {
		fmt.Println(ui.Muted.Render("  no projects yet — run commands inside a git repo"))
		fmt.Println()
		return nil
	}

	// Column widths
	col1 := lipgloss.NewStyle().Width(22)
	col2 := lipgloss.NewStyle().Width(12)
	col3 := lipgloss.NewStyle().Width(12)
	col4 := lipgloss.NewStyle().Width(10)
	hdr := ui.Muted

	// Header
	fmt.Println("  " +
		col1.Render(hdr.Render("project")) +
		col2.Render(hdr.Render("time")) +
		col3.Render(hdr.Render("commands")) +
		col4.Render(hdr.Render("success")))
	fmt.Println("  " + ui.Muted.Render(strings.Repeat("─", 56)))

	for _, p := range projects {
		name := lipgloss.NewStyle().Foreground(ui.ColorCyan).Width(22).Render(ui.Truncate(p.Name, 21))
		timeStr := col2.Render(ui.FormatDuration(p.TotalTimeMS))
		cmds := col3.Render(ui.FormatNumber(p.Commands))
		success := lipgloss.NewStyle().
			Foreground(rateColor(p.SuccessRate)).
			Width(10).
			Render(fmt.Sprintf("%.1f%%", p.SuccessRate))
		fmt.Println("  " + name + timeStr + cmds + success)
	}
	fmt.Println()
	return nil
}

func rateColor(rate float64) lipgloss.Color {
	switch {
	case rate >= 95:
		return ui.ColorGreen
	case rate >= 80:
		return ui.ColorGold
	default:
		return ui.ColorRed
	}
}
