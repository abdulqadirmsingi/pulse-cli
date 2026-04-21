package rules

import (
	"strings"
	"unicode/utf8"

	"github.com/abdulqadirmsingi/pulse-cli/internal/git"
)

var mainBranches = map[string]bool{"main": true, "master": true}

// ForceMainRule blocks force-pushing to main or master.
type ForceMainRule struct{}

func (r *ForceMainRule) Name() string { return "force-push-main" }

func (r *ForceMainRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "push" || !e.IsForce {
		return nil
	}
	// block if the explicit push target is main, OR if pushing from main with no target
	target := e.PushTarget
	if target == "" {
		target = e.Branch
	}
	if !mainBranches[target] {
		return nil
	}
	return &Violation{
		Severity: SeverityBlock,
		Rule:     r.Name(),
		Message:  "Force push to " + target + " — this rewrites shared history",
		Fix:      "Use --force-with-lease if you really must, or open a PR instead",
	}
}

// DirectMainRule warns when committing directly to main or master.
type DirectMainRule struct{}

func (r *DirectMainRule) Name() string { return "direct-main-commit" }

func (r *DirectMainRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "commit" {
		return nil
	}
	if !mainBranches[e.Branch] {
		return nil
	}
	return &Violation{
		Severity: SeverityWarn,
		Rule:     r.Name(),
		Message:  "Direct commit to " + e.Branch + " — consider a feature branch",
		Fix:      "What are you building? Try: git switch -c feat/your-change",
	}
}

// vagueNames are branch names that carry no information about the work.
var vagueNames = map[string]bool{
	"fix": true, "test": true, "update": true, "dev": true,
	"temp": true, "wip": true, "work": true, "stuff": true,
	"changes": true, "patch": true, "hotfix": true, "misc": true,
}

// BranchNameRule warns on vague or unstructured branch names.
type BranchNameRule struct{}

func (r *BranchNameRule) Name() string { return "branch-name" }

func (r *BranchNameRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "checkout" && e.Subcommand != "switch" {
		return nil
	}
	// only applies when creating a branch (-b or -B flag)
	if !hasCreateFlag(e.Args) {
		return nil
	}
	branch := newBranchName(e.Args)
	if branch == "" || mainBranches[branch] {
		return nil
	}
	if vagueNames[strings.ToLower(branch)] {
		return &Violation{
			Severity: SeverityWarn,
			Rule:     r.Name(),
			Message:  "Branch name \"" + branch + "\" is too vague",
			Fix:      "What are you working on? Try: feat/user-auth or fix/login-bug",
		}
	}
	return nil
}

// vagueMessages are single-word commit messages that mean nothing.
var vagueMessages = map[string]bool{
	"fix": true, "update": true, "wip": true, "changes": true,
	"stuff": true, "misc": true, "test": true, "temp": true,
	"patch": true, "work": true, "commit": true, "save": true,
	"done": true, "edit": true,
	// note: "refactor" and "cleanup" are valid conventional commit types — not vague
}

// VagueCommitRule warns on commit messages that carry no information.
type VagueCommitRule struct{}

func (r *VagueCommitRule) Name() string { return "vague-commit" }

func (r *VagueCommitRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "commit" || e.Message == "" {
		return nil
	}
	msg := strings.TrimSpace(e.Message)
	// too short — under 8 visible characters
	if utf8.RuneCountInString(msg) < 8 {
		return &Violation{
			Severity: SeverityWarn,
			Rule:     r.Name(),
			Message:  "Commit message \"" + msg + "\" is too short to be useful",
			Fix:      "What changed? Try: \"fix: prevent nil panic in auth handler\"",
		}
	}
	// exact match on known vague words (case-insensitive, ignores punctuation)
	clean := strings.ToLower(strings.Trim(msg, " .,!?"))
	if vagueMessages[clean] {
		return &Violation{
			Severity: SeverityWarn,
			Rule:     r.Name(),
			Message:  "Commit message \"" + msg + "\" tells future-you nothing",
			Fix:      "What changed? Try: \"fix: prevent nil panic in auth handler\"",
		}
	}
	return nil
}

func hasCreateFlag(args []string) bool {
	for _, a := range args {
		// -b/-B: git checkout; -c/-C: git switch (the modern form)
		if a == "-b" || a == "-B" || a == "-c" || a == "-C" || a == "--orphan" {
			return true
		}
	}
	return false
}

// newBranchName returns the new branch name from checkout/switch args.
// Handles both `git checkout -b feat/x` and `git switch -c feat/x`.
func newBranchName(args []string) string {
	for i, a := range args {
		if (a == "-b" || a == "-B" || a == "-c" || a == "-C") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// BranchConventionRule nudges users toward type-prefixed branch names when
// their branch doesn't follow the feat/fix/chore pattern but isn't just a
// vague single word (BranchNameRule handles that case separately).
type BranchConventionRule struct{}

func (r *BranchConventionRule) Name() string { return "branch-convention" }

func (r *BranchConventionRule) Evaluate(e *git.Event) *Violation {
	if e.Subcommand != "checkout" && e.Subcommand != "switch" {
		return nil
	}
	if !hasCreateFlag(e.Args) {
		return nil
	}
	branch := newBranchName(e.Args)
	if branch == "" || mainBranches[branch] {
		return nil
	}
	lower := strings.ToLower(branch)

	// already follows convention — GoodBranchPraise handles praise
	for _, p := range goodBranchPrefixes {
		if strings.HasPrefix(lower, p) {
			return nil
		}
	}
	// has a slash structure → user is trying to follow convention, don't nag
	if strings.Contains(branch, "/") {
		return nil
	}
	// vague single words → BranchNameRule handles those
	if vagueNames[lower] {
		return nil
	}

	suggestion := suggestBranchPrefix(branch)
	return &Violation{
		Severity: SeverityWarn,
		Rule:     r.Name(),
		Message:  `"` + branch + `" is missing a type prefix`,
		Fix:      "how about: " + suggestion,
	}
}

// suggestBranchPrefix guesses the right conventional prefix for a branch name
// and returns a full suggestion like "fix/login-bug".
func suggestBranchPrefix(branch string) string {
	lower := strings.ToLower(branch)
	slug := toBranchSlug(branch)

	words := strings.FieldsFunc(lower, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}

	prefix := "feat"
	switch {
	case wordSet["fix"] || wordSet["bug"] || wordSet["bugfix"] || wordSet["patch"]:
		prefix = "fix"
	case wordSet["hotfix"] || wordSet["urgent"]:
		prefix = "hotfix"
	case wordSet["doc"] || wordSet["docs"] || wordSet["readme"]:
		prefix = "docs"
	case wordSet["chore"] || wordSet["cleanup"] || wordSet["bump"] || wordSet["deps"]:
		prefix = "chore"
	case wordSet["test"] || wordSet["spec"]:
		prefix = "test"
	case wordSet["refactor"] || wordSet["rewrite"]:
		prefix = "refactor"
	case wordSet["perf"] || wordSet["performance"]:
		prefix = "perf"
	}

	// strip redundant leading type word (e.g. "fix-login-bug" → "fix/login-bug")
	if strings.HasPrefix(slug, prefix+"-") {
		slug = slug[len(prefix)+1:]
	}
	return prefix + "/" + slug
}

// toBranchSlug converts a branch name to clean kebab-case.
func toBranchSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if (r == '-' || r == '_' || r == ' ') && !prevHyphen && b.Len() > 0 {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
