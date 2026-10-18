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

// Version is the lockfile schema this build reads and writes.
const Version = 1

const version = Version

// TooNewError means the lockfile was written by a newer hikma. It is never
// rewritten: saving would drop fields this build does not know about.
type TooNewError struct {
	Path  string
	Found int
}

func (e *TooNewError) Error() string {
	return fmt.Sprintf("%s has lockfile version %d, but this hikma understands up to version %d; upgrade hikma (the file was not changed)", e.Path, e.Found, Version)
}

func checkVersion(path string, found int) error {
	if found > Version {
		return &TooNewError{Path: path, Found: found}
	}
	if found < 0 {
		return fmt.Errorf("%s has an invalid lockfile version %d", path, found)
	}
	return nil
}

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
	if err := checkVersion(Path(root), f.Version); err != nil {
		return File{}, err
	}
	if f.Skills == nil {
		f.Skills = map[string]Entry{}
	}
	// A missing version means 1, the first schema.
	f.Version = version
	return f, nil
}

// Save writes the lockfile under root. It refuses to overwrite a newer one.
func Save(root string, f File) error {
	f.Version = version
	// Never overwrite a file a newer hikma wrote, even if the caller skipped Load.
	if data, err := os.ReadFile(Path(root)); err == nil {
		var existing struct {
			Version int `json:"version"`
		}
		if json.Unmarshal(data, &existing) == nil {
			if err := checkVersion(Path(root), existing.Version); err != nil {
				return err
			}
		}
	}
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
