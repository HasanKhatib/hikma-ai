package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

func TestScopePathsStayUnderRoot(t *testing.T) {
	root := t.TempDir()
	sc := scope{root: root}
	if got, want := sc.path(".claude/skills/a"), filepath.Join(root, ".claude", "skills", "a"); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if got := projectScope().path(".claude/skills/a"); got != filepath.Join(".claude", "skills", "a") {
		t.Fatalf("project path = %q", got)
	}
	if got, want := sc.lockPath(), filepath.Join(root, ".hikma", "lock.json"); got != want {
		t.Fatalf("lockPath = %q, want %q", got, want)
	}
}

func TestScopeLockRoundTrip(t *testing.T) {
	sc := scope{root: t.TempDir()}
	lf, err := sc.loadLock()
	if err != nil {
		t.Fatal(err)
	}
	lf.Skills[".claude/skills/a"] = lock.Entry{Name: "a", Source: "o/r", Files: map[string]string{}}
	if err := sc.saveLock(lf); err != nil {
		t.Fatal(err)
	}
	back, err := sc.loadLock()
	if err != nil || len(back.Skills) != 1 {
		t.Fatalf("round trip: %v, %d skills", err, len(back.Skills))
	}
}

func TestPruneEmptyParentsStopsAtRoot(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".claude", "skills", "a")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	pruneEmptyParents(root, dir)
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatalf(".claude should be pruned: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root must survive: %v", err)
	}
}
