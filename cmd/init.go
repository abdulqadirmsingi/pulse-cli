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
	Short: "set up DevPulse on ur machine fr fr 🚀",
	Long:  "Creates the data directory, initialises the database, and installs shell hooks.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("🚀 setting up DevPulse..."))
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
	if err := writeHook(hookFile, hookContent); err != nil {
		return fmt.Errorf("installing shell hook: %w", err)
	}
	printInitStep("✓", fmt.Sprintf("%s hook installed in %s", shell, hookFile))
	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))
	fmt.Println(ui.Box.Render(
		ui.Success.Render("ur DevPulse is ready to slay 🔥")+"\n\n"+
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

	shared := `
# ── DevPulse shell hook ─────────────────────────────────
_pulse_preexec() {
    export _PULSE_CMD_START=$(date +%s%3N 2>/dev/null || echo 0)
    export _PULSE_CMD="$1"
}
_pulse_precmd() {
    local _exit=$?
    [ -z "$_PULSE_CMD" ] && return
    local _end=$(date +%s%3N 2>/dev/null || echo 0)
    local _dur=$(( _end - _PULSE_CMD_START ))
    pulse log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_dur" --dir "$PWD" 2>/dev/null &
    unset _PULSE_CMD
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

func writeHook(hookFile, content string) error {
	existing, err := os.ReadFile(hookFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(existing), "DevPulse shell hook") {
		printInitStep("~", "hook already installed, skipping")
		return nil
	}
	f, err := os.OpenFile(hookFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
