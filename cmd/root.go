package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/devpulse-cli/devpulse/internal/updater"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pulse",
	Short: "track your dev activity, automatically 🔥",
	Long:  buildBanner(),
	PersistentPostRun: func(cmd *cobra.Command, _ []string) {
		if cmd.Hidden {
			return
		}
		cfg, err := config.Load()
		if err != nil {
			return
		}
		if latest := updater.CheckAvailable(cfg.DataDir); latest != "" {
			cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))
			fmt.Println()
			fmt.Printf("  %s  %s\n",
				ui.Accent.Render("update available →"),
				cyan.Render("pulse update")+" "+ui.Muted.Render("(v"+config.AppVersion+" → v"+latest+")"),
			)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func buildBanner() string {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF")).Bold(true)
	purple := lipgloss.NewStyle().Foreground(lipgloss.Color("#9D4EDD"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	art := cyan.Render(`
  ██████╗ ██╗   ██╗██╗     ███████╗███████╗
  ██╔══██╗██║   ██║██║     ██╔════╝██╔════╝
  ██████╔╝██║   ██║██║     ███████╗█████╗
  ██╔═══╝ ██║   ██║██║     ╚════██║██╔══╝
  ██║     ╚██████╔╝███████╗███████║███████╗
  ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝`)

	return art + "\n\n" +
		purple.Render("  track ur grind faster! 🔥") + "\n" +
		muted.Render("  run `pulse init` to get started\n")
}
