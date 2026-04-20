package rules

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/devpulse-cli/devpulse/internal/git"
)

// ConventionalCommitRule warns if the commit message doesn't follow
// the conventional commit format: type(scope): description
// Good: "feat: add login", "fix(auth): nil panic", "chore: update deps"
// Bad:  "added login", "fixed it", "updates"
type ConventionalCommitRule struct{}

func (r *ConventionalCommitRule) Name() string { return "conventional-commit" }

var conventionalPrefixes = []string{
	"feat:", "fix:", "chore:", "docs:", "style:", "refactor:",
	"test:", "perf:", "ci:", "build:", "revert:",
	"feat(", "fix(", "chore(", "docs(", "style(", "refactor(",
	"test(", "perf(", "ci(", "build(", "revert(",
}

func (r *ConventionalCommitRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "commit" || e.Message == "" {
		return nil
	}
	msg := strings.TrimSpace(e.Message)
	// skip very short messages — VagueCommitRule already handles those
	if utf8.RuneCountInString(msg) < 8 {
		return nil
	}
	lower := strings.ToLower(msg)
	for _, p := range conventionalPrefixes {
		if strings.HasPrefix(lower, p) {
			return nil
		}
	}
	return &Violation{
		Severity: SeverityWarn,
		Rule:     r.Name(),
		Message:  "commit message doesn't follow conventional format",
		Fix:      "use: feat: / fix: / chore: / docs: / refactor: / test:",
	}
}

// FridayAfternoonRule is a soft nudge when pushing on Friday after 4pm.
// Not a block — just a reminder that weekend incidents are no fun.
type FridayAfternoonRule struct{}

func (r *FridayAfternoonRule) Name() string { return "friday-afternoon" }

func (r *FridayAfternoonRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "push" {
		return nil
	}
	now := time.Now()
	if now.Weekday() != time.Friday || now.Hour() < 16 {
		return nil
	}
	return &Violation{
		Severity: SeverityWarn,
		Rule:     r.Name(),
		Message:  "pushing on a Friday afternoon — make sure this is well tested",
		Fix:      "if it can wait until Monday, it probably should",
	}
}

// DirectPushMainRule warns when pushing directly to main/master.
// This is different from ForceMainRule — even a normal push to main
// bypasses code review if the team uses PRs.
type DirectPushMainRule struct{}

func (r *DirectPushMainRule) Name() string { return "direct-push-main" }

func (r *DirectPushMainRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "push" || e.IsForce {
		return nil // force push is handled by ForceMainRule
	}
	target := e.PushTarget
	if target == "" {
		target = e.Branch
	}
	if !mainBranches[target] {
		return nil
	}
	return &Violation{
		Severity: SeverityWarn,
		Rule:     r.Name(),
		Message:  "pushing directly to " + target + " — consider opening a PR instead",
		Fix:      "git checkout -b feat/your-change, push that, then open a PR",
	}
}

// EmptyMergeMessageRule warns on `git merge` without a descriptive message.
// Merge commits are permanent history — a good message matters.
type EmptyMergeMessageRule struct{}

func (r *EmptyMergeMessageRule) Name() string { return "merge-message" }

func (r *EmptyMergeMessageRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "merge" {
		return nil
	}
	// --no-ff without -m means git will open an editor — that's fine.
	// --squash without -m means git stages but doesn't commit — also fine.
	// warn only when there are no args at all (bare `git merge`)
	if len(e.Args) == 0 {
		return &Violation{
			Severity: SeverityWarn,
			Rule:     r.Name(),
			Message:  "bare `git merge` with no branch specified",
			Fix:      "specify the branch: git merge feat/your-feature",
		}
	}
	return nil
}
