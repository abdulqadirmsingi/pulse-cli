package cmd

import (
	"fmt"
	"os"

	igit "github.com/abdulqadirmsingi/pulse-cli/internal/git"
	"github.com/abdulqadirmsingi/pulse-cli/internal/rules"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

// gitCheckCmd is called by the git() shell wrapper installed by `pulse git-guard on`.
// It evaluates only SeverityBlock rules. Exit 0 = git proceeds. Exit 1 = git is blocked.
var gitCheckCmd = &cobra.Command{
	Use:                "git-check",
	Short:              "pre-execution git rule check (called by git-guard wrapper)",
	Hidden:             true,
	DisableFlagParsing: true, // all args belong to git, not cobra
	RunE:               runGitCheck,
}

func init() {
	rootCmd.AddCommand(gitCheckCmd)
}

func runGitCheck(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	// reconstruct the full git command so Parse can work with it
	fullCmd := "git " + joinArgs(args)
	dir, _ := os.Getwd()

	ev := igit.Parse(fullCmd, dir)
	if ev == nil {
		return nil
	}

	engine := rules.Default()
	for _, v := range engine.Evaluate(ev) {
		if v.Severity != rules.SeverityBlock {
			continue
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  🚫 "+ui.Err.Render(v.Message))
		if v.Fix != "" {
			fmt.Fprintln(os.Stderr, "     "+ui.Muted.Render(v.Fix))
		}
		fmt.Fprintln(os.Stderr, "")
		// exit 1 causes the shell wrapper to abort before calling `command git`
		os.Exit(1)
	}
	return nil
}

// joinArgs reassembles args into a single string, quoting args that contain spaces.
func joinArgs(args []string) string {
	out := make([]string, len(args))
	for i, a := range args {
		if containsSpace(a) {
			out[i] = `"` + a + `"`
		} else {
			out[i] = a
		}
	}
	result := ""
	for i, s := range out {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\t' {
			return true
		}
	}
	return false
}
