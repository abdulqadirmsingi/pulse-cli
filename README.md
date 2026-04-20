```
  ██████╗ ██╗   ██╗██╗     ███████╗███████╗
  ██╔══██╗██║   ██║██║     ██╔════╝██╔════╝
  ██████╔╝██║   ██║██║     ███████╗█████╗
  ██╔═══╝ ██║   ██║██║     ╚════██║██╔══╝
  ██║     ╚██████╔╝███████╗███████║███████╗
  ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝
```

> Your terminal's personal trainer — tracks every command you run, how long you grind, and what you actually ship.



Pulse sits quietly in your shell and logs every command you run — which tools you reach for, how much time you spend per project, your streak of active days, and your overall success rate. Then it surfaces all of that in a clean dashboard that actually tells you something useful.

No cloud. No account. No phone number. Everything lives in a single SQLite file at `~/.devpulse/pulse.db`.

---

## What it tracks

- **Commands** — every command you run, deduplicated and ranked
- **Projects** — time spent per git repo, detected automatically from your working directory
- **Streaks** — consecutive days with coding activity
- **Success rate** — ratio of zero-exit commands to total
- **Grind time** — total active terminal time per project and overall

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/abdulqadirmsingi/pulse-cli/main/scripts/install.sh | bash
```

The script handles everything: downloads the right binary for your OS and chip, installs it to `~/.local/bin`, adds it to your PATH, and runs `pulse init` to set up the database and shell hook.

When it finishes, it'll show you one command to run:

```bash
source ~/.zshrc   # or ~/.bashrc if you use bash
```

This activates the hook in your **current** terminal. Any new terminal you open after installing will work automatically without this step.

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

| Command | What it does |
|---------|-------------|
| `pulse stats` | your command count, grind time, streak, top commands + projects |
| `pulse stats -d 30` | same but for the last 30 days |
| `pulse today` | hour-by-hour heatmap of today's activity |
| `pulse projects` | every detected project with time, commands, and success rate |
| `pulse vibe` | pattern insights — what your data says about how you work |
| `pulse dash` | live auto-refreshing TUI dashboard (updates every 5s) |
| `pulse doctor` | check if tracking is set up correctly |
| `pulse update` | update to the latest version |
| `pulse reset --force` | clear all command history and start fresh |
| `pulse uninstall` | remove pulse from your machine |
| `pulse version` | show the installed version |

---

## Example output

```
📊  ur dev pulse  ·  last 7 days

╭──────────────────────────────────────╮
│  🔥  streak            9 day streak 🔥 │
│  ⚡  commands          1,247           │
│  ⏰  grind time        14h 32m         │
│  ✅  success rate      94.1%           │
╰──────────────────────────────────────╯

  💻  top commands

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
    /path/to/pulse log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" >/dev/null 2>&1 &|
    unset _PULSE_CMD _PULSE_CMD_START
}
autoload -Uz add-zsh-hook
add-zsh-hook preexec _pulse_preexec
add-zsh-hook precmd  _pulse_precmd
```

`preexec` fires before each command and captures the command string and start time. `precmd` fires after it exits and calls `pulse log` in the background — it never blocks your prompt. The full binary path is embedded so it works regardless of what's in your PATH at hook time.

Your data never leaves your machine. The SQLite file is yours — query it directly with any SQLite client.

---

## Troubleshooting

**Commands aren't being tracked**

Run `pulse doctor` — it checks your setup end to end and tells you exactly what's wrong.

The most common cause: the terminal was opened before `pulse init` was run. The hook only loads in terminals started after it was written to `.zshrc`. Either open a new terminal or run `source ~/.zshrc`.

**Stats look wrong / showing old data**

`pulse stats` is a snapshot — it shows data at the moment you run it. For a live auto-refreshing view use `pulse dash`.

To wipe old data and start fresh: `pulse reset --force`

---

## Data location

| Path | What's there |
|------|-------------|
| `~/.devpulse/pulse.db` | SQLite database — all your command history |

---

## Project structure

```
cmd/            one file per subcommand
internal/
  config/       paths and version
  db/           all SQLite queries
  ui/           shared lipgloss styles and formatters
  tui/          Bubble Tea live dashboard
  insights/     rule-based pattern analysis
scripts/        curl-pipe installer
```

No file exceeds 200 lines.

---

## License

MIT
