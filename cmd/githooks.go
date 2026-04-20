package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var gitHooksCmd = &cobra.Command{
	Use:   "hooks [install|uninstall|status]",
	Short: "track git activity from any client — IDE, terminal, desktop 🔗",
	Long:  "Installs global git hooks so Pulse tracks commits and blocks force-pushes regardless of whether you use the terminal, VS Code, Cursor, or GitHub Desktop.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGitHooks,
}

func init() {
	rootCmd.AddCommand(gitHooksCmd)
}

func runGitHooks(_ *cobra.Command, args []string) error {
	switch strings.ToLower(args[0]) {
	case "install":
		return gitHooksInstall()
	case "uninstall":
		return gitHooksUninstall()
	case "status":
		return gitHooksStatus()
	default:
		return fmt.Errorf("unknown argument %q — use install, uninstall, or status", args[0])
	}
}

func hooksDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "git", "hooks"), nil
}

func gitHooksInstall() error {
	self, err := os.Executable()
	if err != nil || self == "" {
		self = "pulse"
	}

	dir, err := hooksDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating hooks dir: %w", err)
	}

	hooks := map[string]string{
		"post-commit": postCommitHook(self),
		"pre-push":    prePushHook(self),
	}
	for name, content := range hooks {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			return fmt.Errorf("writing %s hook: %w", name, err)
		}
	}

	// set global core.hooksPath — works for every repo on the machine
	if err := exec.Command("git", "config", "--global", "core.hooksPath", dir).Run(); err != nil {
		return fmt.Errorf("setting core.hooksPath: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render("hooks installed in "+dir))
	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render("git config --global core.hooksPath set"))
	fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render("commits tracked from VS Code, Cursor, GitHub Desktop, and terminal"))
	fmt.Println()
	return nil
}

func gitHooksUninstall() error {
	dir, err := hooksDir()
	if err != nil {
		return err
	}

	for _, name := range []string{"post-commit", "pre-push"} {
		path := filepath.Join(dir, name)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		if !strings.Contains(string(content), "pulse") {
			continue // not our hook — don't touch it
		}
		os.Remove(path)
	}

	exec.Command("git", "config", "--global", "--unset", "core.hooksPath").Run()

	fmt.Printf("\n  %s  %s\n\n", ui.Success.Render("✓"), ui.Muted.Render("git hooks removed"))
	return nil
}

func gitHooksStatus() error {
	dir, err := hooksDir()
	if err != nil {
		return err
	}
	fmt.Println()
	for _, name := range []string{"post-commit", "pre-push"} {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(content), "pulse") {
			fmt.Printf("  %s  %-14s  %s\n", ui.Muted.Render("○"), name, ui.Muted.Render("not installed"))
		} else {
			fmt.Printf("  %s  %-14s  %s\n", ui.Success.Render("✓"), name, ui.Success.Render("active"))
		}
	}
	fmt.Println()
	return nil
}

func postCommitHook(pulseBin string) string {
	return fmt.Sprintf(`#!/bin/sh
# Pulse post-commit hook — tracks commits from any git client
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
MSG=$(git log -1 --pretty=%%s 2>/dev/null)
DIR=$(git rev-parse --show-toplevel 2>/dev/null)
%s git-event post-commit --branch "$BRANCH" --message "$MSG" --dir "$DIR" >/dev/null 2>&1 || true
`, pulseBin)
}

func prePushHook(pulseBin string) string {
	return fmt.Sprintf(`#!/bin/sh
# Pulse pre-push hook — blocks force-pushes to main from any git client
ZERO="0000000000000000000000000000000000000000"
while IFS=' ' read -r local_ref local_sha remote_ref remote_sha; do
    [ "$local_sha" = "$ZERO" ] && continue
    [ "$remote_sha" = "$ZERO" ] && continue
    branch=$(echo "$remote_ref" | sed 's|refs/heads/||')
    # if remote SHA is not an ancestor of local SHA, this is a force push
    if ! git merge-base --is-ancestor "$remote_sha" "$local_sha" 2>/dev/null; then
        %s git-check push --force origin "$branch" || exit 1
    fi
done
`, pulseBin)
}
