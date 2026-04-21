package rules

import (
	"math/rand"
	"strings"

	"github.com/abdulqadirmsingi/pulse-cli/internal/git"
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
	"Pulse approves this commit message 🤌",
	"Your teammates just smiled reading that commit 😊",
	"Pulse has seen 10,000 commits — this one passes 🎯",
	"Somewhere a senior dev shed a single tear of joy",
	"Future-you will actually know what happened here — rare",
	"Your code reviewer's prayers have been answered",
	"Your git log is going to look so clean because of this 📖",
	"Clean, clear, conventional — the holy trinity ✅",
	"Code archaeologists in 2040 will understand this. Respect.",
	"Pulse clocked the conventional format. Keep going 📊",
	"No 'wip wip wip' detected — you're evolving 📈",
	"That commit message could honestly be in a textbook fr",
	"Pulse is silently judging everyone else's commits rn",
	"The kind of message that makes PRs a joy to review 🚀",
	"Your future self just sent you a thank you note",
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
	"Pulse can tell what this branch does without reading a single line of code 👀",
	"Anyone opening this PR will immediately know what's going on",
	"Feat/fix/chore discipline — Pulse respects it 🫡",
	"A branch name so good it barely needs a PR description",
	"Your branch naming is so clean it's suspicious",
	"Future-you won't spend 10 minutes wondering what this was",
}

var pushPraises = []string{
	"Pushed to a feature branch — that's the move ✓",
	"Keeping main clean 🔥",
	"PR flow respected 🤝",
	"That's how team players push code",
	"Feature branch push — good discipline ✓",
	"Keeping main clean — Pulse clocked that 👀",
	"Main is sacred and you already know it 🙏",
	"Feature branch push, exactly how it's supposed to go ✅",
	"Somewhere a CI pipeline is about to be very happy",
	"PR flow intact, team flow intact — Pulse approves 🤝",
	"Main stays untouched. Pulse respects that discipline 🔥",
	"Clean push to a feature branch — this is the way 🚀",
}

var goodBranchPrefixes = []string{
	"feat/", "fix/", "chore/", "docs/", "refactor/",
	"test/", "perf/", "ci/", "build/", "hotfix/",
}

type GoodCommitPraise struct{}

func (r *GoodCommitPraise) Name() string { return "good-commit" }

func (r *GoodCommitPraise) Evaluate(e *git.Event) *Praise {
	if e.Subcommand != "commit" || e.Message == "" {
		return nil
	}
	msg := strings.TrimSpace(e.Message)
	// only praise when the case is also correct — wrong-case gets a violation instead
	for _, p := range conventionalPrefixes {
		if strings.HasPrefix(msg, p) {
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
