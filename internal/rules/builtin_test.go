package rules

import (
	"testing"

	"github.com/abdulqadirmsingi/pulse-cli/internal/git"
)

func TestForceMainRule(t *testing.T) {
	r := &ForceMainRule{}
	cases := []struct {
		name    string
		event   git.Event
		wantHit bool
	}{
		{"force push to main",        git.Event{Subcommand: "push", Branch: "main", IsForce: true}, true},
		{"force push to master",      git.Event{Subcommand: "push", Branch: "master", IsForce: true}, true},
		{"force push to feature",     git.Event{Subcommand: "push", Branch: "feat/login", IsForce: true}, false},
		{"normal push to main",       git.Event{Subcommand: "push", Branch: "main", IsForce: false}, false},
		{"commit on main (not push)", git.Event{Subcommand: "commit", Branch: "main", IsForce: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := r.Evaluate(&c.event)
			if (v != nil) != c.wantHit {
				t.Errorf("got violation=%v, want hit=%v", v, c.wantHit)
			}
			if v != nil && v.Severity != SeverityBlock {
				t.Errorf("expected SeverityBlock, got %d", v.Severity)
			}
		})
	}
}

func TestDirectMainRule(t *testing.T) {
	r := &DirectMainRule{}
	cases := []struct {
		name    string
		event   git.Event
		wantHit bool
	}{
		{"commit on main",        git.Event{Subcommand: "commit", Branch: "main"}, true},
		{"commit on master",      git.Event{Subcommand: "commit", Branch: "master"}, true},
		{"commit on feature",     git.Event{Subcommand: "commit", Branch: "feat/login"}, false},
		{"push on main (not commit)", git.Event{Subcommand: "push", Branch: "main"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := r.Evaluate(&c.event)
			if (v != nil) != c.wantHit {
				t.Errorf("got violation=%v, want hit=%v", v, c.wantHit)
			}
			if v != nil && v.Severity != SeverityWarn {
				t.Errorf("expected SeverityWarn, got %d", v.Severity)
			}
		})
	}
}

func TestBranchNameRule(t *testing.T) {
	r := &BranchNameRule{}
	cases := []struct {
		name    string
		event   git.Event
		wantHit bool
	}{
		{"vague: fix",               git.Event{Subcommand: "checkout", Args: []string{"-b", "fix"}}, true},
		{"vague: test",              git.Event{Subcommand: "checkout", Args: []string{"-b", "test"}}, true},
		{"vague: wip",               git.Event{Subcommand: "checkout", Args: []string{"-b", "wip"}}, true},
		{"good: feat/login",         git.Event{Subcommand: "checkout", Args: []string{"-b", "feat/login"}}, false},
		{"good: fix/null-auth",      git.Event{Subcommand: "checkout", Args: []string{"-b", "fix/null-auth"}}, false},
		{"checkout without -b",      git.Event{Subcommand: "checkout", Args: []string{"main"}}, false},
		{"switch with -b good",      git.Event{Subcommand: "switch", Args: []string{"-b", "feat/x"}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := r.Evaluate(&c.event)
			if (v != nil) != c.wantHit {
				t.Errorf("got violation=%v, want hit=%v", v, c.wantHit)
			}
		})
	}
}

func TestVagueCommitRule(t *testing.T) {
	r := &VagueCommitRule{}
	cases := []struct {
		name    string
		event   git.Event
		wantHit bool
	}{
		{"vague: update",                     git.Event{Subcommand: "commit", Message: "update"}, true},
		{"vague: fix",                        git.Event{Subcommand: "commit", Message: "fix"}, true},
		{"vague: wip",                        git.Event{Subcommand: "commit", Message: "wip"}, true},
		{"too short: ok",                     git.Event{Subcommand: "commit", Message: "ok"}, true},
		{"good: feat: add login flow",        git.Event{Subcommand: "commit", Message: "feat: add login flow"}, false},
		{"good: fix: nil panic in auth",      git.Event{Subcommand: "commit", Message: "fix: nil panic in auth"}, false},
		{"no message (amend etc)",            git.Event{Subcommand: "commit", Message: ""}, false},
		{"not a commit",                      git.Event{Subcommand: "push", Message: "update"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := r.Evaluate(&c.event)
			if (v != nil) != c.wantHit {
				t.Errorf("got violation=%v, want hit=%v", v, c.wantHit)
			}
		})
	}
}
