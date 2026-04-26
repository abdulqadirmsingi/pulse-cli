package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:     "backup [path]",
	Aliases: []string{"export"},
	Short:   "backup your local Pulse data 💾",
	Long:    "Copies your local SQLite database to a portable backup file. No cloud, no account, just your data.",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)
}

func runBackup(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if _, err := os.Stat(cfg.DBPath); err != nil {
		return fmt.Errorf("database not found — run pulse init first")
	}

	out := defaultBackupPath()
	if len(args) == 1 {
		out = args[0]
	}
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("creating backup directory: %w", err)
	}
	if err := copyFile(out, cfg.DBPath); err != nil {
		return fmt.Errorf("writing backup: %w", err)
	}

	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Println()
	fmt.Println(ui.Title.Render("💾  backup complete"))
	fmt.Println()
	fmt.Println("  " + ui.Success.Render("saved") + "  " + cyan.Render(out))
	fmt.Println("  " + ui.Muted.Render("keep this file safe — it contains your full local Pulse history"))
	fmt.Println()
	return nil
}

func defaultBackupPath() string {
	name := "pulse-backup-" + time.Now().Format("20060102-150405") + ".db"
	home, err := os.UserHomeDir()
	if err != nil {
		return name
	}
	return filepath.Join(home, name)
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
