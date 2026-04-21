package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var gitGuardCmd = &cobra.Command{
	Use:   "git-guard [on|off|status]",
	Short: "block dangerous git operations before they run 🛡️",
	Long:  "Wraps the git command in your shell to intercept force-pushes to main before they happen. Disabled by default.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGitGuard,
}

func init() {
	rootCmd.AddCommand(gitGuardCmd)
}

func runGitGuard(_ *cobra.Command, args []string) error {
	switch strings.ToLower(args[0]) {
	case "on":
		return gitGuardEnable()
	case "off":
		return gitGuardDisable()
	case "status":
		return gitGuardStatus()
	default:
		return fmt.Errorf("unknown argument %q — use on, off, or status", args[0])
	}
}

func gitGuardEnable() error {
	rcFile, err := shellRCFile()
	if err != nil {
		return err
	}
	existing, _ := os.ReadFile(rcFile)
	if strings.Contains(string(existing), "Pulse git-guard") {
		fmt.Printf("  %s  %s\n", ui.Success.Render("~"), ui.Muted.Render("git-guard is already enabled"))
		return nil
	}

	self, err := os.Executable()
	if err != nil || self == "" {
		self = "pulse"
	}

	hook := fmt.Sprintf(`
# ── Pulse git-guard ─────────────────────────────────────
git() {
    %s git-check "$@" || return $?
    command git "$@"
}
# ────────────────────────────────────────────────────────
`, self)

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("writing to %s: %w", rcFile, err)
	}
	defer f.Close()
	if _, err := f.WriteString(hook); err != nil {
		return fmt.Errorf("writing git-guard hook: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render("git-guard enabled in "+rcFile))
	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render("force-pushes to main will now be blocked"))
	fmt.Println()
	fmt.Println("  " + ui.Muted.Render("activate now with: source "+rcFile))
	fmt.Println()
	return nil
}

func gitGuardDisable() error {
	rcFile, err := shellRCFile()
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(rcFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", rcFile, err)
	}
	if !strings.Contains(string(existing), "Pulse git-guard") {
		fmt.Printf("  %s  %s\n", ui.Muted.Render("~"), ui.Muted.Render("git-guard is not enabled"))
		return nil
	}
	cleaned := removeGitGuardBlock(string(existing))
	if err := os.WriteFile(rcFile, []byte(cleaned), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", rcFile, err)
	}
	fmt.Printf("\n  %s  %s\n\n", ui.Success.Render("✓"), ui.Muted.Render("git-guard removed — run: source "+rcFile))
	return nil
}

func gitGuardStatus() error {
	rcFile, err := shellRCFile()
	if err != nil {
		return err
	}
	existing, _ := os.ReadFile(rcFile)
	fmt.Println()
	if strings.Contains(string(existing), "Pulse git-guard") {
		fmt.Printf("  %s  git-guard  %s\n", ui.Success.Render("✓"), ui.Success.Render("enabled"))
		fmt.Println("  " + ui.Muted.Render("force-pushes to main are blocked"))
	} else {
		fmt.Printf("  %s  git-guard  %s\n", ui.Muted.Render("○"), ui.Muted.Render("disabled"))
		fmt.Println("  " + ui.Muted.Render("run `pulse git-guard on` to enable"))
	}
	fmt.Println()
	return nil
}

func shellRCFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return filepath.Join(home, ".zshrc"), nil
	}
	return filepath.Join(home, ".bashrc"), nil
}

var gitGuardBlockRe = regexp.MustCompile(`(?s)\n# ── Pulse git-guard.*?# ────────────────────────────────────────────────────────\n`)

func removeGitGuardBlock(s string) string {
	return gitGuardBlockRe.ReplaceAllString(s, "\n")
}
