package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// run executes the hikma CLI in-process and returns combined output.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := Root()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// project isolates config and moves into an empty project directory.
func project(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	t.Setenv("USERPROFILE", home)
	for _, k := range []string{"HIKMA_AGENT", "HIKMA_REGISTRY", "HIKMA_NAMING"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return dir
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

const skillMD = "---\nname: %s\ndescription: %s skill\nmetadata:\n  owner: someone\n---\n# %s\n"

// registryRepo builds a git repo with two skills under skills/.
func registryRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git(t, dir, "init", "--quiet", "-b", "main")
	write(t, filepath.Join(dir, "skills", "alpha", "SKILL.md"), strings.NewReplacer("%s", "alpha").Replace(skillMD))
	write(t, filepath.Join(dir, "skills", "alpha", "scripts", "run.sh"), "echo v1\n")
	write(t, filepath.Join(dir, "skills", "My Skill_v2", "SKILL.md"), strings.NewReplacer("%s", "loose").Replace(skillMD))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "--quiet", "-m", "init")
	return dir
}

func TestInstallFromLocalPathRecordsLock(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)

	out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatalf("skill not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "alpha", ".git")); err == nil {
		t.Fatal(".git must not be copied")
	}
	lf, err := lock.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	e, ok := lf.Skills[".claude/skills/alpha"]
	if !ok || e.Name != "alpha" || e.Source != reg || e.Commit == "" || len(e.Files) == 0 {
		t.Fatalf("lock entry = %+v ok=%v", e, ok)
	}
	if !strings.Contains(out, "Source: ") || !strings.Contains(out, "Commit: ") {
		t.Fatalf("install output should name the source and commit:\n%s", out)
	}
}

func TestInstallBareNameUsesConfiguredRegistry(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	t.Setenv("HIKMA_REGISTRY", reg)

	out, err := run(t, "skill", "install", "alpha", "--agent", "codex")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "env HIKMA_REGISTRY") {
		t.Fatalf("output should say where the registry came from:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".agents", "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallBareNameWithoutRegistryExplainsHowToFix(t *testing.T) {
	project(t)
	_, err := run(t, "skill", "install", "alpha")
	if err == nil || !strings.Contains(err.Error(), "no registry configured") {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallEnforcesNoNamingConvention(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "config", "set", "naming", "kebab-case"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := run(t, "skill", "install", reg, "My Skill_v2", "--agent", "claude"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "My Skill_v2", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallFromGitURL(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	url := "file://" + filepath.ToSlash(reg)
	if !strings.HasPrefix(url, "file:///") {
		url = "file:///" + strings.TrimPrefix(url, "file://")
	}
	out, err := run(t, "skill", "install", url, "alpha", "--agent", "claude")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallWithoutNameListsSkillsWhenNotInteractive(t *testing.T) {
	reg := registryRepo(t)
	project(t)
	out, err := run(t, "skill", "install", reg)
	if err == nil {
		t.Fatal("expected an error asking for a skill name")
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "My Skill_v2") {
		t.Fatalf("expected skill list in output:\n%s", out)
	}
}

func TestInstallRefusesExistingWithoutForce(t *testing.T) {
	reg := registryRepo(t)
	project(t)
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err == nil {
		t.Fatal("expected already-installed error")
	}
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude", "--force"); err != nil {
		t.Fatalf("--force: %v\n%s", err, out)
	}
}

func TestListJSON(t *testing.T) {
	reg := registryRepo(t)
	project(t)
	cmd := Root()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skill", "list", reg, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	// printJSON writes to the process stdout; the status lines go to stderr here.
	if strings.Contains(stdout.String(), "Source:") {
		t.Fatalf("status text leaked into stdout: %q", stdout.String())
	}
}

func TestUpdateUsesRecordedSourceAndProtectsLocalEdits(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatal(err)
	}

	// No registry is configured at all: update must still find the original source.
	out, err := run(t, "skill", "update", "alpha", "--agent", "claude")
	if err != nil || !strings.Contains(out, "up to date") {
		t.Fatalf("expected up to date: %v\n%s", err, out)
	}

	// Upstream changes a script.
	write(t, filepath.Join(reg, "skills", "alpha", "scripts", "run.sh"), "echo v2\n")
	git(t, reg, "commit", "--quiet", "-am", "v2")

	// A local edit blocks the update.
	local := filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")
	write(t, local, "edited\n")
	if out, err := run(t, "skill", "update", "alpha", "--agent", "claude"); err == nil {
		t.Fatalf("expected local-changes error:\n%s", out)
	}

	out, err = run(t, "skill", "update", "alpha", "--agent", "claude", "--force")
	if err != nil {
		t.Fatalf("--force update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "scripts/run.sh") || !strings.Contains(out, "scripts/ changed") {
		t.Fatalf("expected a scripts warning:\n%s", out)
	}
	data, _ := os.ReadFile(filepath.Join(proj, ".claude", "skills", "alpha", "scripts", "run.sh"))
	if string(data) != "echo v2\n" {
		t.Fatalf("script not updated: %q", data)
	}
}

func TestUpdateAllSkipsUntrackedSkills(t *testing.T) {
	reg := registryRepo(t)
	project(t)
	if _, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "skill", "update", "--all")
	if err != nil || !strings.Contains(out, "Done: 1 processed, 0 skipped") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestConfirmPushRequiresYesWhenNotInteractive(t *testing.T) {
	cmd := Root()
	var out bytes.Buffer
	cmd.SetOut(&out)
	r, _, err := resolvePushRegistry("owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if err := confirmPush(cmd, "alpha", ".claude/skills/alpha", r, "--registry flag", false); err == nil {
		t.Fatal("expected confirmation error")
	}
	if !strings.Contains(out.String(), "owner/repo") || !strings.Contains(out.String(), "--registry flag") {
		t.Fatalf("gate must show the target and its origin:\n%s", out.String())
	}
	if err := confirmPush(cmd, "alpha", ".claude/skills/alpha", r, "--registry flag", true); err != nil {
		t.Fatalf("--yes: %v", err)
	}
}

func TestPushRequiresGitHubRegistry(t *testing.T) {
	project(t)
	if _, _, err := resolvePushRegistry("/some/local/path"); err == nil {
		t.Fatal("expected error for non-GitHub registry")
	}
}

func TestInitSetsUpSelectedAgentsAndInstallsSkills(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)

	out, err := run(t, "init", "--agent", "claude,codex", "--registry", reg, "--skill", "alpha",
		"--project-name", "demo", "--owner", "me", "--technology", "Go")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	for _, rel := range []string{
		"AGENTS.md", "CLAUDE.md",
		".claude/skills/alpha/SKILL.md", ".agents/skills/alpha/SKILL.md",
		".hikma/config.json", ".hikma/lock.json",
	} {
		if _, err := os.Stat(filepath.Join(proj, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join(proj, ".hikma", "config.json"))
	for _, want := range []string{`"claude"`, `"codex"`, `"registry"`} {
		if !strings.Contains(string(cfg), want) {
			t.Errorf("project config missing %s:\n%s", want, cfg)
		}
	}

	// The shared project config now drives later commands: no flags needed.
	if out, err := run(t, "skill", "install", "My Skill_v2"); err != nil {
		t.Fatalf("install via project config: %v\n%s", err, out)
	}
	for _, d := range []string{".claude/skills/My Skill_v2", ".agents/skills/My Skill_v2"} {
		if _, err := os.Stat(filepath.Join(proj, d, "SKILL.md")); err != nil {
			t.Errorf("expected install in %s: %v", d, err)
		}
	}
}

func TestInitDryRunChangesNothing(t *testing.T) {
	proj := project(t)
	out, err := run(t, "init", "--agent", "claude", "--dry-run", "--project-name", "demo")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "would write AGENTS.md") || !strings.Contains(out, "would write CLAUDE.md") {
		t.Fatalf("output:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(proj, "AGENTS.md")); err == nil {
		t.Fatal("dry run wrote files")
	}
}

func TestInitSkillWithoutRegistryFailsBeforeWriting(t *testing.T) {
	proj := project(t)
	_, err := run(t, "init", "--agent", "claude", "--skill", "alpha", "--project-name", "demo")
	if err == nil || !strings.Contains(err.Error(), "needs a registry") {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(proj, "AGENTS.md")); statErr == nil {
		t.Fatal("files were written before the error")
	}
}

func TestInitSingleAgentOnlyTouchesItsOwnFolder(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "init", "--agent", "claude", "--registry", reg, "--skill", "alpha", "--project-name", "demo"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".agents")); err == nil {
		t.Fatal("claude-only init created .agents")
	}
}

func TestOverlappingAgentsShareOneSkillFolder(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "init", "--agent", "codex,copilot,opencode", "--registry", reg, "--skill", "alpha", "--project-name", "demo"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	lf, _ := lock.Load(proj)
	if len(lf.Skills) != 1 {
		t.Fatalf("lock entries = %v", lf.Keys())
	}
}
