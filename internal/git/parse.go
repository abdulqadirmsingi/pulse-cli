package git

import "strings"

type Event struct {
	Subcommand string   // "commit", "push", "checkout", etc.
	Args       []string // args after the subcommand
	Branch     string   // local branch at time of command, from .git/HEAD
	IsForce    bool     // --force or -f present in args
	Remote     string   // first non-flag positional for push/pull/fetch
	PushTarget string   // explicit target branch for `git push <remote> <branch>`
	Message    string   // -m value for git commit
	Dir        string   // working directory
}

func Parse(cmd, dir string) *Event {
	fields := tokenize(cmd)
	if len(fields) < 2 || strings.ToLower(fields[0]) != "git" {
		return nil
	}

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
	e.Message = stripQuotes(flagValue(e.Args, "-m", "--message"))
	switch e.Subcommand {
	case "push", "pull", "fetch", "clone":
		positionals := allPositionals(e.Args)
		if len(positionals) > 0 {
			e.Remote = positionals[0]
		}
		if e.Subcommand == "push" && len(positionals) > 1 {
			e.PushTarget = positionals[1]
		}
	}
	return e
}

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
		if set[a] || a == "--force-with-lease" {
			return true
		}
	}
	return false
}

func firstPositional(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func allPositionals(args []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			out = append(out, a)
		}
	}
	return out
}

func tokenize(s string) []string {
	var tokens []string
	var cur strings.Builder
	inDouble := false
	inSingle := false

	for _, r := range s {
		switch {
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case (r == ' ' || r == '\t') && !inDouble && !inSingle:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func flagValue(args []string, names ...string) string {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for i, a := range args {
		for _, n := range names {
			if strings.HasPrefix(a, n+"=") {
				return strings.TrimPrefix(a, n+"=")
			}
		}
		if nameSet[a] && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
