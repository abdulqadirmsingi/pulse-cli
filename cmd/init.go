package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "set up Pulse on ur machine fr fr 🚀",
	Long:  "Creates the data directory, initialises the database, and installs shell hooks.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("🚀 setting up Pulse..."))
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := cfg.EnsureDir(); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	printInitStep("✓", "data directory ready at "+cfg.DataDir)

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("initialising database: %w", err)
	}
	database.Close()
	printInitStep("✓", "database initialised")

	shell := detectShell()
	hookFile, hookContent := shellHook(shell)
	wrote, err := writeHook(hookFile, hookContent)
	if err != nil {
		return fmt.Errorf("installing shell hook: %w", err)
	}
	if wrote {
		printInitStep("✓", fmt.Sprintf("%s hook installed in %s", shell, hookFile))
	}
	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))
	fmt.Println(ui.Box.Render(
		ui.Success.Render("ur Pulse is ready to slay 🔥")+"\n\n"+
			ui.Muted.Render("activate by running:")+"\n"+
			cyan.Render("  source "+hookFile)+"\n\n"+
			ui.Muted.Render("then try:")+"\n"+
			cyan.Render("  pulse stats"),
	))
	fmt.Println()
	return nil
}

func printInitStep(icon, msg string) {
	fmt.Printf("  %s  %s\n", ui.Success.Render(icon), ui.Muted.Render(msg))
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	switch {
	case strings.Contains(shell, "zsh"):
		return "zsh"
	case strings.Contains(shell, "fish"):
		return "fish"
	default:
		return "bash"
	}
}

func shellHook(shell string) (hookFile, content string) {
	home, _ := os.UserHomeDir()

	// date +%s%3N (milliseconds) is Linux-only and breaks on macOS.
	// We use date +%s (seconds) universally, then multiply by 1000 for ms.
	shared := `
# ── Pulse shell hook ────────────────────────────────────
_pulse_preexec() {
    _PULSE_CMD_START=$(date +%s)
    _PULSE_CMD="$1"
}
_pulse_precmd() {
    local _exit=$?
    [ -z "$_PULSE_CMD" ] && return
    local _ms=$(( ($(date +%s) - ${_PULSE_CMD_START:-0}) * 1000 ))
    pulse log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" 2>/dev/null &
    unset _PULSE_CMD _PULSE_CMD_START
}
`

	switch shell {
	case "zsh":
		content = shared + `
autoload -Uz add-zsh-hook
add-zsh-hook preexec _pulse_preexec
add-zsh-hook precmd  _pulse_precmd
# ────────────────────────────────────────────────────────
`
		return filepath.Join(home, ".zshrc"), content

	default: // bash
		content = shared + `
trap '_pulse_preexec "$BASH_COMMAND"' DEBUG
PROMPT_COMMAND="_pulse_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
# ────────────────────────────────────────────────────────
`
		return filepath.Join(home, ".bashrc"), content
	}
}

// writeHook appends the hook to the shell config. Returns (true, nil) if written,
// (false, nil) if already present. Safe to run multiple times (idempotent).
func writeHook(hookFile, content string) (wrote bool, err error) {
	existing, err := os.ReadFile(hookFile)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if strings.Contains(string(existing), "Pulse shell hook") {
		printInitStep("~", "hook already installed, nothing to do")
		return false, nil
	}
	f, err := os.OpenFile(hookFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err == nil, err
}
