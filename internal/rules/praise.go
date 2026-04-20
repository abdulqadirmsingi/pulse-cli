package rules

import (
	"math/rand"
	"strings"

	"github.com/devpulse-cli/devpulse/internal/git"
)

var commitPraises = []string{
	"Clean commit message ✓",
	"Solid commit message 🎯",
	"That's how you commit 🔥",
	"Future-you will thank you for this one",
	"This is what good git history looks like 📖",
	"Chef's kiss on that commit message 🤌",
	"Your teammates just smiled reading that",
	"Clean, clear, conventional — perfect",
	"This is the way 🚀",
	"Commit message game strong 💪",
	"Exactly what git log should look like",
	"Now that's a commit message worth keeping",
}

var branchPraises = []string{
	"Clean branch name ✓",
	"Good branch name 🎯",
	"That's a proper branch name ✓",
	"Anyone can tell what this branch does just by reading it 👀",
	"Clean branch, clean mind 🧠",
	"That name tells a whole story 📖",
	"Branch name on point 🔥",
	"This is how PRs stay organized",
	"Future-you won't be confused by this one",
}

var pushPraises = []string{
	"Pushed to a feature branch — that's the move ✓",
	"Keeping main clean 🔥",
	"PR flow respected 🤝",
	"That's how team players push code",
	"Feature branch push — good discipline ✓",
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

// GoodPushPraise fires when pushing to a non-main branch (keeping main clean).
type GoodPushPraise struct{}

func (r *GoodPushPraise) Name() string { return "good-push" }

func (r *GoodPushPraise) Evaluate(e *git.Event) *Praise {
	if e.Subcommand != "push" || e.IsForce {
		return nil
	}
	target := e.PushTarget
	if target == "" {
		target = e.Branch
	}
	if mainBranches[target] {
		return nil
	}
	return &Praise{
		Rule:    r.Name(),
		Message: pushPraises[rand.Intn(len(pushPraises))],
	}
}
