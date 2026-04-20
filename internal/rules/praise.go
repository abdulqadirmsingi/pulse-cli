package rules

import (
	"math/rand"
	"strings"

	"github.com/devpulse-cli/devpulse/internal/git"
)

var commitPraises = []string{
	"clean commit message ✓",
	"solid commit message 🎯",
	"commit message on point ✓",
	"that's how you commit 🔥",
	"good commit ✓",
}

var branchPraises = []string{
	"clean branch name ✓",
	"good branch name 🎯",
	"that's a proper branch name ✓",
	"good branch name 🔥",
}

var goodBranchPrefixes = []string{
	"feat/", "fix/", "chore/", "docs/", "refactor/",
	"test/", "perf/", "ci/", "build/", "hotfix/",
}

// GoodCommitPraise fires when a commit message follows the conventional format.
type GoodCommitPraise struct{}

func (r *GoodCommitPraise) Name() string { return "good-commit" }

func (r *GoodCommitPraise) Evaluate(e *git.Event) *Praise {
	if e.Subcommand != "commit" || e.Message == "" {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(e.Message))
	for _, p := range conventionalPrefixes {
		if strings.HasPrefix(lower, p) {
			return &Praise{
				Rule:    r.Name(),
				Message: commitPraises[rand.Intn(len(commitPraises))],
			}
		}
	}
	return nil
}

// GoodBranchPraise fires when a new branch follows the feat/fix/chore naming convention.
type GoodBranchPraise struct{}

func (r *GoodBranchPraise) Name() string { return "good-branch" }

func (r *GoodBranchPraise) Evaluate(e *git.Event) *Praise {
	if e.Subcommand != "checkout" && e.Subcommand != "switch" {
		return nil
	}
	if !hasCreateFlag(e.Args) {
		return nil
	}
	branch := strings.ToLower(newBranchName(e.Args))
	for _, p := range goodBranchPrefixes {
		if strings.HasPrefix(branch, p) {
			return &Praise{
				Rule:    r.Name(),
				Message: branchPraises[rand.Intn(len(branchPraises))],
			}
		}
	}
	return nil
}
