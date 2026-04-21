package rules

import "github.com/abdulqadirmsingi/pulse-cli/internal/git"

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

// Praise is positive feedback when the user does something right.
type Praise struct {
	Rule    string
	Message string
}

// Rule is the interface every git rule must satisfy.
// Evaluate returns nil when the event is clean.
type Rule interface {
	Name() string
	Evaluate(e *git.Event) *Violation
}

// PraiseRule fires when the user does something worth acknowledging.
type PraiseRule interface {
	Name() string
	Evaluate(e *git.Event) *Praise
}

// Engine holds a set of rules and evaluates them in order.
type Engine struct {
	rules       []Rule
	praiseRules []PraiseRule
}

// Default returns an Engine pre-loaded with every built-in rule.
func Default() *Engine {
	return &Engine{
		rules: []Rule{
			&ForceMainRule{},
			&DirectMainRule{},
			&DirectPushMainRule{},
			&BranchNameRule{},
			&BranchConventionRule{},
			&VagueCommitRule{},
			&ConventionalCommitRule{},
			&FridayAfternoonRule{},
			&EmptyMergeMessageRule{},
		},
		praiseRules: []PraiseRule{
			&GoodCommitPraise{},
			&GoodBranchPraise{},
			&GoodPushPraise{},
		},
	}
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

// EvaluatePraise runs praise rules and returns any positive feedback.
func (eng *Engine) EvaluatePraise(ev *git.Event) []Praise {
	var out []Praise
	for _, r := range eng.praiseRules {
		if p := r.Evaluate(ev); p != nil {
			out = append(out, *p)
		}
	}
	return out
}
