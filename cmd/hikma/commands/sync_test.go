package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// fileURL turns a local path into a file:// git URL (a git source, not a local-path source).
func fileURL(p string) string {
	p = filepath.ToSlash(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + p
}

// installFromURL installs alpha for claude through a git URL and returns the project dir.
func installFromURL(t *testing.T, reg string, agents string) string {
	t.Helper()
	proj := project(t)
	if out, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", agents); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	return proj
}

func TestSyncRestoresOnAFreshClone(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude,codex")

	// A fresh clone has the lockfile but not the skills folders.
	for _, d := range []string{".claude", ".agents"} {
		if err := os.RemoveAll(filepath.Join(proj, d)); err != nil {
			t.Fatal(err)
		}
	}
	out, err := run(t, "sync", "--yes")
	if err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	for _, d := range []string{".claude/skills/alpha", ".agents/skills/alpha"} {
		if _, err := os.Stat(filepath.Join(proj, filepath.FromSlash(d), "SKILL.md")); err != nil {
			t.Errorf("not restored: %s: %v", d, err)
		}
	}
	if !strings.Contains(out, "Done: 2 restored") {
		t.Fatalf("output:\n%s", out)
	}
	// No registry was configured: sync uses the recorded sources.
	if strings.Contains(out, "no registry configured") {
		t.Fatalf("sync should not need a configured registry:\n%s", out)
	}
}

func TestSyncRestoresTheRecordedCommitNotTheNewest(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")

	// Upstream moves on after the install.
	write(t, filepath.Join(reg, "skills", "alpha", "scripts", "run.sh"), "echo v2\n")
	git(t, reg, "commit", "--quiet", "-am", "v2")

	if err := os.RemoveAll(filepath.Join(proj, ".claude")); err != nil {
		t.Fatal(err)
	}
	if out, err := run(t, "sync", "--yes"); err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	data, _ := os.ReadFile(filepath.Join(proj, ".claude", "skills", "alpha", "scripts", "run.sh"))
	if strings.ReplaceAll(string(data), "\r\n", "\n") != "echo v1\n" { // git may check out CRLF on Windows
		t.Fatalf("sync must restore the recorded commit, got %q", data)
	}
}

func TestSyncWhenNothingToDoDoesNotPrompt(t *testing.T) {
	reg := registryRepo(t)
	installFromURL(t, reg, "claude")
	out, err := run(t, "sync") // no --yes, no terminal: fine because there is no work
	if err != nil || !strings.Contains(out, "up to date") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestSyncNeedsYesWithoutATerminal(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")
	_ = os.RemoveAll(filepath.Join(proj, ".claude"))
	out, err := run(t, "sync")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(proj, ".claude")); statErr == nil {
		t.Fatal("nothing should be fetched before confirmation")
	}
}

func TestSyncProtectsLocalEditsUnlessForced(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")
	skill := filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")
	write(t, skill, "my edit\n")

	out, err := run(t, "sync", "--yes")
	if err == nil || !strings.Contains(out, "local changes") {
		t.Fatalf("expected a skip with an error: %v\n%s", err, out)
	}
	if data, _ := os.ReadFile(skill); string(data) != "my edit\n" {
		t.Fatal("local edit was overwritten without --force")
	}
	if out, err := run(t, "sync", "--yes", "--force"); err != nil {
		t.Fatalf("--force: %v\n%s", err, out)
	}
	if data, _ := os.ReadFile(skill); string(data) == "my edit\n" {
		t.Fatal("--force did not restore the file")
	}
}

func TestSyncRefusesContentThatDoesNotMatchTheLockfile(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")

	lf, _ := lock.Load(".")
	e := lf.Skills[".claude/skills/alpha"]
	e.Files["SKILL.md"] = strings.Repeat("0", 64) // pretend the recorded hash is different
	lf.Skills[".claude/skills/alpha"] = e
	if err := lock.Save(".", lf); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Join(proj, ".claude"))

	out, err := run(t, "sync", "--yes")
	if err == nil || !strings.Contains(out, "do not match") {
		t.Fatalf("expected a hash mismatch: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(proj, ".claude", "skills", "alpha")); statErr == nil {
		t.Fatal("mismatched content was installed")
	}
}

func TestSyncRejectsLockfileEntriesOutsideTheSkillFolders(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	bad := map[string]any{
		"version": 1,
		"skills": map[string]any{
			"../outside/alpha":          map[string]any{"name": "alpha", "source": reg, "files": map[string]string{}},
			".claude/skills/../../evil": map[string]any{"name": "evil", "source": reg, "files": map[string]string{}},
			"/abs/path/alpha":           map[string]any{"name": "alpha", "source": reg, "files": map[string]string{}},
			".claude/skills/other":      map[string]any{"name": "alpha", "source": reg, "files": map[string]string{}},
			"somewhere/else/alpha":      map[string]any{"name": "alpha", "source": reg, "files": map[string]string{}},
			".claude/skills/alpha":      map[string]any{"name": "alpha", "source": "not a source", "files": map[string]string{}},
		},
	}
	data, _ := json.Marshal(bad)
	write(t, filepath.Join(proj, ".hikma", "lock.json"), string(data))

	out, err := run(t, "sync", "--yes")
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("expected refusal: %v\n%s", err, out)
	}
	for _, p := range []string{"../outside", "evil", "abs", "somewhere", ".claude"} {
		if _, statErr := os.Stat(filepath.Join(proj, p)); statErr == nil {
			t.Errorf("sync created %s", p)
		}
	}
}

func TestSyncDryRunChangesNothing(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")
	_ = os.RemoveAll(filepath.Join(proj, ".claude"))
	out, err := run(t, "sync", "--dry-run")
	if err != nil || !strings.Contains(out, "restore") || !strings.Contains(out, "Dry run") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(proj, ".claude")); statErr == nil {
		t.Fatal("dry run installed something")
	}
}

func TestSyncWithNoLockfile(t *testing.T) {
	project(t)
	out, err := run(t, "sync")
	if err != nil || !strings.Contains(out, "No skills recorded") {
		t.Fatalf("%v\n%s", err, out)
	}
}
