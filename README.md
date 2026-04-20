# pulse

> your terminal's personal trainer — tracks every command you run, how long you grind, and what you actually ship.

```
  ██████╗ ██╗   ██╗██╗     ███████╗███████╗
  ██╔══██╗██║   ██║██║     ██╔════╝██╔════╝
  ██████╔╝██║   ██║██║     ███████╗█████╗
  ██╔═══╝ ██║   ██║██║     ╚════██║██╔══╝
  ██║     ╚██████╔╝███████╗███████║███████╗
  ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝
```

Pulse sits quietly in your shell and logs every command you run — which tools you reach for, how much time you spend per project, your streak of active days, and your overall success rate. Then it surfaces all of that in a clean, opinionated dashboard that actually tells you something useful.

No cloud, no account, no phone number. Everything lives in a single SQLite file at `~/.devpulse/pulse.db`.

---

## What it tracks

- **Commands** — every command you run, deduplicated and ranked
- **Projects** — time spent per git repository, detected automatically from your working directory
- **Streaks** — consecutive days with coding activity, GitHub-contribution style
- **Success rate** — ratio of zero-exit commands to total commands
- **Grind time** — total active terminal time per project and overall

---

## Install

### Option 1 — one-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/devpulse-cli/devpulse/main/scripts/install.sh | bash
```

This downloads the right pre-built binary for your OS and architecture (macOS arm64/amd64, Linux arm64/amd64, Windows) and puts it in `/usr/local/bin`.

### Option 2 — Go install

```bash
go install github.com/devpulse-cli/devpulse@latest
```

Requires Go 1.21+. The binary ends up in your `$GOPATH/bin`.

### Option 3 — from source

```bash
git clone https://github.com/devpulse-cli/devpulse
cd devpulse
make install
```

---

## Setup

After installing, run this once:

```bash
pulse init
```

This creates `~/.devpulse/`, initialises the database, and appends a small hook to your `.zshrc` or `.bashrc`. Then reload your shell:

```bash
source ~/.zshrc   # or ~/.bashrc
```

From this point forward Pulse records every command in the background. It adds no visible latency — the logging happens in a forked subprocess.

---

## Usage

```bash
# your stats for the last 7 days
pulse stats

# zoom out to the last 30 days
pulse stats --days 30

# check what version you're running
pulse version
```

Example output:

```
📊  ur dev pulse  ·  last 7 days
╭──────────────────────────────────────╮
│  🔥  streak            9 day streak 🔥  │
│  ⚡  commands          1,247          │
│  ⏰  grind time        14h 32m        │
│  ✅  success rate      94.1%          │
╰──────────────────────────────────────╯
╭─────────────────────────────────────────╮
│  💻  top commands              │
│                                         │
│  git           ██████████████  342 runs │
│  npm           ██████████░░░░  214 runs │
│  vim           ████████░░░░░░  156 runs │
│  go            ██████░░░░░░░░  123 runs │
│  docker        ████░░░░░░░░░░   89 runs │
╰─────────────────────────────────────────╯
╭─────────────────────────────────────────────╮
│  📁  top projects                           │
│                                             │
│  myapp             ██████████████  6h 12m   │
│  api-service       ████████░░░░░░  4h 45m   │
│  devpulse          █████░░░░░░░░░  2h 58m   │
╰─────────────────────────────────────────────╯
```

---

## How it works

When you run `pulse init`, it appends a small hook to your shell config:

```bash
_pulse_preexec() {
    _PULSE_CMD_START=$(date +%s)
    _PULSE_CMD="$1"
}
_pulse_precmd() {
    local _exit=$?
    local _ms=$(( ($(date +%s) - ${_PULSE_CMD_START:-0}) * 1000 ))
    pulse log --cmd "$_PULSE_CMD" --exit "$_exit" --ms "$_ms" --dir "$PWD" 2>/dev/null &
}
```

`preexec` fires before each command and captures the command text and start time. `precmd` fires after the command exits and calls `pulse log` in the background (the `&` means it never blocks your prompt). The log command writes one row to SQLite and exits. The whole round trip is under 10ms.

Your data never leaves your machine. The database is a plain SQLite file — you can query it directly with any SQLite client if you want to build your own views.

---

## Data location

| Path | Purpose |
|------|---------|
| `~/.devpulse/pulse.db` | SQLite database — all your command history |

To clear all data: `rm ~/.devpulse/pulse.db` then run `pulse init` again.

---

## Roadmap

- [ ] `pulse dash` — interactive real-time TUI dashboard
- [ ] `pulse today` — hour-by-hour activity heatmap for the current day
- [ ] `pulse vibe` — AI-powered insights and pattern detection (Claude API)
- [ ] `pulse goals` — set weekly command/time targets with progress tracking
- [ ] Homebrew tap for one-line install on macOS

---

## Contributing

The project is structured to be easy to navigate:

```
cmd/           # CLI commands — one file per subcommand
internal/
  config/      # path resolution and app settings
  db/          # all SQLite queries
  ui/          # shared lipgloss styles and formatters
scripts/       # install.sh for the curl-pipe installer
```

No file exceeds 200 lines. If a file is getting long, it's a sign the logic should be split into a new package.

```bash
# run from source without installing
go run . stats

# build a local binary
make build

# seed test data to develop against
make seed
```

---

## License

MIT
