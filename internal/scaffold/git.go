package scaffold

import (
	"os"
	"path/filepath"
)

// FindGitRoot walks up the directory tree from dir looking for a .git entry
// (either a directory for normal repos or a file for git worktrees).
// It returns the directory containing .git and true if found, or ("", false)
// if no git repository exists anywhere in the tree.
func FindGitRoot(dir string) (string, bool) {
	// Resolve to absolute path so parent traversal is unambiguous.
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}

	current := abs
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached the filesystem root with no .git found.
			return "", false
		}
		current = parent
	}
}
