package cmd

import (
	"fmt"

	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var resetForce bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "clear your command history and start fresh 🗑️",
	Long:  "Clears all logged commands from the database. Use --force to skip the confirmation prompt.",
	RunE:  runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().BoolVarP(&resetForce, "force", "f", false, "skip confirmation prompt")
}

func runReset(_ *cobra.Command, _ []string) error {
	if !resetForce {
		fmt.Println()
		fmt.Println(ui.Err.Render("  ⚠️  this will delete ALL ur command history"))
		fmt.Println(ui.Muted.Render("  run with --force to confirm:"))
		fmt.Println(ui.Muted.Render("    pulse reset --force"))
		fmt.Println()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	n, err := database.ResetCommands()
	if err != nil {
		return fmt.Errorf("resetting database: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render(fmt.Sprintf("deleted %s commands — ur starting fresh fr", ui.FormatNumber(n))))
	fmt.Println()
	return nil
}
