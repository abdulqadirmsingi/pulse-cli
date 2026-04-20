package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var (
	searchDays  int
	searchLimit int
)

var searchCmd = &cobra.Command{
	Use:     "search <query>",
	Aliases: []string{"s"},
	Short:   "search your command history 🔍",
	Long:    "Search all logged commands by keyword. Use --days to limit to recent history.",
	Args:    cobra.ExactArgs(1),
	RunE:    runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVar(&searchDays, "days", 0, "limit to last N days (default: all time)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 200, "max results to show")
}

func runSearch(_ *cobra.Command, args []string) error {
	query := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	cmds, err := database.SearchCommands(query, searchDays, searchLimit)
	if err != nil {
		return fmt.Errorf("searching history: %w", err)
	}

	fmt.Println()
	scope := "all time"
	if searchDays > 0 {
		scope = fmt.Sprintf("last %d days", searchDays)
	}
	fmt.Println(ui.Title.Render(fmt.Sprintf("🔍  results for \"%s\"  ·  %s", query, scope)))
	fmt.Println()

	if len(cmds) == 0 {
		fmt.Println(ui.Muted.Render("  no matches found"))
		fmt.Println()
		fmt.Println("  " + ui.Muted.Render("tip: try a shorter keyword  ·  or drop --days to search all time"))
		fmt.Println()
		return nil
	}

	timeStyle := lipgloss.NewStyle().Foreground(ui.ColorGray).Width(10)
	cmdStyle  := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	durStyle  := lipgloss.NewStyle().Foreground(ui.ColorGray)
	failStyle := lipgloss.NewStyle().Foreground(ui.ColorRed)
	dateStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)

	for _, c := range cmds {
		date := c.CreatedAt.Format("01/02 15:04")
		dur  := ui.FormatDuration(c.DurationMS)
		cmd  := ui.Truncate(c.Command, 60)

		var cmdRendered string
		if c.ExitCode != 0 {
			cmdRendered = failStyle.Render(cmd)
		} else {
			cmdRendered = cmdStyle.Render(cmd)
		}

		exitMark := ""
		if c.ExitCode != 0 {
			exitMark = " " + failStyle.Render(fmt.Sprintf("✗ %d", c.ExitCode))
		}

		fmt.Printf("  %s  %s  %s%s\n",
			dateStyle.Render(date),
			timeStyle.Render(""),
			cmdRendered,
			exitMark,
		)
		_ = durStyle.Render(dur) // available if we want to add duration column later
	}

	fmt.Println()
	fmt.Println("  " + ui.Muted.Render(fmt.Sprintf("%d match(es)", len(cmds))))
	fmt.Println()

	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Println("  " + ui.Muted.Render("tip: narrow results with ") + cyan.Render("--days 7") +
		ui.Muted.Render("  ·  save a fav with ") + cyan.Render("pulse f add \"<cmd>\""))
	fmt.Println()

	return nil
}
