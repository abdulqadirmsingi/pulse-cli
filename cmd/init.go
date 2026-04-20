package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var initReinstall bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "set up Pulse on ur machine fr fr 🚀",
	Long:  "Creates the data directory, initialises the database, and installs shell hooks.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initReinstall, "reinstall", false, "remove old hook and install a fresh one")
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

	// Resolve the full path of the running binary so the hook never gets
	// "command not found" errors, regardless of the user's PATH at hook time.
	//
	// 🧠 Go Lesson #44: os.Executable() returns the path of the current binary.
	// This is how self-referential tools embed their own install location.
	binaryPath, err := os.Executable()
	if err != nil || binaryPath == "" {
		binaryPath = "pulse" // fallback — works if pulse is in PATH
	}

	shell := detectShell()
	hookFile, hookContent := shellHook(shell, binaryPath)
	wrote, err := writeHook(hookFile, hookContent, initReinstall)
	if err != nil {
		return fmt.Errorf("installing shell hook: %w", err)
	}
	if wrote {
		printInitStep("✓", fmt.Sprintf("%s hook installed in %s", shell, hookFile))
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))
	fmt.Println(ui.Box.Render(
		ui.Success.Render("Your Pulse is ready to slay 🔥")+"\n\n"+
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

// shellHook builds the hook script for the given shell.
// binaryPath is the absolute path to the pulse binary, embedded directly in
// the hook so it works even if pulse is not in the shell's PATH at hook time.
//
// Key fixes vs the old hook:
//   - Full binary path instead of bare `pulse` — fixes "exit 127"
//   - zsh uses `&|` (background + immediate disown) — fixes [1] job noise
//   - bash uses `& disown $!` — same effect on bash
func shellHook(shell, binaryPath string) (hookFile, content string) {
	home, _ := os.UserHomeDir()

	switch shell {
	case "zsh":
		content = fmt.Sprintf(`
# ── Pulse shell hook ────────────────────────────────────
_pulse_preexec() {
    _PULSE_CMD_START=$(date +%%s)
    _PULSE_CMD="$1"
}
_pulse_precmd() {
    local _exit=$?
    [ -z "$_PULSE_CMD" ] && return
    local _ms=$(( ($(date +%%s) - ${_PULSE_CMD_START:-0}) * 1000 ))
    %s log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" >/dev/null 2>&1 &|
    unset _PULSE_CMD _PULSE_CMD_START
}
autoload -Uz add-zsh-hook
add-zsh-hook preexec _pulse_preexec
add-zsh-hook precmd  _pulse_precmd
# ────────────────────────────────────────────────────────
`, binaryPath)
		return filepath.Join(home, ".zshrc"), content

	default: // bash
		content = fmt.Sprintf(`
# ── Pulse shell hook ────────────────────────────────────
_pulse_preexec() {
    _PULSE_CMD_START=$(date +%%s)
    _PULSE_CMD="$BASH_COMMAND"
}
_pulse_precmd() {
    local _exit=$?
    [ -z "$_PULSE_CMD" ] && return
    local _ms=$(( ($(date +%%s) - ${_PULSE_CMD_START:-0}) * 1000 ))
    %s log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" >/dev/null 2>&1 &
    disown $! 2>/dev/null || true
    unset _PULSE_CMD _PULSE_CMD_START
}
trap '_pulse_preexec' DEBUG
PROMPT_COMMAND="_pulse_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
# ────────────────────────────────────────────────────────
`, binaryPath)
		return filepath.Join(home, ".bashrc"), content
	}
}

// writeHook writes the hook to the config file.
// If reinstall=true it removes any existing Pulse hook block first.
func writeHook(hookFile, content string, reinstall bool) (bool, error) {
	existing, err := os.ReadFile(hookFile)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	text := string(existing)

	if reinstall {
		text = removeHookBlock(text)
		printInitStep("~", "removed old hook")
	} else if strings.Contains(text, "Pulse shell hook") {
		printInitStep("~", "hook already installed — use --reinstall to update it")
		return false, nil
	}

	return true, os.WriteFile(hookFile, []byte(text+content), 0644)
}

// removeHookBlock strips the Pulse hook block from a shell config string.
//
// 🧠 Go Lesson #45: regexp.MustCompile panics at startup if the pattern is
// invalid — use it for compile-time-known patterns. For user-supplied patterns
// use regexp.Compile which returns (Regexp, error).
var hookBlockRe = regexp.MustCompile(`(?s)\n# ── Pulse shell hook.*?# ────────────────────────────────────────────────────────\n`)

func removeHookBlock(s string) string {
	return hookBlockRe.ReplaceAllString(s, "\n")
}
