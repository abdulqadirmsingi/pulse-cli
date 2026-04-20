// Package insights is a rule engine for developer analytics — no external API, just patterns.
package insights

import (
	"fmt"
	"strings"

	"github.com/devpulse-cli/devpulse/internal/db"
)

// Level controls the icon and tone of an insight.
// 🧠 Go Lesson #46: `type Level int` is a named integer type — compiler prevents
// you from passing a plain int where Level is expected. iota auto-increments.
type Level int

const (
	LevelFire    Level = iota // 🔥 something great is happening
	LevelGood                 // ✅ solid, keep it up
	LevelHeadsUp              // ⚠️  something worth noticing
	LevelRoast                // 💀 gentle call-out
	LevelTip                  // 💡 actionable suggestion
)

func (l Level) Icon() string {
	switch l {
	case LevelFire:
		return "🔥"
	case LevelGood:
		return "✅"
	case LevelHeadsUp:
		return "⚠️ "
	case LevelRoast:
		return "💀"
	default:
		return "💡"
	}
}

// Insight is a single observation or tip.
type Insight struct {
	Level   Level
	Message string
}

// Report bundles all insights for display.
type Report struct {
	Observations []Insight
	Tips         []Insight
}

// Analyse runs every rule and returns a Report.
// 🧠 Go Lesson #46: No framework needed — a plain slice of function calls in
// order is the simplest possible "plugin" system in Go.
func Analyse(stats *db.Stats, topCmds, topProjects []db.TopEntry) Report {
	var r Report
	if stats == nil {
		return r
	}
	r.obs(streakRule(stats.StreakDays))
	r.obs(successRule(stats.SuccessRate))
	r.obs(timeRule(stats.TotalTimeMS))
	if len(topCmds) > 0 {
		r.obs(topCommandRule(topCmds[0].Name))
		r.obs(toolStackRule(topCmds))
	}
	if len(topProjects) > 0 {
		r.obs(projectFocusRule(topProjects))
	}
	r.tips(commandTips(topCmds)...)
	return r
}

func (r *Report) obs(i Insight) {
	if i.Message != "" {
		r.Observations = append(r.Observations, i)
	}
}

func (r *Report) tips(items ...Insight) {
	for _, i := range items {
		if i.Message != "" {
			r.Tips = append(r.Tips, i)
		}
	}
}
func streakRule(days int) Insight {
	switch {
	case days == 0:
		return Insight{LevelRoast, "no active streak — open ur terminal bestie 💀"}
	case days >= 30:
		return Insight{LevelFire, fmt.Sprintf("%d day streak — bro is literally a machine 🤖", days)}
	case days >= 14:
		return Insight{LevelFire, fmt.Sprintf("%d day streak — ur literally built different rn", days)}
	case days >= 7:
		return Insight{LevelFire, fmt.Sprintf("%d day streak — on fire fr fr", days)}
	default:
		return Insight{LevelGood, fmt.Sprintf("%d day streak — just getting started, keep pushing", days)}
	}
}

func successRule(rate float64) Insight {
	switch {
	case rate >= 97:
		return Insight{LevelFire, fmt.Sprintf("%.1f%% success rate — typing with purpose fr", rate)}
	case rate >= 90:
		return Insight{LevelGood, fmt.Sprintf("%.1f%% success rate — clean execution, solid", rate)}
	case rate >= 80:
		return Insight{LevelHeadsUp, fmt.Sprintf("%.1f%% success rate — some commands flopping ngl", rate)}
	default:
		return Insight{LevelRoast, fmt.Sprintf("%.1f%% success rate — ur failing a lot, slow down", rate)}
	}
}
func timeRule(ms int64) Insight {
	hours := ms / 1000 / 60 / 60
	switch {
	case hours >= 8:
		return Insight{LevelHeadsUp, "grinding hard — hydrate and take breaks bestie 🥤"}
	case hours >= 4:
		return Insight{LevelFire, "solid session, ur putting in real hours 🔥"}
	case ms < 60_000:
		return Insight{LevelRoast, "barely any time logged — just opened terminal to flex? 💀"}
	}
	return Insight{}
}

// topCommandRule personality-matches on the #1 command. 🧠 Go Lesson #47: map literal
// lookup beats a long switch when every case maps string→value with no shared logic.
func topCommandRule(top string) Insight {
	messages := map[string]Insight{
		"git":     {LevelGood, "git is ur most used command — certified dev behavior"},
		"vim":     {LevelFire, "vim user detected — absolute chad energy 🗿"},
		"nvim":    {LevelFire, "neovim user — ur that person at the coffee shop fr"},
		"cd":      {LevelRoast, "ur #1 is cd — bro just navigates 💀 try zoxide"},
		"ls":      {LevelRoast, "ls is ur #1 — just looking around aren't u 👀"},
		"python":  {LevelGood, "python dev spotted — snake gang 🐍"},
		"python3": {LevelGood, "python3 dev spotted — snake gang 🐍"},
		"go":      {LevelFire, "go dev, certified based 🐹"},
		"node":    {LevelGood, "node dev spotted in the wild"},
		"docker":  {LevelGood, "container enjoyer detected — cloud native era"},
		"kubectl": {LevelFire, "k8s navigator — ur in the deep end fr"},
		"make":    {LevelGood, "makefile enjoyer — old school but it works"},
	}
	if i, ok := messages[strings.ToLower(top)]; ok {
		return i
	}
	return Insight{LevelGood, fmt.Sprintf("ur most used tool is `%s` — niche but valid", top)}
}

func toolStackRule(cmds []db.TopEntry) Insight {
	names := make(map[string]bool)
	for _, c := range cmds {
		names[strings.ToLower(c.Name)] = true
	}
	hasJS := names["npm"] || names["yarn"] || names["pnpm"] || names["node"]
	hasGo, hasPy := names["go"], names["python"] || names["python3"] || names["pip"]
	switch {
	case hasJS && hasGo:
		return Insight{LevelGood, "go + js in ur top commands — full-stack era fr"}
	case hasJS:
		return Insight{LevelGood, "npm nation detected — js dev spotted 👀"}
	case hasPy && hasGo:
		return Insight{LevelGood, "python + go combo — polyglot dev behavior"}
	}
	return Insight{}
}
func projectFocusRule(projects []db.TopEntry) Insight {
	switch {
	case len(projects) == 1:
		return Insight{LevelFire, fmt.Sprintf("all in on `%s` — hyper focused era 🎯", projects[0].Name)}
	case len(projects) >= 5:
		return Insight{LevelHeadsUp, fmt.Sprintf("juggling %d projects — context switching is a tax, just saying", len(projects))}
	}
	return Insight{}
}

var tipCatalog = map[string]string{
	"git":    "running git a lot? `git status -s` and `git log --oneline` save time",
	"cd":     "check out `zoxide` (z) — it learns ur most visited dirs and jumps instantly",
	"ls":     "try `eza` (or `lsd`) for colour-coded ls with icons",
	"find":   "`fd` is a faster friendlier alternative to find",
	"grep":   "`rg` (ripgrep) is grep but 10x faster, try it",
	"cat":    "`bat` is cat with syntax highlighting — quality of life upgrade",
	"docker": "`lazydocker` is a TUI that saves you a lot of docker typing",
}

func commandTips(cmds []db.TopEntry) []Insight {
	var out []Insight
	seen := map[string]bool{}
	for _, c := range cmds {
		key := strings.ToLower(c.Name)
		if msg, ok := tipCatalog[key]; ok && !seen[key] {
			out = append(out, Insight{LevelTip, msg})
			seen[key] = true
		}
		if len(out) >= 3 {
			break
		}
	}
	return out
}
