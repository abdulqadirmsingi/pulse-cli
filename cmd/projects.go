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
	Short: "where ur time actually goes 📁",
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

	// Rows
	// 🧠 Go Lesson #42: Range over a slice returns (index, value).
	// We use _ to discard the index when we only need the value.
	// This is idiomatic Go — never use i when you don't need it.
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

// rateColor maps a success percentage to a traffic-light color.
//
// 🧠 Go Lesson #43: lipgloss.Color is just `type Color string` under the hood.
// Go lets you define methods and use types like this freely — no boxing,
// no heap allocation, just a named string with extra type safety.
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
