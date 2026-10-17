package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// scope is the root that skill folders and the lockfile live under. Project
// scope is the current directory; lockfile keys and layout folders stay
// slash-separated and relative to the root, so they are the same at any scope.
type scope struct {
	root   string
	global bool
}

func projectScope() scope { return scope{root: "."} }

// globalScope is the user's home directory. It is only ever built from an
// explicit --global flag, never from config, the lockfile, or the environment
// of a committed file. HOME (USERPROFILE on Windows) is what os.UserHomeDir
// reads, so tests point it at a temp directory.
func globalScope() (scope, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return scope{}, fmt.Errorf("--global needs a home directory: %v", err)
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return scope{}, err
	}
	return scope{root: abs, global: true}, nil
}

// scopeFor returns the global scope when asked, otherwise the project scope.
func scopeFor(global bool) (scope, error) {
	if global {
		return globalScope()
	}
	return projectScope(), nil
}

// lockName is the lockfile as the user should read it in messages.
func (s scope) lockName() string {
	if s.global {
		return "~/.hikma/lock.json"
	}
	return ".hikma/lock.json"
}

// guard refuses to write through a symlink at global scope. People link their
// global skill folders to their own dev repos; hikma must not replace or delete
// what the link points at. rel is a root-relative, slash-separated path.
func (s scope) guard(rel string) error {
	if !s.global {
		return nil
	}
	cur := s.root
	for _, part := range strings.Split(filepath.ToSlash(filepath.Clean(rel)), "/") {
		if part == "." || part == "" {
			continue
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			return nil // nothing there yet, so nothing to write through
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through the symlink %s; point hikma at a real folder or remove the link", cur)
		}
	}
	return nil
}

// guardWrite checks the skill folders and the lockfile folder for symlinks.
func (s scope) guardWrite(keys ...string) error {
	if err := s.guard(".hikma"); err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.guard(k); err != nil {
			return err
		}
	}
	return nil
}

// confirmWrite shows the absolute paths a global command will change and asks
// first. Project scope needs no confirmation. Scripts, the CI case, and
// --yes all follow the same rule: no prompt without a terminal unless --yes.
func (s scope) confirmWrite(cmd *cobra.Command, yes bool, keys []string) error {
	if !s.global {
		return nil
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\nThis changes your home directory (%s):\n", s.root)
	for _, k := range keys {
		fmt.Fprintf(out, "  %s\n", s.path(k))
	}
	fmt.Fprintf(out, "  %s\n", s.lockPath())
	if yes {
		return nil
	}
	return askGlobal(out)
}

func askGlobal(_ io.Writer) error {
	if !isInteractive() {
		return fmt.Errorf("confirmation required: re-run with --yes to change your home directory")
	}
	ok, err := prompter.Confirm("Continue?", false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("cancelled")
	}
	return nil
}

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

// otherScope is the scope a duplicate would live in: global for a project, and
// the project (current directory) for global. It is false when the two are the
// same folder, for example when hikma runs from the home directory.
func (s scope) otherScope() (scope, bool) {
	var o scope
	if s.global {
		o = projectScope()
	} else {
		g, err := globalScope()
		if err != nil {
			return scope{}, false
		}
		o = g
	}
	if sameDir(s.root, o.root) {
		return scope{}, false
	}
	return o, true
}

func (s scope) label() string {
	if s.global {
		return "global"
	}
	return "project"
}

// installedNames returns the names installed in a scope: tracked or on disk.
// Reading is harmless, so an unreadable scope reads as empty.
func installedNames(s scope) map[string]string {
	names := map[string]string{}
	rows, err := listInstalledSkills(s)
	if err != nil {
		return names
	}
	for _, r := range rows {
		names[r.Name] = r.Path
	}
	return names
}

// shadowWarnings returns one warning per name that is installed at both scopes.
// It never blocks: which copy an agent loads differs by agent and is mostly
// undocumented (the Claude Code slash command runs the global copy).
func (s scope) shadowWarnings(names []string) []string {
	o, ok := s.otherScope()
	if !ok {
		return nil
	}
	there := installedNames(o)
	var out []string
	seen := map[string]bool{}
	for _, n := range names {
		p, dup := there[n]
		if !dup || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, fmt.Sprintf("warning: %q is also installed at %s scope (%s). the Claude Code slash command runs the global copy; other agents differ or do not document which copy wins", n, o.label(), o.path(p)))
	}
	return out
}
