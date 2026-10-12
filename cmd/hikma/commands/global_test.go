package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// globalHome isolates HOME in a temp dir (via project) and returns it, with the
// cwd moved to a plain folder that is not a git repository.
func globalHome(t *testing.T) string {
	t.Helper()
	project(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(home, os.TempDir()) {
		t.Fatalf("HOME %q does not look like a temp dir; refusing to run", home)
	}
	t.Chdir(t.TempDir())
	return home
}

func TestGlobalInstallWritesUnderHomeAndLocksThere(t *testing.T) {
	reg := registryRepo(t)
	home := globalHome(t)
	cwd, _ := os.Getwd()

	out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "--global", "--yes")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !exists(filepath.Join(home, ".claude", "skills", "alpha", "SKILL.md")) {
		t.Fatalf("skill not installed under HOME:\n%s", out)
	}
	if !strings.Contains(out, filepath.Join(home, ".claude", "skills", "alpha")) {
		t.Errorf("output should name the absolute target:\n%s", out)
	}
	lf, err := lock.Load(home)
	if err != nil || len(lf.Skills) != 1 {
		t.Fatalf("global lockfile: %v, %d entries", err, len(lf.Skills))
	}
	if exists(filepath.Join(cwd, ".claude")) || exists(filepath.Join(cwd, ".hikma")) {
		t.Fatal("a global install must not touch the current directory")
	}
}

func TestGlobalInstallNeedsConfirmationWithoutTerminal(t *testing.T) {
	reg := registryRepo(t)
	home := globalHome(t)

	out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "--global")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected a confirmation error, got %v\n%s", err, out)
	}
	if !strings.Contains(out, home) {
		t.Errorf("the prompt should show the absolute target before failing:\n%s", out)
	}
	if exists(filepath.Join(home, ".claude")) || exists(filepath.Join(home, ".hikma")) {
		t.Fatal("nothing may be written without confirmation")
	}
}

func TestGlobalRefusesSymlinkedTargets(t *testing.T) {
	reg := registryRepo(t)
	home := globalHome(t)
	dev := t.TempDir() // someone's dev repo that ~/.claude points at
	if err := os.Symlink(dev, filepath.Join(home, ".claude")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "--global", "--yes")
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected a symlink refusal, got %v\n%s", err, out)
	}
	if entries, _ := os.ReadDir(dev); len(entries) != 0 {
		t.Fatalf("wrote through the symlink: %v", entries)
	}
}

func TestGlobalVerifyDiffRemoveAndSync(t *testing.T) {
	reg := registryRepo(t)
	home := globalHome(t)
	skill := filepath.Join(home, ".claude", "skills", "alpha")

	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "-g", "-y"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if out, err := run(t, "skill", "verify", "-g"); err != nil {
		t.Fatalf("verify: %v\n%s", err, out)
	}
	write(t, filepath.Join(skill, "notes.md"), "mine\n")
	if out, err := run(t, "skill", "verify", "-g"); err == nil {
		t.Fatalf("verify should fail on a local edit:\n%s", out)
	}
	out, err := run(t, "skill", "diff", "alpha", "--agent", "claude", "-g")
	if err != nil || !strings.Contains(out, "notes.md") {
		t.Fatalf("diff: %v\n%s", err, out)
	}

	if out, err := run(t, "skill", "remove", "alpha", "--agent", "claude", "-g", "-y", "--force"); err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	if exists(skill) {
		t.Fatal("skill folder should be gone")
	}
}

func TestGlobalSyncRestoresAfterWipe(t *testing.T) {
	reg := registryRepo(t)
	home := globalHome(t)
	skill := filepath.Join(home, ".claude", "skills", "alpha")
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "-g", "-y"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if err := os.RemoveAll(filepath.Join(home, ".claude")); err != nil {
		t.Fatal(err)
	}
	if out, err := run(t, "sync", "-g"); err == nil {
		t.Fatalf("sync without --yes must ask first:\n%s", out)
	}
	if out, err := run(t, "sync", "-g", "-y"); err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	if !exists(filepath.Join(skill, "SKILL.md")) {
		t.Fatal("sync --global should restore the skill under HOME")
	}
}

func TestGlobalLockfileKeysCannotEscape(t *testing.T) {
	home := globalHome(t)
	lf := lock.File{Skills: map[string]lock.Entry{
		"../outside/evil": {Name: "evil", Source: "owner/repo", Files: map[string]string{}},
	}}
	if err := lock.Save(home, lf); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "skill", "update", "--all", "-g", "-y")
	if err != nil {
		t.Fatalf("update --all reports per-skill failures without failing the run: %v", err)
	}
	if !strings.Contains(out, "skipped") || exists(filepath.Join(filepath.Dir(home), "outside")) {
		t.Fatalf("a traversal key must be refused:\n%s", out)
	}
}

func TestDuplicateAcrossScopesWarnsAndListsBoth(t *testing.T) {
	reg := registryRepo(t)
	home := globalHome(t)
	proj := t.TempDir()
	t.Chdir(proj)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "-g", "-y"); err != nil {
		t.Fatalf("global install: %v\n%s", err, out)
	}

	out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude")
	if err != nil || !strings.Contains(out, "also installed at global scope") {
		t.Fatalf("project install should warn, not block: %v\n%s", err, out)
	}
	out, err = run(t, "skill", "list", "--installed")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SCOPE", "project", "global", "also installed at"} {
		if !strings.Contains(out, want) {
			t.Errorf("list missing %q:\n%s", want, out)
		}
	}
	out, _ = run(t, "skill", "list", "--installed", "-g")
	if strings.Contains(out, "project ") || !strings.Contains(out, filepath.ToSlash(".claude/skills/alpha")) {
		t.Errorf("--global should list only global skills:\n%s", out)
	}
	out, err = run(t, "sync", "--dry-run")
	if err != nil || !strings.Contains(out, "inherit permissions") {
		t.Errorf("sync should flag the shared name: %v\n%s", err, out)
	}
	_ = home
}
