package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/spf13/cobra"
)

// logCmd is intentionally hidden from `pulse help` — it's called by the shell
// hook automatically, not by the user directly.
var logCmd = &cobra.Command{
	Use:    "log",
	Short:  "log a shell command (called by the shell hook)",
	Hidden: true,
	RunE:   runLog,
}

// Flag variables for `pulse log`.
//
// 🧠 Go Lesson #27: Cobra flags bind directly to Go variables via pointers.
// StringVar, IntVar, Int64Var all write their parsed value into the variable
// you pass. This avoids manual string-to-type conversion boilerplate.
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

// runLog fires on every command the user runs (via shell hook).
// It must be fast and silent — failures are swallowed so they never
// interrupt the user's actual workflow.
//
// 🧠 Go Lesson #28: Returning nil from a RunE function means "success".
// Here we return nil even on internal errors because this runs in the
// background and should NEVER cause visible noise in the terminal.
func runLog(_ *cobra.Command, _ []string) error {
	if shouldSkip(logFlagCmd) {
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil // silent fail
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil // silent fail
	}
	defer database.Close()

	dir := logFlagDir
	if dir == "" {
		dir, _ = os.Getwd()
	}

	project := projectFromDir(dir)
	_ = database.InsertCommand(logFlagCmd, dir, project, logFlagExit, logFlagMS)
	return nil
}

// shouldSkip returns true for commands we don't want to log.
func shouldSkip(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return true
	}
	base := strings.Fields(cmd)[0]
	return base == "pulse"
}

// projectFromDir resolves a human-readable project name from a directory.
// It walks up the directory tree looking for a .git folder — that root
// becomes the project name. Falls back to the immediate directory name.
//
// 🧠 Go Lesson #29: filepath.Dir() returns the parent directory.
// filepath.Base() returns the last path element (the directory name).
// filepath.Join() builds OS-correct paths (/ on Unix, \ on Windows).
func projectFromDir(dir string) string {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return filepath.Base(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break // reached filesystem root
		}
		current = parent
	}
	return filepath.Base(dir)
}
