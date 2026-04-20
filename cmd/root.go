// Package cmd contains every CLI command Pulse exposes.
//
// 🧠 Go Lesson #21: The `cmd` package is the boundary between user interaction
// and business logic. Commands parse flags and call internal/ packages.
// This separation keeps everything testable and easy to reason about.
package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// rootCmd is the base command — what runs when you type just `pulse`.
//
// 🧠 Go Lesson #22: &cobra.Command{...} fills a struct by field name.
// Fields you omit get their zero value (empty string, nil, false, etc.).
var rootCmd = &cobra.Command{
	Use:   "pulse",
	Short: "track ur grind, no cap 🔥",
	Long:  buildBanner(),
}

// Execute is the public entry point called from main().
//
// 🧠 Go Lesson #23: Exported names start with a capital letter.
// Execute() is visible to main.go. execute() would be package-private.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init() auto-runs when the package loads — used only for wiring, not logic.
//
// 🧠 Go Lesson #24: Every package can have multiple init() functions.
// We use one per cmd/*.go file to register each subcommand onto rootCmd.
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// buildBanner renders the ASCII welcome screen with lipgloss colors.
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
		purple.Render("  track ur grind, no cap 🔥") + "\n" +
		muted.Render("  run `pulse init` to get started\n")
}
