# Pulse — Project Rules

## Code Structure

- **No file exceeds 200 lines.** If it's getting long, split it into focused sub-files in the same package.
- **All business logic lives in `internal/`.** The `cmd/` layer only parses flags and calls into `internal/`.
- **One package per concern.** `internal/db` owns all SQL. `internal/ui` owns all styling. `internal/insights` owns all rules. `internal/tui` owns the Bubble Tea model. `internal/config` owns paths and version.
- **No circular imports.** `cmd/` imports `internal/*`. `internal/` packages never import `cmd/`.

## Go Practices

- **Errors are values — always handle them.** Never use `_` to discard an error unless you have a documented reason. Background tasks (shell hook) are the one exception — they must fail silently so they never interrupt the user.
- **Return `(value, error)` from any function that can fail.** Don't panic in library code; panic only in `main` or `init` for unrecoverable startup failures.
- **Use `fmt.Errorf("context: %w", err)` to wrap errors** so the call chain is visible without a stack trace library.
- **Prefer named types over raw primitives for domain concepts.** Example: `type Level int` instead of bare `int` for insight severity — the compiler enforces it at call sites.
- **Use `iota` for sequential constants** in named types. Never hardcode magic integers.
- **Package-level `var` blocks for pre-built styles and compiled regexps.** `regexp.MustCompile` at package level panics at startup on a bad pattern — that's correct behaviour for compile-time-known patterns.
- **Use `regexp.Compile` (not `MustCompile`) for any pattern that comes from user input.**
- **Interfaces only when you need them.** Don't define an interface for a type that has one implementation. Add the interface when you add the second implementation or when you need to mock in tests.
- **Struct embedding over inheritance.** Go has no classes. Compose behaviour by embedding structs or implementing interfaces.
- **`defer` for cleanup** — `defer db.Close()`, `defer rows.Close()`, `defer resp.Body.Close()`. Always defer immediately after the resource is opened so it can't be forgotten on early-return paths.
- **Short variable names in short scopes.** `i`, `e`, `r` are fine inside a 3-line loop. Use full names at package or function scope.
- **`strings.Fields(s)[0]`** to get the base command from a full command string — handles multiple spaces and tabs safely.
- **`filepath.Join` for all path construction** — never concatenate with `/`. It handles OS differences and double-slash issues.
- **`os.Executable()` for self-referential paths** — use it when a binary needs to know its own install location (hooks, uninstall).
- **`runtime.GOOS` / `runtime.GOARCH`** for platform detection at runtime — never shell out to `uname`.
- **Atomic file replacement:** write to `file.new`, then `os.Rename(file.new, file)`. Rename on the same filesystem is a single syscall — no partial-write risk.
- **SQL: always `defer rows.Close()`** after a successful `Query`. A forgotten close leaks the connection back to the pool.
- **SQL: use `?` placeholders, never string-format user input into queries** — prevents SQL injection.
- **Migrations are idempotent.** Use `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` — safe to run on every startup.

## CLI and UX

- **Friendly tone in all user-facing output.** No corporate speak. Keep it punchy, real, occasionally self-aware.
- **Hidden commands stay hidden.** `pulse log` is called by the shell hook only — `Hidden: true` keeps it out of `pulse help`.
- **Shell hooks must be silent.** All hook output is redirected to `/dev/null`. Failures must never interrupt the user's terminal.
- **zsh hook uses `&|`** (background + immediate disown) — prevents `[1] job` notifications in the terminal.
- **bash hook uses `& disown $!`** — same effect, bash-compatible syntax.
- **Always embed the full binary path in the hook** (via `os.Executable()`), never a bare `pulse` — prevents `exit 127` when pulse isn't in the background process's PATH.
- **`pulse doctor` is the first debugging step.** Before digging into code, run `pulse doctor` — it checks config, data dir, DB, hook, and command count.

## Styling

- **All colours defined once in `internal/ui/styles.go`.** Never hardcode a hex colour anywhere else — reference `ui.ColorCyan` etc.
- **Don't wrap bar chart sections in `ui.Box`.** Block characters (█ ░) render as double-width in some terminals, causing lipgloss to miscalculate border position. Use plain indented text for bar sections; reserve `Box` for fixed-width metric rows.
- **Always truncate names before passing to lipgloss `Width()`.** `Width()` pads to a minimum — it does not clip. Long names overflow and break alignment. Use `ui.Truncate(name, max)` first.

## Documentation Maintenance

- **Every new command or feature must update two things before the PR merges:**
  1. `README.md` — add it to the relevant section so users know it exists
  2. The `pulse --help` output — make sure the `Short` description in the Cobra command is clear and helpful
- The help text is the first thing a new user sees. A command with a missing or confusing `Short` is considered incomplete.

## Releasing a New Version

When you make changes and want to ship them to users:

**1. Bump the version** in `internal/config/config.go`:
```go
const AppVersion = "0.3.0"  // change this
```

**2. Commit, tag, push:**
```bash
git add .
git commit -m "release v0.3.0"
git tag v0.3.0
git push origin main v0.3.0
```

**3. Release with GoReleaser:**
```bash
goreleaser release --clean
```

This builds binaries for all platforms and publishes them to GitHub Releases automatically. Users then run:
```bash
pulse update
```
and get the new version installed in-place.

**Rules for version numbers (`MAJOR.MINOR.PATCH`):**
- `PATCH` (0.2.**1**) — bug fix only, no new commands or flags
- `MINOR` (0.**3**.0) — new command, new flag, or changed output format
- `MAJOR` (**1**.0.0) — breaking change to hook format, DB schema, or config structure

## Go Lessons Index

Inline comments in the codebase are labelled `🧠 Go Lesson #N`. Current range: #1–#50. The next lesson added should be #51. Topics covered so far include: package main, Cobra flags, SQLite with `database/sql`, lipgloss styling, Bubble Tea Elm architecture, `iota` and named types, `os.Executable`, `regexp.MustCompile` vs `Compile`, `fmt.Sprintf` with `%%` escaping, `runtime.GOOS`/`runtime.GOARCH`, and atomic file replacement.
