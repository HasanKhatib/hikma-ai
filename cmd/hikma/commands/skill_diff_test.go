package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffShowsLocalEditsAndUpstreamChanges(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}

	out, err := run(t, "skill", "diff", "alpha", "--agent", "claude")
	if err != nil {
		t.Fatalf("diff: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Local edits:\n  none") || !strings.Contains(out, "none, up to date") {
		t.Fatalf("clean skill should show nothing:\n%s", out)
	}

	// Edit locally, and change two files upstream (one the same, plus a script and a new file).
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "scripts", "run.sh"), "echo local\n")
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "notes.md"), "mine\n")
	write(t, filepath.Join(reg, "skills", "alpha", "scripts", "run.sh"), "echo v2\n")
	write(t, filepath.Join(reg, "skills", "alpha", "extra.md"), "new\n")
	git(t, reg, "add", "-A")
	git(t, reg, "commit", "--quiet", "-m", "v2")

	out, err = run(t, "skill", "diff", "alpha", "--agent", "claude")
	if err != nil {
		t.Fatalf("diff: %v\n%s", err, out)
	}
	for _, want := range []string{
		"modified scripts/run.sh",
		"added    notes.md",
		"added    extra.md",
		"modified scripts/run.sh  (also edited locally)",
		"warning: scripts/ changed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// Read-only: nothing changed on disk or in the lockfile.
	if b, _ := os.ReadFile(filepath.Join(proj, ".claude", "skills", "alpha", "scripts", "run.sh")); string(b) != "echo local\n" {
		t.Fatalf("diff modified a file: %q", b)
	}
	if exists(filepath.Join(proj, ".claude", "skills", "alpha", "extra.md")) {
		t.Fatal("diff must not install upstream files")
	}

	out, err = run(t, "skill", "diff", "alpha", "--agent", "claude", "--patch")
	if err != nil {
		t.Fatalf("diff --patch: %v\n%s", err, out)
	}
	if !strings.Contains(out, "-echo local") || !strings.Contains(out, "+echo v2") {
		t.Fatalf("patch missing contents:\n%s", out)
	}
}

func TestDiffErrorsForUntrackedSkills(t *testing.T) {
	proj := project(t)
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	if out, err := run(t, "skill", "diff", "alpha", "--agent", "claude"); err == nil || !strings.Contains(err.Error(), "not installed by hikma") {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if _, err := run(t, "skill", "diff"); err == nil {
		t.Fatal("missing name accepted")
	}
}
