package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// ghInstalled puts alpha into .claude/skills the way `gh skill install` does:
// the files plus tracking metadata (the ref is omitted because the test registry is a local path) in the SKILL.md frontmatter.
func ghInstalled(t *testing.T, reg string) string {
	t.Helper()
	proj := project(t)
	skill := "---\nname: alpha\ndescription: alpha skill\nmetadata:\n    github-path: skills/alpha\n    github-repo: " + reg + "\n    github-tree-sha: abc123\n---\n# alpha\n"
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), skill)
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "scripts", "run.sh"), "echo v1\n")
	return proj
}

func TestGhSkillInstallIsDetectedAsUntracked(t *testing.T) {
	reg := registryRepo(t)
	ghInstalled(t, reg)

	out, err := run(t, "skill", "list", "--installed")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "installed by gh skill") || !strings.Contains(out, "hikma skill adopt alpha") {
		t.Fatalf("list output:\n%s", out)
	}
	rows, err := listInstalledSkills()
	if err != nil || len(rows) != 1 || rows[0].Origin != "gh skill" || rows[0].Status != stateUntracked {
		t.Fatalf("rows = %+v, err = %v", rows, err)
	}
}

func TestUpdateOfGhSkillPointsToAdopt(t *testing.T) {
	ghInstalled(t, registryRepo(t))
	out, err := run(t, "skill", "update", "alpha", "--agent", "claude")
	if err == nil {
		t.Fatalf("expected an error\n%s", out)
	}
	if !strings.Contains(err.Error(), "hikma skill adopt alpha") {
		t.Fatalf("err = %v\n%s", err, out)
	}
}

func TestAdoptRecordsLockAndUpdateWorks(t *testing.T) {
	reg := registryRepo(t)
	proj := ghInstalled(t, reg)

	out, err := run(t, "skill", "adopt", "alpha", "--agent", "claude")
	if err != nil {
		t.Fatalf("adopt: %v\n%s", err, out)
	}
	lf, _ := lock.Load(".")
	e, ok := lf.Skills[".claude/skills/alpha"]
	if !ok || e.Source != reg || e.Commit == "" || len(e.Files) == 0 {
		t.Fatalf("lock entry = %+v ok=%v", e, ok)
	}

	// Adopting again is a no-op, and the skill now shows as tracked.
	if out, err = run(t, "skill", "adopt", "alpha", "--agent", "claude"); err == nil {
		t.Fatalf("second adopt should find nothing to adopt\n%s", out)
	}
	out, _ = run(t, "skill", "list", "--installed")
	if strings.Contains(out, "untracked") {
		t.Fatalf("still untracked:\n%s", out)
	}

	// Update treats it like any hikma install: no local edits, so it pulls the registry's version.
	if out, err = run(t, "skill", "update", "alpha", "--agent", "claude"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !exists(filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")) {
		t.Fatal("skill vanished")
	}
}

func TestAdoptNeedsASourceWithoutGhMetadata(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), "---\nname: alpha\ndescription: alpha skill\n---\n")

	if out, err := run(t, "skill", "adopt", "alpha", "--agent", "claude"); err == nil || !strings.Contains(err.Error(), "--source") {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if out, err := run(t, "skill", "adopt", "alpha", "--agent", "claude", "--source", reg); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestAdoptRejectsASourceWithoutTheSkill(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".claude", "skills", "ghost", "SKILL.md"), "---\nname: ghost\ndescription: d\n---\n")
	if out, err := run(t, "skill", "adopt", "ghost", "--agent", "claude", "--source", reg); err == nil {
		t.Fatalf("expected not found\n%s", out)
	}
}

func TestUpdateAllMentionsGhSkills(t *testing.T) {
	reg := registryRepo(t)
	ghInstalled(t, reg)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "codex"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	out, err := run(t, "skill", "update", "--all")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "installed by gh skill") || !strings.Contains(out, "alpha") {
		t.Fatalf("output:\n%s", out)
	}
}
