package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/tui"
	"github.com/spf13/cobra"
)

var dashDays int

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "open the live dashboard fr fr 🎮",
	Long:  "Interactive TUI dashboard. Tab through four views, auto-refreshes every 5 seconds.",
	RunE:  runDash,
}

func init() {
	rootCmd.AddCommand(dashCmd)
	dashCmd.Flags().IntVarP(&dashDays, "days", "d", 7, "days to include in stats")
}

// runDash launches the Bubble Tea program full-screen.
//
// 🧠 Go Lesson #39: tea.NewProgram(model, opts...) wires up the event loop.
// p.Run() blocks until the user quits — it reads keyboard input, calls
// Update on each event, then calls View to redraw.
//
// tea.WithAltScreen() switches to the terminal's alternate screen buffer
// so the dashboard doesn't pollute your scroll history. When you quit,
// the terminal snaps back to exactly where it was before.
func runDash(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	p := tea.NewProgram(tui.New(database, dashDays), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
