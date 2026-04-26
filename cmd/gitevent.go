package cmd

import (
	"os"
	"path/filepath"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/spf13/cobra"
)

// gitEventCmd is called by the global git hooks (post-commit, pre-push).
// It logs git events originating from IDEs and GUI clients into the same DB.
var gitEventCmd = &cobra.Command{
	Use:    "git-event [subcommand]",
	Short:  "record a git event from a git hook (internal)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runGitEvent,
}

var (
	gitEventBranch  string
	gitEventMessage string
	gitEventDir     string
)

func init() {
	rootCmd.AddCommand(gitEventCmd)
	gitEventCmd.Flags().StringVar(&gitEventBranch, "branch", "", "branch name")
	gitEventCmd.Flags().StringVar(&gitEventMessage, "message", "", "commit message")
	gitEventCmd.Flags().StringVar(&gitEventDir, "dir", "", "repo root directory")
}

func runGitEvent(_ *cobra.Command, args []string) error {
	subcommand := args[0]

	cfg, err := config.Load()
	if err != nil {
		recordHookError("git-event config", err)
		return nil
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		recordHookError("git-event database", err)
		return nil
	}
	defer database.Close()

	dir := gitEventDir
	if dir == "" {
		dir, _ = os.Getwd()
	}
	project := filepath.Base(dir)

	// insert a command row so it shows up in history and stats
	cmd := "git " + subcommand
	id, err := database.InsertCommandGetID(cmd, dir, project, 0, 0, false)
	if err != nil {
		recordHookError("git-event insert command", err)
		return nil
	}

	if err := database.InsertGitEvent(id, subcommand, gitEventBranch, "", gitEventMessage, false); err != nil {
		recordHookError("git-event insert metadata", err)
		return nil
	}
	clearHookError()
	return nil
}
