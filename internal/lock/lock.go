// Package lock records where installed skills came from, so updates can use
// the original source and detect local edits.
package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const version = 1

// Entry describes one installed skill.
type Entry struct {
	Name   string            `json:"name"`
	Source string            `json:"source"`
	Ref    string            `json:"ref,omitempty"`
	Commit string            `json:"commit,omitempty"`
	Files  map[string]string `json:"files"`
}

// File is the contents of .hikma/lock.json. Skills are keyed by their
// slash-separated install directory, such as ".claude/skills/my-skill".
type File struct {
	Version int              `json:"version"`
	Skills  map[string]Entry `json:"skills"`
}

// Path returns the lockfile location under root.
func Path(root string) string { return filepath.Join(root, ".hikma", "lock.json") }

// Load reads the lockfile under root. A missing file yields an empty one.
func Load(root string) (File, error) {
	data, err := os.ReadFile(Path(root))
	if os.IsNotExist(err) {
		return File{Version: version, Skills: map[string]Entry{}}, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read lockfile: %w", err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", Path(root), err)
	}
	if f.Skills == nil {
		f.Skills = map[string]Entry{}
	}
	f.Version = version
	return f, nil
}

// Save writes the lockfile under root.
func Save(root string, f File) error {
	f.Version = version
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return fmt.Errorf("create lockfile directory: %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	if err := os.WriteFile(Path(root), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	return nil
}

// Keys returns the install directories in sorted order.
func (f File) Keys() []string {
	keys := make([]string, 0, len(f.Skills))
	for k := range f.Skills {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
