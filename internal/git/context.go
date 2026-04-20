// Package git provides git-aware parsing and context extraction for Pulse.
package git

import (
	"os"
	"path/filepath"
	"strings"
)

// RepoRoot walks up from dir until it finds a .git directory.
// Returns "" if dir is not inside a git repo.
//
// 🧠 Go Lesson #53: Read files directly instead of shelling out.
// os.Stat on .git takes ~0.1ms. exec.Command("git", "rev-parse") takes ~30ms.
// At hook time that 30ms is visible — the file read is not.
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

// BranchFromDir reads the current branch by parsing .git/HEAD directly.
// Returns "" if dir is not inside a git repo or HEAD is detached.
func BranchFromDir(dir string) string {
	root := RepoRoot(dir)
	if root == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	// HEAD contains either:
	//   ref: refs/heads/main\n   (normal branch)
	//   <sha>\n                  (detached HEAD)
	line := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if !strings.HasPrefix(line, prefix) {
		return "" // detached HEAD
	}
	return strings.TrimPrefix(line, prefix)
}
