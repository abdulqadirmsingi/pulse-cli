package rules

import "github.com/devpulse-cli/devpulse/internal/git"

type Severity int

const (
	SeverityWarn  Severity = iota // show message, never block
	SeverityBlock                 // block execution (Phase 4, opt-in only)
)

// Violation is a single rule breach with context for the user.
type Violation struct {
	Severity Severity
	Rule     string // stable identifier, e.g. "force-push-main"
	Message  string // one short line — what went wrong
	Fix      string // optional: actionable next step
}

// Rule is the interface every git rule must satisfy.
// Evaluate returns nil when the event is clean.
type Rule interface {
	Name() string
	Evaluate(e *git.Event) *Violation
}

// Engine holds a set of rules and evaluates them in order.
type Engine struct {
	rules []Rule
}

// Default returns an Engine pre-loaded with every built-in rule.
func Default() *Engine {
	return &Engine{rules: []Rule{
		&ForceMainRule{},
		&DirectMainRule{},
		&DirectPushMainRule{},
		&BranchNameRule{},
		&VagueCommitRule{},
		&ConventionalCommitRule{},
		&FridayAfternoonRule{},
		&EmptyMergeMessageRule{},
	}}
}

// Evaluate runs all rules against ev and returns every violation found.
func (eng *Engine) Evaluate(ev *git.Event) []Violation {
	var out []Violation
	for _, r := range eng.rules {
		if v := r.Evaluate(ev); v != nil {
			out = append(out, *v)
		}
	}
	return out
}
