```
  ██████╗ ██╗   ██╗██╗     ███████╗███████╗
  ██╔══██╗██║   ██║██║     ██╔════╝██╔════╝
  ██████╔╝██║   ██║██║     ███████╗█████╗
  ██╔═══╝ ██║   ██║██║     ╚════██║██╔══╝
  ██║     ╚██████╔╝███████╗███████║███████╗
  ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝
```

> Your terminal's personal trainer — tracks every command you run, coaches your git discipline, and tells you what your data actually means.

Pulse sits quietly in your shell and logs every command you run — which tools you reach for, how much time you spend per project, your streak of active days, and your overall success rate. It also watches your git habits in real time and flags problems before they become incidents.

No cloud. No account. No phone number. Everything lives in a single SQLite file at `~/.devpulse/pulse.db`.

---

## What it tracks

- **Commands** — every command you run, deduplicated and ranked
- **Projects** — time spent per git repo, detected automatically from your working directory
- **Streaks** — consecutive days with coding activity
- **Success rate** — ratio of zero-exit commands to total
- **Grind time** — total active terminal time per project and overall
- **Git activity** — every commit, push, branch, and merge with structured metadata

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/abdulqadirmsingi/pulse-cli/main/scripts/install.sh | bash
```

The script handles everything: downloads the right binary for your OS and chip, installs it to `~/.local/bin`, adds it to your PATH, and runs `pulse init` to set up the database and shell hook.

When it finishes, run:

```bash
source ~/.zshrc   # or ~/.bashrc if you use bash
```

This activates the hook in your **current** terminal. Any new terminal you open after installing will work automatically.

### Install from source

```bash
git clone https://github.com/abdulqadirmsingi/pulse-cli
cd pulse-cli
make install
pulse init
source ~/.zshrc
```

Requires Go 1.21+.

---

## Commands

### Activity tracking

| Command | What it does |
|---------|-------------|
| `pulse stats` | command count, grind time, streak, top commands + projects |
| `pulse stats -d 30` | same but for the last 30 days |
| `pulse history` | every command you ran today in chronological order |
| `pulse history --no-noise` | same, hiding ls / cd / clear |
| `pulse today` | hour-by-hour heatmap of today's activity |
| `pulse projects` | every detected project with time, commands, and success rate |
| `pulse vibe` | pattern insights — what your data says about how you work |
| `pulse dash` | live auto-refreshing TUI dashboard (updates every 5s) |

### Git discipline

| Command | What it does |
|---------|-------------|
| `pulse hooks install` | track commits from VS Code, Cursor, GitHub Desktop — not just the terminal |
| `pulse hooks uninstall` | remove the global git hooks |
| `pulse hooks status` | check which hooks are active |
| `pulse git-guard on` | block force-pushes to main before they run (terminal only) |
| `pulse git-guard off` | disable the guard |
| `pulse git-guard status` | check if the guard is active |

### Maintenance

| Command | What it does |
|---------|-------------|
| `pulse doctor` | check if tracking is set up correctly |
| `pulse update` | update to the latest version |
| `pulse reset --force` | clear all command history and start fresh |
| `pulse uninstall` | remove Pulse from your machine |
| `pulse version` | show the installed version |

---

## Git discipline engine

This is where Pulse goes beyond a tracker. After every git command, Pulse evaluates a set of rules against what you just did and prints a short warning if something looks off. No noise — each warning is one line with an actionable fix.

### Rules

| Rule | Trigger | Severity |
|------|---------|----------|
| Force push to main | `git push --force` to `main` or `master` | 🚫 Block (with git-guard) / ⚠️ Warn |
| Direct commit to main | `git commit` while on `main` or `master` | ⚠️ Warn |
| Direct push to main | `git push origin main` without a PR | ⚠️ Warn |
| Vague branch name | `git checkout -b fix` / `wip` / `temp` / `test` | ⚠️ Warn |
| Vague commit message | `git commit -m "update"` / `"fix"` / `"wip"` | ⚠️ Warn |
| Non-conventional commit | message without `feat:` / `fix:` / `chore:` prefix | ⚠️ Warn |
| Friday afternoon push | `git push` on Friday after 4pm | ⚠️ Warn |
| Bare merge | `git merge` with no branch specified | ⚠️ Warn |

### What good looks like

```bash
# good branch names
git checkout -b feat/user-auth
git checkout -b fix/null-pointer-login
git checkout -b chore/update-dependencies

# good commit messages (conventional format)
git commit -m "feat: add OAuth login flow"
git commit -m "fix: prevent nil panic in auth handler"
git commit -m "chore: upgrade Go to 1.22"
git commit -m "refactor(auth): extract token validation to its own function"

# good push workflow
git push origin feat/user-auth   # push your feature branch
# open a PR on GitHub, get review, merge via GitHub
# never push directly to main
```

### Example warnings

```
$ git commit -m "fix"

  ⚠️  commit message "fix" is too short to be useful
     describe the why: "fix: prevent nil panic in auth handler"

$ git checkout -b wip

  ⚠️  branch name "wip" is too vague
     try: feat/your-feature, fix/the-bug, chore/what-you-did

$ git push origin main

  ⚠️  pushing directly to main — consider opening a PR instead
     git checkout -b feat/your-change, push that, then open a PR
```

With `git-guard` enabled, force-pushes to main are blocked before git even runs:

```
$ git push --force origin main

  🚫 force push to main — this rewrites shared history
     use --force-with-lease if you really must, or open a PR instead

(git never executed)
```

### IDE and GUI support

By default, Pulse only sees commands you type in the terminal. If you commit through VS Code's git panel, Cursor's AI commit, or GitHub Desktop, those are invisible.

Run this once to fix that:

```bash
pulse hooks install
```

This sets a global git hooks path (`~/.config/git/hooks`) that fires for **every** git operation on your machine, regardless of where it originates. The `post-commit` hook logs the commit. The `pre-push` hook detects force-pushes by comparing commit SHAs — no flags required, so it catches force-pushes even from GUI clients.

| How you commit | Tracked | Force-push blocked |
|----------------|---------|-------------------|
| Terminal | ✅ | ✅ (with git-guard on) |
| VS Code / Cursor panel | ✅ after `hooks install` | ✅ after `hooks install` |
| GitHub Desktop | ✅ after `hooks install` | ✅ after `hooks install` |
| AI-generated commit (Cursor) | ✅ after `hooks install` | ✅ after `hooks install` |

---

## How projects are detected

Pulse detects which project you're working on automatically — no configuration required.

When you run any command, Pulse receives the current working directory. It walks **up** the directory tree looking for a `.git` folder:

```
You run a command in:   /Users/you/code/myapp/src/components/auth

Pulse checks:
  /Users/you/code/myapp/src/components/auth/.git  ← not found
  /Users/you/code/myapp/src/components/.git       ← not found
  /Users/you/code/myapp/.git                      ← found!
  project = "myapp"
```

The project name is the folder that contains `.git` — the repo root. If no `.git` is found anywhere in the tree, Pulse falls back to the name of the current directory.

This means every command you run inside a repo — no matter how deep in the folder structure — is automatically attributed to the right project.

---

## How to trigger warnings (testing)

Run these to see Pulse in action:

```bash
# vague commit message
git commit -m "fix"
git commit -m "update"
git commit -m "wip"

# non-conventional format
git commit -m "added some stuff to the login page"

# vague branch name
git checkout -b test
git checkout -b wip
git checkout -b temp

# direct push to main (if you're on main)
git push origin main

# force push to main — blocked with git-guard, warned without
git push --force origin main

# Friday afternoon push (only works on Fridays after 4pm)
git push origin feat/something
```

To see blocking in action:
```bash
pulse git-guard on
source ~/.zshrc
git push --force origin main   # this will be stopped before git runs
```

---

## Example output

```
📊  Your dev pulse  ·  last 7 days

╭──────────────────────────────────────╮
│  🔥  streak            9 day streak 🔥 │
│  ⚡  commands          1,247  ·  +43 noise │
│  ⏰  grind time        14h 32m         │
│  ✅  success rate      94.1%           │
╰──────────────────────────────────────╯

  💻  top 5 commands
  run `pulse history` to see every command in full

  git             ██████████████  342 runs
  npm             ██████████░░░░  214 runs
  vim             ████████░░░░░░  156 runs
  go              ██████░░░░░░░░  123 runs
  docker          ████░░░░░░░░░░   89 runs

  📁  top projects

  myapp           ██████████████  6h 12m
  api-service     ████████░░░░░░  4h 45m
  pulse-cli       █████░░░░░░░░░  2h 58m
```

---

## How it works

`pulse init` appends a small hook to your `.zshrc` or `.bashrc`:

```zsh
_pulse_preexec() {
    _PULSE_CMD_START=$(date +%s)
    _PULSE_CMD="$1"
}
_pulse_precmd() {
    local _exit=$?
    [ -z "$_PULSE_CMD" ] && return
    local _ms=$(( ($(date +%s) - ${_PULSE_CMD_START:-0}) * 1000 ))
    case "$_PULSE_CMD" in
        git\ *)
            /path/to/pulse log ... 2>&1          # sync — so warnings reach your terminal
            ;;
        *)
            /path/to/pulse log ... >/dev/null &| # async — zero latency
            ;;
    esac
}
```

`preexec` fires before each command and captures the command string and start time. `precmd` fires after it exits. Non-git commands are logged in the background — they never block your prompt. Git commands run synchronously so any warnings can appear before your next prompt.

---

## Troubleshooting

**Commands aren't being tracked**

Run `pulse doctor` — it checks your setup end to end. The most common cause: the terminal was open before `pulse init` ran. Either open a new terminal or run `source ~/.zshrc`.

**IDE commits aren't showing up**

Run `pulse hooks install` — the shell hook only fires in the terminal.

**Stats look wrong**

`pulse stats` is a snapshot at the moment you run it. For a live view use `pulse dash`. To wipe old data: `pulse reset --force`.

---

## Data location

| Path | What's there |
|------|-------------|
| `~/.devpulse/pulse.db` | SQLite database — all your command history |
| `~/.config/git/hooks/` | Global git hooks (only if you ran `hooks install`) |

---

## Project structure

```
cmd/            one file per subcommand
internal/
  config/       paths and version
  db/           all SQLite queries
  git/          git command parsing and context extraction
  rules/        git discipline rule engine
  ui/           shared lipgloss styles and formatters
  tui/          Bubble Tea live dashboard
  insights/     rule-based pattern analysis
scripts/        curl-pipe installer
```

No file exceeds 200 lines.

---

## License

MIT
