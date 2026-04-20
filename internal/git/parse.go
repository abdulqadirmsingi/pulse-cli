package git

import "strings"

// Event is a parsed git command captured from the shell hook.
type Event struct {
	Subcommand string   // "commit", "push", "checkout", etc.
	Args       []string // args after the subcommand
	Branch     string   // branch at time of command, from .git/HEAD
	IsForce    bool     // --force or -f present in args
	Remote     string   // first non-flag positional after subcommand for push/pull/fetch
	Message    string   // -m value for git commit
	Dir        string   // working directory
}

// Parse returns an Event from a raw command string and working directory.
// Returns nil if the command is not a git command or has no subcommand.
func Parse(cmd, dir string) *Event {
	fields := strings.Fields(cmd)
	// must start with "git" and have at least one subcommand
	if len(fields) < 2 || strings.ToLower(fields[0]) != "git" {
		return nil
	}

	// skip any global git flags before the subcommand (e.g. git -C /path commit)
	subIdx := 1
	for subIdx < len(fields) && strings.HasPrefix(fields[subIdx], "-") {
		subIdx++
	}
	if subIdx >= len(fields) {
		return nil
	}

	e := &Event{
		Subcommand: strings.ToLower(fields[subIdx]),
		Args:       fields[subIdx+1:],
		Dir:        dir,
		Branch:     BranchFromDir(dir),
	}

	e.IsForce = hasFlag(e.Args, "--force", "-f")
	e.Remote = firstPositional(e.Args)
	e.Message = flagValue(e.Args, "-m", "--message")
	return e
}

// IsGit returns true if cmd starts with "git".
func IsGit(cmd string) bool {
	f := strings.Fields(cmd)
	return len(f) > 0 && strings.ToLower(f[0]) == "git"
}

func hasFlag(args []string, flags ...string) bool {
	set := make(map[string]bool, len(flags))
	for _, f := range flags {
		set[f] = true
	}
	for _, a := range args {
		// handle --force-with-lease as a force variant
		if set[a] || a == "--force-with-lease" {
			return true
		}
	}
	return false
}

// firstPositional returns the first arg that is not a flag.
func firstPositional(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// flagValue returns the value of a named flag, e.g. -m "message" or --message=foo.
func flagValue(args []string, names ...string) string {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for i, a := range args {
		// --message=value form
		for _, n := range names {
			if strings.HasPrefix(a, n+"=") {
				return strings.TrimPrefix(a, n+"=")
			}
		}
		// -m value form
		if nameSet[a] && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
