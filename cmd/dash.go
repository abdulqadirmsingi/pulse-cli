package cmd

import (
	"fmt"
	"time"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var dashDays int
var dashRefresh int

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "open the live activity dashboard 🎮",
	Long:  "Interactive TUI dashboard. Tab through four views, auto-refreshes every 5 seconds.",
	RunE:  runDash,
}

func init() {
	rootCmd.AddCommand(dashCmd)
	dashCmd.Flags().IntVarP(&dashDays, "days", "d", 7, "days to include in stats")
	dashCmd.Flags().IntVar(&dashRefresh, "refresh", 5, "dashboard refresh interval in seconds")
}

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

	p := tea.NewProgram(tui.New(database, dashDays, time.Duration(dashRefresh)*time.Second), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
