package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var historyNoNoise bool

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "every command you ran today, in order 🕒",
	Long:  "Full chronological list of today's commands with time, duration, and exit status.",
	RunE:  runHistory,
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().BoolVar(&historyNoNoise, "no-noise", false, "hide noise commands (ls, cd, clear, etc.)")
}

func runHistory(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	cmds, err := database.GetTodayCommands()
	if err != nil {
		return fmt.Errorf("loading history: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("🕒  command history  ·  today"))
	fmt.Println()

	if len(cmds) == 0 {
		fmt.Println(ui.Muted.Render("  nothing logged yet today — run some commands first"))
		fmt.Println()
		return nil
	}

	timeStyle  := lipgloss.NewStyle().Foreground(ui.ColorGray).Width(8)
	cmdStyle   := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	noiseStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)
	durStyle   := lipgloss.NewStyle().Foreground(ui.ColorGray)
	failStyle  := lipgloss.NewStyle().Foreground(ui.ColorRed)

	shown := 0
	for _, c := range cmds {
		if historyNoNoise && c.Noise {
			continue
		}
		t := c.CreatedAt.Format("15:04:05")
		dur := ui.FormatDuration(c.DurationMS)
		cmd := ui.Truncate(c.Command, 60)

		var cmdRendered string
		if c.Noise {
			cmdRendered = noiseStyle.Render(cmd)
		} else if c.ExitCode != 0 {
			cmdRendered = failStyle.Render(cmd)
		} else {
			cmdRendered = cmdStyle.Render(cmd)
		}

		exitMark := ""
		if c.ExitCode != 0 {
			exitMark = " " + failStyle.Render(fmt.Sprintf("✗ %d", c.ExitCode))
		}

		fmt.Printf("  %s  %s%s  %s\n",
			timeStyle.Render(t),
			cmdRendered,
			exitMark,
			durStyle.Render(dur),
		)
		shown++
	}

	fmt.Println()
	summary := fmt.Sprintf("%d commands", shown)
	if historyNoNoise {
		total := len(cmds)
		hidden := total - shown
		if hidden > 0 {
			summary += fmt.Sprintf("  ·  %d noise hidden", hidden)
		}
	}
	fmt.Println("  " + ui.Muted.Render(summary))
	fmt.Println()

	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	if !historyNoNoise {
		fmt.Println("  " + ui.Muted.Render("tip: hide noise with ") + cyan.Render("--no-noise") +
			ui.Muted.Render("  ·  search history with ") + cyan.Render("pulse s <query>") +
			ui.Muted.Render("  ·  save a fav with ") + cyan.Render("pulse f add \"<cmd>\""))
		fmt.Println()
	}

	return nil
}
