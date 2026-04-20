package git

import (
	"os"
	"path/filepath"
	"strings"
)

func RepoRoot(dir string) string {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func BranchFromDir(dir string) string {
	root := RepoRoot(dir)
	if root == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if !strings.HasPrefix(line, prefix) {
		return "" // detached HEAD
	}
	return strings.TrimPrefix(line, prefix)
}
