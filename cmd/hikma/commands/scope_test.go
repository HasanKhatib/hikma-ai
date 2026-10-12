package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

// installedElsewhere installs a skill into a project, then moves the cwd to an
// unrelated directory so only the scope root can find the install.
func installedElsewhere(t *testing.T) (scope, string) {
	t.Helper()
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	t.Chdir(t.TempDir())
	return scope{root: proj}, reg
}

func TestResolveSkillDirUsesScopeRoot(t *testing.T) {
	sc, _ := installedElsewhere(t)
	got, err := resolveSkillDirForAgent(sc, "alpha", "claude")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(sc.root, ".claude", "skills", "alpha"); got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
}

func TestDiffOneReadsFolderFromScopeRoot(t *testing.T) {
	sc, _ := installedElsewhere(t)
	lf, err := sc.loadLock()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	u := &updater{sc: sc, out: &out, errOut: &out, lf: lf, checkouts: map[string]*openCheckout{}}
	defer u.close()
	if err := diffOne(u, ".claude/skills/alpha", false); err != nil {
		t.Fatalf("diffOne: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "folder is missing") || !strings.Contains(out.String(), "Local edits:\n  none") {
		t.Fatalf("diff should see the folder under the scope root:\n%s", out.String())
	}
}
