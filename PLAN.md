# Pulse — Git Discipline Engine: Implementation Plan

> All decisions in this plan are constrained by `RULES.md`.
> No file > 200 lines. All business logic in `internal/`. One package per concern.
> Next Go Lesson: #52.

---

## Current State (what already exists)

```
cmd/            → cobra commands (log, stats, vibe, init, doctor, dash, today, projects, reset, update, uninstall)
internal/
  config/       → paths, AppVersion
  db/           → all SQL (commands table, Stats, InsertCommand, GetStats, GetTopCommands, GetTopProjects)
  insights/     → rule engine for dev analytics (streak, success rate, tool stack)
  ui/           → lipgloss styles, ProgressBar, FormatDuration, FormatNumber, Truncate
  tui/          → Bubble Tea model for `pulse dash`
```

The shell hook fires `pulse log` **after** every command (postcmd/precmd). It is async and backgrounded — zero terminal latency.

---

## Architecture Decision: Interception Strategy

**Honest assessment of each option:**

| Approach | Latency | Blocks? | Reliability | Verdict |
|---|---|---|---|---|
| Post-execution async (current hook) | 0ms | No | ✅ Solid | Use for analytics + soft warnings |
| preexec blocking check | +10–30ms | Yes | ⚠️ Risky — slows every command | Only for high-risk git ops |
| Shell function wrapping `git` | +5–15ms | Yes | ✅ Reliable, opt-in | Best for blocking rules |
| Proxy binary replaces `git` | +5ms | Yes | ❌ Breaks aliases, PATH issues | Bad idea — don't do it |

**Decision:**
- Phases 1–3 use **post-execution only** (extends the existing hook, zero new latency).
- Phase 4 introduces an **opt-in shell wrapper** for `git` that is disabled by default. Users enable it with `pulse git-guard on`. It adds ~10ms to git commands only.
- Never slow non-git commands. Never block unless the user opts in.

---

## Package Structure After All Phases

```
internal/
  config/       → unchanged
  db/           → add git_events table (Phase 1), branch_snapshots table (Phase 4)
  insights/     → unchanged (general dev analytics)
  git/          → NEW: git command parsing, git context extraction
  rules/        → NEW: rule engine interface + built-in rule set
  ui/           → unchanged
  tui/          → unchanged
cmd/
  git.go        → NEW: `pulse git` command (git health dashboard)
  gitguard.go   → NEW: `pulse git-guard on|off|status` (Phase 4)
```

**Why `internal/git` and `internal/rules` are separate packages:**
- `internal/git` owns git-specific parsing and metadata — it knows what `git push --force` means.
- `internal/rules` owns the rule engine interface and evaluation loop — it doesn't care whether the input came from git or elsewhere.
- Neither imports the other at the top level. `cmd/` wires them together.

---

## Phase 1 — Git Event Parsing & Storage

**Goal:** When the shell hook logs a command, detect if it's a git command, parse it, and store structured git metadata in a new table.

**New file: `internal/git/parse.go`** (~80 lines)

```go
package git

// Event represents a parsed git command captured from the shell hook.
type Event struct {
    Subcommand string   // "commit", "push", "checkout", etc.
    Args       []string // raw args after subcommand
    Branch     string   // current branch at time of command (read from .git/HEAD)
    IsForce    bool     // --force or -f present
    Remote     string   // "origin", "upstream", etc. if present
    Dir        string   // working directory
}

// Parse returns an Event from a raw command string and working directory.
// Returns nil if the command is not a git command.
func Parse(cmd, dir string) *Event
```

**New file: `internal/git/context.go`** (~60 lines)

```go
package git

// BranchFromDir reads the current branch by parsing .git/HEAD directly.
// Never shells out — no exec.Command, no latency.
// Returns "" if dir is not inside a git repo.
func BranchFromDir(dir string) string

// RepoRoot walks up from dir to find the .git directory.
// Returns "" if not in a git repo.
func RepoRoot(dir string) string
```

> **Why parse `.git/HEAD` directly instead of running `git branch`?**
> `exec.Command("git", "branch")` adds 20–50ms. Reading a 30-byte file takes ~0.1ms.

**DB change: `internal/db/db.go`** — add `git_events` table in `migrate()`:

```sql
CREATE TABLE IF NOT EXISTS git_events (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    command_id  INTEGER  REFERENCES commands(id),
    subcommand  TEXT     NOT NULL,
    branch      TEXT     NOT NULL DEFAULT '',
    is_force    INTEGER  NOT NULL DEFAULT 0,
    remote      TEXT     NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_git_branch   ON git_events(branch);
CREATE INDEX IF NOT EXISTS idx_git_created  ON git_events(created_at);
```

**Change: `cmd/log.go`** — after `InsertCommand`, if command is a git command, parse and insert a git event. Still async, still silent.

**New method: `internal/db/db.go`**

```go
func (db *DB) InsertGitEvent(commandID int64, e *git.Event) error
func (db *DB) GetGitEvents(days int) ([]GitEvent, error)
```

### ✋ Phase 1 Review Checkpoint

Before moving to Phase 2, verify:
- [ ] `git_events` table is created on fresh install and on existing DBs (ALTER TABLE migration)
- [ ] `git parse` correctly extracts subcommand, branch, force flag for: `git commit -m "msg"`, `git push --force`, `git checkout -b feature/x`, `git push origin main`
- [ ] `cmd/log.go` stays under 200 lines — split into `log_git.go` if needed
- [ ] `pulse doctor` still passes
- [ ] No latency added to non-git commands
- [ ] Run `pulse log --cmd "git push --force" --exit 0 --ms 1200 --dir $(pwd)` and confirm row appears in `git_events`

**Report format:** List each file changed, lines added, and confirm all checkboxes pass.

---

## Phase 2 — Rule Engine

**Goal:** A typed, extensible rule system that evaluates git events and returns feedback. Rules are pure functions — no I/O, no DB calls inside a rule.

**New file: `internal/rules/rule.go`** (~60 lines)

```go
package rules

import "github.com/devpulse-cli/devpulse/internal/git"

// Severity mirrors insights.Level but is git-specific.
// 🧠 Go Lesson #52: Define domain-specific types even when they mirror
// existing ones — it keeps packages decoupled and the compiler enforces intent.
type Severity int

const (
    SeverityWarn  Severity = iota // show message, don't block
    SeverityBlock                 // block execution (Phase 4 only, opt-in)
)

// Violation is a rule breach with context.
type Violation struct {
    Severity Severity
    Rule     string // rule name, e.g. "branch-name"
    Message  string // human-readable, actionable, short
    Fix      string // optional: "try: git checkout -b feat/your-feature"
}

// Rule is the interface every git rule implements.
// Evaluate returns nil if the event is clean.
type Rule interface {
    Name() string
    Evaluate(e *git.Event) *Violation
}
```

**New file: `internal/rules/builtin.go`** (~120 lines)

Built-in rules, each under 20 lines:

```go
// BranchNameRule — warns on vague branch names.
// Bad: "fix", "test", "update", "dev", "temp", "wip"
// Good: "feat/login-flow", "fix/null-pointer-auth"
type BranchNameRule struct{}

// ForceMainRule — warns (or blocks) on `git push --force` to main/master.
type ForceMainRule struct{}

// DirectMainCommitRule — detects `git commit` while on main/master.
type DirectMainCommitRule struct{}

// VagueCommitRule — detects commit messages matching noise patterns.
// Evaluated post-execution by parsing the git event args.
// Bad: "update", "fix", "wip", "asdf", single-word messages under 5 chars.
type VagueCommitRule struct{}
```

**New file: `internal/rules/engine.go`** (~50 lines)

```go
package rules

// Engine holds a slice of rules and evaluates all of them.
type Engine struct {
    rules []Rule
}

// Default returns an Engine pre-loaded with all built-in rules.
func Default() *Engine

// Evaluate runs all rules against the event and returns every violation found.
// Stops at the first SeverityBlock violation (Phase 4).
func (e *Engine) Evaluate(ev *git.Event) []Violation
```

**Design note:** Rules are registered at startup via `Default()`. Making rules configurable (user can disable a rule in `~/.devpulse/rules.toml`) is Phase 5 scope. Do not design for it now — add the config hook when you add the second implementation.

### ✋ Phase 2 Review Checkpoint

Before moving to Phase 3, verify:
- [ ] `internal/rules/rule.go` compiles, `Rule` interface is satisfied by all built-in rules
- [ ] Unit tests for each built-in rule: test at least one passing case and one violation case
- [ ] `BranchNameRule` correctly passes `feat/login` and flags `fix`, `test`, `update`
- [ ] `ForceMainRule` only triggers on `git push --force` when branch is `main` or `master`
- [ ] `VagueCommitRule` flags `"update"`, `"fix"`, `"wip"` and passes `"feat: add login flow"`
- [ ] No rule does I/O — all inputs come from the `git.Event` struct
- [ ] `internal/rules/` package does not import `cmd/` or `internal/db/`

**Report format:** Show each rule name, a passing test case, and a failing test case. Confirm no circular imports.

---

## Phase 3 — Real-Time Post-Execution Feedback

**Goal:** After a git command runs, if there are violations, print them to the terminal. No blocking. No annoyance. One line per violation max.

**How it works:**
The shell hook runs `pulse log` in the background with output redirected to `/dev/null`. We change this for git commands only: run synchronously (still fast, just not backgrounded) and print violations to stderr.

**Hook change (zsh):**

```zsh
_pulse_precmd() {
    local _exit=$?
    [ -z "$_PULSE_CMD" ] && return
    local _ms=$(( ($(date +%s) - ${_PULSE_CMD_START:-0}) * 1000 ))
    # git commands run sync so feedback can reach the terminal
    case "$_PULSE_CMD" in
        git\ *)
            /path/to/pulse log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" 2>&1
            ;;
        *)
            /path/to/pulse log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" >/dev/null 2>&1 &|
            ;;
    esac
    unset _PULSE_CMD _PULSE_CMD_START
}
```

> **Latency reality check:** `pulse log` for a git command does: open DB (~2ms), insert row (~1ms), parse git event (~0.1ms), evaluate rules (~0.1ms), print. Total: ~5ms. Imperceptible.

**Change: `cmd/log.go`** — after inserting the git event, evaluate rules and print violations to stderr:

```go
if violations := engine.Evaluate(gitEvent); len(violations) > 0 {
    for _, v := range violations {
        printViolation(v) // writes to os.Stderr
    }
}
```

**UX rules for feedback:**
- Max 2 lines per violation: icon + message on line 1, fix hint on line 2 (indented, muted)
- Use `⚠️` for warn, `🚫` for block
- Never print if exit code != 0 (command already failed — don't pile on)
- Never print for the same violation twice in 60 seconds (cooldown map in memory)

**Example output:**
```
⚠️  direct commit to main — consider a feature branch
   try: git checkout -b feat/your-change && git cherry-pick HEAD
```

### ✋ Phase 3 Review Checkpoint

Before moving to Phase 4, verify:
- [ ] Feedback only prints for git commands, not `ls`, `cd`, `npm install`, etc.
- [ ] Feedback never prints when the git command itself failed (exit != 0)
- [ ] Output goes to stderr (doesn't pollute stdout piped output)
- [ ] The 60-second cooldown works — run the same bad command twice fast, only one warning
- [ ] `pulse log` still takes < 20ms for git commands (measure with `time pulse log --cmd "git push --force" --exit 0 --ms 0 --dir $(pwd)`)
- [ ] The hook change is applied correctly by `pulse init --reinstall`
- [ ] Run `pulse doctor` — still passes

**Report format:** Show the exact terminal output produced by: (1) `git commit -m "fix"` on main, (2) `git push --force` to main, (3) `git checkout -b feat/login`. Confirm timing.

---

## Phase 4 — Opt-In Pre-Execution Blocking (Git Guard)

**Goal:** For high-stakes operations only, offer a shell wrapper around `git` that can ask for confirmation or outright block. Disabled by default. One command to enable.

**New command: `pulse git-guard on|off|status`**
- `on` → adds a `git()` shell function to `~/.zshrc` above the Pulse hook
- `off` → removes it
- `status` → shows which rules are in block mode

**Shell function (added by `pulse git-guard on`):**

```zsh
git() {
    /path/to/pulse git-check "$@"
    local _pulse_exit=$?
    [ $_pulse_exit -ne 0 ] && return $_pulse_exit
    command git "$@"
}
```

**New command: `pulse git-check [git args...]`**
- Parses the incoming git args as a `git.Event`
- Runs `SeverityBlock` rules only
- Exits 0 if clean (git proceeds), exits 1 if blocked (git never runs)
- Prints the reason to stderr

**Which rules are block-level (non-negotiable):**
- `git push --force` to `main`/`master` → blocked, always
- `git push` directly to `main` without a PR → warn only (can't know PR status locally)

**Everything else stays warn-only.** Block sparingly — every false positive trains users to disable the guard.

**New file: `cmd/gitguard.go`** (~80 lines)
**New file: `cmd/gitcheck.go`** (~60 lines, hidden command like `log`)

### ✋ Phase 4 Review Checkpoint

Before moving to Phase 5, verify:
- [ ] `pulse git-guard on` adds the shell function and `pulse git-guard off` cleanly removes it
- [ ] `pulse git-guard on` is idempotent (running it twice doesn't add the function twice)
- [ ] `command git "$@"` is used inside the wrapper — no infinite recursion
- [ ] `pulse git-check push --force origin main` exits 1 and prints a clear block message
- [ ] `pulse git-check commit -m "feat: add login"` exits 0 silently
- [ ] `pulse git-guard off` restores normal git behavior immediately (no shell restart needed)
- [ ] `pulse doctor` reports git-guard status (on/off)

**Report format:** Show `pulse git-guard on`, then `git push --force origin main` output, then `pulse git-guard off` output. Confirm `command git` fallback works.

---

## Phase 5 — Git Health Analytics

**Goal:** New metrics computed from `git_events` and exposed via `pulse git` command.

**Metrics to compute:**

### Commit Health Score (0–100)
```
score = 100
- deduct 20 if > 30% of commit messages are vague (< 10 chars or match noise patterns)
- deduct 20 if any force-pushes to main in the last 30 days
- deduct 15 if average commits per day > 20 (spray-and-pray committing)
- deduct 15 if < 1 commit per active day (infrequent commits = large risky batches)
- add 10 if all branch names follow a pattern (feat/, fix/, chore/)
```

### Branch Discipline Score (0–100)
```
score = 100
- deduct 30 if any direct commits to main
- deduct 20 if branches older than 7 days with no merge/delete
- deduct 10 for each vague branch name (fix, test, temp, dev)
- add 10 if branch lifespan average < 3 days (fast, focused work)
```

**New methods in `internal/db/analytics.go`:**
```go
func (db *DB) GetGitHealthData(days int) (*GitHealthData, error)
// returns: commit messages, branch names, force push count, direct main commits
```

**New file: `internal/git/score.go`** (~80 lines)
```go
func CommitHealthScore(data *db.GitHealthData) int
func BranchDisciplineScore(data *db.GitHealthData) int
```

**New command: `cmd/git.go`** — `pulse git` shows:
```
🔀  git health  ·  last 30 days

╭──────────────────────────────────────────╮
│  📝  commit health     82 / 100          │
│  🌿  branch discipline  91 / 100         │
│  🚫  force pushes       0                │
│  ⚡  direct main commits 2               │
╰──────────────────────────────────────────╯

  📝  commit breakdown
  good messages    ████████░░  82%
  vague messages   ██░░░░░░░░  18%

  🌿  recent branches
  feat/noise-commands   ████  2 days  ✅ merged
  fix                   ██░░  4 days  ⚠️  vague name
```

### ✋ Phase 5 Review Checkpoint

Before shipping:
- [ ] `CommitHealthScore` and `BranchDisciplineScore` are pure functions — no DB calls inside them
- [ ] Score deductions are additive and capped at 0 (never go negative)
- [ ] `pulse git` renders correctly with 0 git events (shows "start using git guard to track git health")
- [ ] `internal/git/score.go` is under 200 lines
- [ ] `cmd/git.go` is under 200 lines — split into `cmd/git_render.go` if needed
- [ ] `pulse vibe` and `pulse stats` are unaffected

**Report format:** Run `pulse git` and paste the full output. Show score calculation for your actual data.

---

## Differentiation Ideas (Beyond Basic Git Tools)

These are things no other local CLI tool does:

**1. Commit Rhythm Detection**
Track the time between commits within a session. If a developer commits 12 times in 10 minutes, that's spray-and-pray. If they commit once every 2 hours on a complex feature, that's thoughtful. Show this in `pulse git` as "commit rhythm: thoughtful / spray-and-pray / inactive."

**2. "Context Switch Tax" Score**
Measure how often a developer switches between branches mid-session. Every checkout is a potential context switch. More than 3 branch switches in an hour is a focus problem. This is computable from `git_events` alone.

**3. Branch Lifespan Heatmap**
In `pulse dash`, show a visual of branch lifespans. Short-lived branches (< 3 days) are green. Long-lived branches (> 14 days) are red. This visualizes merge debt without connecting to GitHub.

**4. "The Friday Rule" Detection**
Detect `git push` events on Fridays after 4pm. Print a single line: `⚠️  pushing on a Friday afternoon — bold move`. Not a block, just acknowledgement. Memorable, shareable, keeps the tool's personality intact.

**5. Commit Message Improvement Suggestions**
When `VagueCommitRule` triggers, don't just say "bad message." Parse the diff context from `git diff --staged --stat` (fast, no network) and suggest a better message format. Example: if 3 Go files changed, suggest `"feat(auth): ..."` over `"update"`.

---

## Performance Constraints Summary

| Operation | Where | Budget | Strategy |
|---|---|---|---|
| Non-git command logging | postcmd, async | 0ms visible | Already backgrounded |
| Git command logging + rule eval | postcmd, sync | < 20ms | Read `.git/HEAD` directly, no exec.Command |
| Pre-execution block check (Phase 4) | preexec, sync | < 15ms | Only `SeverityBlock` rules, no DB read |
| `pulse git` dashboard | user-invoked | < 200ms | Single SQL query with aggregation |
| Score computation | user-invoked | < 5ms | Pure functions over in-memory structs |

**No goroutines in `pulse log`.** The command is already a background process — adding goroutines inside it creates zombie processes. Keep it synchronous and fast.

---

## Go Lessons Index Update

| Lesson | Topic | File |
|---|---|---|
| #52 | Domain-specific types even when mirroring existing ones | `internal/rules/rule.go` |
| #53 | Parsing files directly vs shelling out (`.git/HEAD`) | `internal/git/context.go` |
| #54 | Pure functions for scoring — no I/O, deterministic, testable | `internal/git/score.go` |
| #55 | `errors.Is` for sentinel errors vs type assertions | `cmd/update.go` (already added) |

---

## What Would Be a Bad Idea

**Don't build a language model integration** to suggest better commit messages. It breaks the offline constraint, adds a dependency, and requires an API key. The rule-based suggestions in Phase 5 are good enough and work offline.

**Don't build a `git` proxy binary** that replaces the real git. It breaks too many tools (IDEs, CI scripts, aliases) and is a support nightmare. Shell function wrapping (Phase 4) is the right level of intervention.

**Don't track commit message quality by running `git log`** on every command. `git log` takes 50–200ms on large repos. Parse commit messages from the `pulse log` args instead — the shell hook already captures the full command string including `-m "message"`.

**Don't add a `--no-pulse` flag to git commands.** If your tool requires users to escape it, the tool is too aggressive. Tune the rules instead.
