package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	igit "github.com/devpulse-cli/devpulse/internal/git"
	"github.com/devpulse-cli/devpulse/internal/rules"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:    "log",
	Short:  "log a shell command (called by the shell hook)",
	Hidden: true,
	RunE:   runLog,
}

var (
	logFlagCmd  string
	logFlagExit int
	logFlagMS   int64
	logFlagDir  string
)

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.Flags().StringVar(&logFlagCmd, "cmd", "", "the command that ran")
	logCmd.Flags().IntVar(&logFlagExit, "exit", 0, "exit code")
	logCmd.Flags().Int64Var(&logFlagMS, "ms", 0, "duration in milliseconds")
	logCmd.Flags().StringVar(&logFlagDir, "dir", "", "working directory")
}

// reANSI matches ANSI/VT100 escape sequences that terminals embed in command strings.
// 🧠 Go Lesson #51: Package-level compiled regexps are free after startup —
// compile once with MustCompile, reuse on every call with no allocation.
var reANSI = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[a-zA-Z]|\][^\x07]*\x07|[^[\]])`)

// sanitizeCmd strips escape sequences, control characters, and excess whitespace.
func sanitizeCmd(cmd string) string {
	cmd = reANSI.ReplaceAllString(cmd, "")
	return strings.Join(strings.Fields(cmd), " ")
}

func runLog(_ *cobra.Command, _ []string) error {
	cmd := sanitizeCmd(logFlagCmd)
	if shouldSkip(cmd) {
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil
	}
	defer database.Close()

	dir := logFlagDir
	if dir == "" {
		dir, _ = os.Getwd()
	}

	project := projectFromDir(dir)
	noise := isNoise(cmd)

	// for git commands: store event + evaluate rules
	if igit.IsGit(cmd) {
		id, err := database.InsertCommandGetID(cmd, dir, project, logFlagExit, logFlagMS, noise)
		if err == nil {
			if ev := igit.Parse(cmd, dir); ev != nil {
				_ = database.InsertGitEvent(id, ev.Subcommand, ev.Branch, ev.Remote, ev.Message, ev.IsForce)
				// only surface feedback when the git command itself succeeded
				if logFlagExit == 0 {
					printViolations(rules.Default().Evaluate(ev))
				}
			}
		}
		return nil
	}

	_ = database.InsertCommand(cmd, dir, project, logFlagExit, logFlagMS, noise)
	return nil
}

func printViolations(violations []rules.Violation) {
	for _, v := range violations {
		icon := "⚠️ "
		style := ui.Muted
		if v.Severity == rules.SeverityBlock {
			icon = "🚫"
			style = ui.Err
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  "+icon+" "+style.Render(v.Message))
		if v.Fix != "" {
			fmt.Fprintln(os.Stderr, "     "+ui.Muted.Render(v.Fix))
		}
	}
}

func shouldSkip(cmd string) bool {
	if cmd == "" {
		return true
	}
	base := strings.Fields(cmd)[0]
	return base == "pulse"
}

// noiseCommands are shell housekeeping commands that clutter dev stats.
var noiseCommands = map[string]bool{
	"clear": true, "cls": true,
	"ls": true, "ll": true, "la": true, "l": true,
	"cd": true, "pwd": true,
	"exit": true, "logout": true,
	"history": true,
	"cat": true, "less": true, "more": true,
	"man": true,
}

func isNoise(cmd string) bool {
	if cmd == "" {
		return false
	}
	base := strings.Fields(cmd)[0]
	return noiseCommands[base]
}

func projectFromDir(dir string) string {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return filepath.Base(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return filepath.Base(dir)
}
