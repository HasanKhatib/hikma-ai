package commands

import (
	"path/filepath"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// scope is the root that skill folders and the lockfile live under. Project
// scope is the current directory; lockfile keys and layout folders stay
// slash-separated and relative to the root, so they are the same at any scope.
type scope struct{ root string }

func projectScope() scope { return scope{root: "."} }

// path returns the on-disk location of a root-relative, slash-separated path.
func (s scope) path(rel string) string { return filepath.Join(s.root, filepath.FromSlash(rel)) }

// skillDir returns the on-disk folder where an agent keeps the named skill.
func (s scope) skillDir(t config.Selection, name string) string {
	return filepath.Join(s.root, t.SkillDir(name))
}

// key returns the lockfile key for the named skill under agent t.
func (s scope) key(t config.Selection, name string) string {
	return lockKey(t.SkillDir(name))
}

func (s scope) loadLock() (lock.File, error) { return lock.Load(s.root) }
func (s scope) saveLock(f lock.File) error   { return lock.Save(s.root, f) }
func (s scope) lockPath() string             { return lock.Path(s.root) }
