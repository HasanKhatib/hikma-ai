package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

func installAlpha(t *testing.T, agents string) string {
	t.Helper()
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", agents); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	return proj
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestRemoveDeletesTheFolderAndLockEntry(t *testing.T) {
	proj := installAlpha(t, "claude,codex")

	out, err := run(t, "skill", "remove", "alpha", "--agent", "claude")
	if err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	if exists(filepath.Join(proj, ".claude", "skills", "alpha")) {
		t.Fatal(".claude copy still there")
	}
	if !exists(filepath.Join(proj, ".agents", "skills", "alpha", "SKILL.md")) {
		t.Fatal("the other agent's copy must stay")
	}
	lf, _ := lock.Load(".")
	if len(lf.Skills) != 1 || lf.Skills[".agents/skills/alpha"].Name != "alpha" {
		t.Fatalf("lock = %v", lf.Keys())
	}

	// Removing the last one deletes the now-empty lockfile.
	if out, err := run(t, "skill", "remove", "alpha", "--agent", "codex"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if exists(lock.Path(".")) {
		t.Fatal("empty lockfile should be removed")
	}
}

func TestRemoveUsesTheProjectsAgents(t *testing.T) {
	proj := installAlpha(t, "claude,codex")
	if _, err := run(t, "config", "set", "agents", "claude,codex", "--project"); err != nil {
		t.Fatal(err)
	}
	if out, err := run(t, "skill", "remove", "alpha"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if exists(filepath.Join(proj, ".claude", "skills", "alpha")) || exists(filepath.Join(proj, ".agents", "skills", "alpha")) {
		t.Fatal("both copies should be removed")
	}
}

func TestRemoveKeepsLocalEditsUnlessForced(t *testing.T) {
	proj := installAlpha(t, "claude")
	skill := filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")
	write(t, skill, "my notes\n")

	out, err := run(t, "skill", "remove", "alpha", "--agent", "claude")
	if err == nil || !strings.Contains(out, "local changes") {
		t.Fatalf("expected it to keep the edited skill: %v\n%s", err, out)
	}
	if !exists(skill) {
		t.Fatal("edited skill was deleted without --force")
	}
	if out, err := run(t, "skill", "remove", "alpha", "--agent", "claude", "--force"); err != nil {
		t.Fatalf("--force: %v\n%s", err, out)
	}
	if exists(filepath.Join(proj, ".claude", "skills", "alpha")) {
		t.Fatal("--force should delete it")
	}
}

func TestRemoveKeepsUntrackedSkillsUnlessForced(t *testing.T) {
	proj := project(t)
	manual := filepath.Join(proj, ".claude", "skills", "manual")
	write(t, filepath.Join(manual, "SKILL.md"), "---\nname: manual\ndescription: d\n---\n")

	out, err := run(t, "skill", "remove", "manual", "--agent", "claude")
	if err == nil || !strings.Contains(out, "not installed by hikma") {
		t.Fatalf("%v\n%s", err, out)
	}
	if !exists(manual) {
		t.Fatal("untracked skill was deleted without --force")
	}
	if out, err := run(t, "skill", "remove", "manual", "--agent", "claude", "--force"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if exists(manual) {
		t.Fatal("--force should delete it")
	}
}

func TestRemoveErrorsWhenNotInstalledOrUnsafe(t *testing.T) {
	project(t)
	if _, err := run(t, "skill", "remove", "nothing", "--agent", "claude"); err == nil {
		t.Fatal("expected not-installed error")
	}
	for _, bad := range []string{"../x", "a/b", ".."} {
		if _, err := run(t, "skill", "remove", bad, "--agent", "claude", "--force"); err == nil {
			t.Errorf("remove %q should be rejected", bad)
		}
	}
}

func TestListInstalledShowsStatusesAndUntrackedSkills(t *testing.T) {
	proj := installAlpha(t, "claude,codex")
	// modified: edit the codex copy; missing: delete the claude copy; untracked: add one by hand.
	write(t, filepath.Join(proj, ".agents", "skills", "alpha", "SKILL.md"), "edited\n")
	if err := os.RemoveAll(filepath.Join(proj, ".claude", "skills", "alpha")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(proj, ".agents", "skills", "manual", "SKILL.md"), "---\nname: manual\ndescription: d\n---\n")

	rows, err := listInstalledSkills()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range rows {
		got[r.Path] = r.Status
	}
	want := map[string]string{
		".agents/skills/alpha":  stateModified,
		".claude/skills/alpha":  stateMissing,
		".agents/skills/manual": stateUntracked,
	}
	if len(got) != len(want) {
		t.Fatalf("rows = %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q (all: %v)", k, got[k], v, got)
		}
	}

	out, err := run(t, "skill", "list", "--installed")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, w := range []string{"modified", "missing", "untracked", "(not recorded in .hikma/lock.json)"} {
		if !strings.Contains(out, w) {
			t.Errorf("table missing %q:\n%s", w, out)
		}
	}
}

func TestListInstalledOK(t *testing.T) {
	installAlpha(t, "claude")
	out, err := run(t, "skill", "list", "--installed")
	if err != nil || !strings.Contains(out, ".claude/skills/alpha") || !strings.Contains(out, "ok") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestListInstalledWithNothingInstalledAndWithASource(t *testing.T) {
	project(t)
	out, err := run(t, "skill", "list", "--installed")
	if err != nil || !strings.Contains(out, "No skills installed") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := run(t, "skill", "list", "--installed", "owner/repo"); err == nil {
		t.Fatal("--installed with a source should be an error")
	}
}
