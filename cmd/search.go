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

var (
	searchDays  int
	searchLimit int
)

var searchCmd = &cobra.Command{
	Use:     "search [query...]",
	Aliases: []string{"s"},
	Short:   "search your command history 🔍",
	Long:    "Search all logged commands by keyword. Quotes are optional — pulse s git checkout works fine.",
	Args:    cobra.ArbitraryArgs,
	RunE:    runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVar(&searchDays, "days", 0, "limit to last N days (default: all time)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 200, "max results to show")
}

func runSearch(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	// join all args so  pulse s git checkout  works without quotes
	query := strings.TrimSpace(strings.Join(args, " "))

	// no query → show top commands as a starting point
	if query == "" {
		return runSearchHelp(database)
	}
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

	cmdStyle  := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	durStyle  := lipgloss.NewStyle().Foreground(ui.ColorGray)
	failStyle := lipgloss.NewStyle().Foreground(ui.ColorRed)
	dateStyle := lipgloss.NewStyle().Foreground(ui.ColorGray).Width(14)

	for _, c := range cmds {
		date := c.CreatedAt.Local().Format("01/02 15:04")
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

		fmt.Printf("  %s  %s%s  %s\n",
			dateStyle.Render(date),
			cmdRendered,
			exitMark,
			durStyle.Render(dur),
		)
	}

	fmt.Println()
	word := "matches"
	if len(cmds) == 1 {
		word = "match"
	}
	fmt.Println("  " + ui.Muted.Render(fmt.Sprintf("%d %s", len(cmds), word)))
	fmt.Println()

	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Println("  " + ui.Muted.Render("tip: narrow results with ") + cyan.Render("--days 7") +
		ui.Muted.Render("  ·  save a fav with ") + cyan.Render("pulse f add \"<cmd>\""))
	fmt.Println()

	return nil
}

// runSearchHelp is shown when pulse s is run with no query.
// It shows the user's top commands as search suggestions.
func runSearchHelp(database *db.DB) error {
	top, _ := database.GetTopCommands(90, 8)

	fmt.Println()
	fmt.Println(ui.Title.Render("🔍  search your history"))
	fmt.Println()

	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	muted := ui.Muted

	fmt.Println("  " + muted.Render("usage:  ") + cyan.Render("pulse s <keyword>") +
		muted.Render("   or   ") + cyan.Render("pulse s git checkout"))
	fmt.Println()

	if len(top) > 0 {
		fmt.Println("  " + ui.Accent.Render("your most-used commands  ·  try searching one:"))
		fmt.Println()

		nameStyle  := lipgloss.NewStyle().Foreground(ui.ColorCyan).Width(20)
		countStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)

		for _, t := range top {
			count := fmt.Sprintf("%d runs", t.Count)
			fmt.Printf("  %s  %s  %s\n",
				nameStyle.Render(ui.Truncate(t.Name, 18)),
				countStyle.Render(count),
				muted.Render("→  pulse s \""+t.Name+"\""),
			)
		}
		fmt.Println()
		fmt.Println("  " + muted.Render("save a command you keep coming back to:  ") + cyan.Render("pulse f add \"<cmd>\""))
	} else {
		fmt.Println("  " + muted.Render("no history yet — run some commands first, then come back"))
	}

	fmt.Println()
	return nil
}
