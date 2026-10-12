package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const acmeAgent = `{"customAgents":{"acme":{"skillsDir":".acme/skills","instructionFile":"ACME.md"}}}`

func TestInitAndInstallUseACustomAgent(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"), acmeAgent)

	out, err := run(t, "init", "--agent", "claude,acme", "--registry", reg, "--skill", "alpha",
		"--project-name", "demo", "--owner", "me", "--technology", "Go")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	for _, rel := range []string{
		"AGENTS.md", "ACME.md",
		".claude/skills/alpha/SKILL.md", ".acme/skills/alpha/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(proj, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join(proj, ".hikma", "config.json"))
	if !strings.Contains(string(cfg), `"acme"`) || !strings.Contains(string(cfg), ".acme/skills") {
		t.Errorf("init dropped the customAgents key:\n%s", cfg)
	}
}

func TestSyncRestoresACustomAgent(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"), acmeAgent)
	if out, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "acme"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if err := os.RemoveAll(filepath.Join(proj, ".acme")); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "sync", "--yes")
	if err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".acme", "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatalf("not restored: %v\n%s", err, out)
	}
}

func TestSyncRejectsACustomFolderWithoutItsDefinition(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	cfgPath := filepath.Join(proj, ".hikma", "config.json")
	write(t, cfgPath, acmeAgent)
	if out, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "acme"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	// The definition is gone, so the lock entry points outside every known skills folder.
	if err := os.Remove(cfgPath); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(proj, ".acme")); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "sync", "--yes")
	if err == nil {
		t.Fatalf("sync should reject the entry:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(proj, ".acme")); statErr == nil {
		t.Fatal("sync wrote into a folder no agent declares")
	}
}

func TestUnknownAgentListsCustomAgents(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"), acmeAgent)
	_, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "nope")
	if err == nil || !strings.Contains(err.Error(), "acme") {
		t.Fatalf("error should list acme, got %v", err)
	}
}

func TestInvalidCustomAgentInConfigFailsLoudly(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"),
		`{"customAgents":{"evil":{"skillsDir":"../outside"}}}`)
	_, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "claude")
	if err == nil || !strings.Contains(err.Error(), "customAgents.evil.skillsDir") {
		t.Fatalf("want a custom agent error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(proj, ".claude")); statErr == nil {
		t.Fatal("nothing should be installed when the config is invalid")
	}
}

func TestForcedSyncNeverReplacesAFolderThatIsNotASkill(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"),
		`{"customAgents":{"src":{"skillsDir":"src"}}}`)
	if out, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "src"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	// Replace the installed skill with project code that is not a skill.
	if err := os.RemoveAll(filepath.Join(proj, "src", "alpha")); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(proj, "src", "alpha", "main.go")
	write(t, keep, "package main\n")

	out, err := run(t, "sync", "--yes", "--force")
	if err == nil {
		t.Fatalf("sync --force should fail on a non-skill folder:\n%s", out)
	}
	if _, statErr := os.Stat(keep); statErr != nil {
		t.Fatalf("project file was deleted: %v", statErr)
	}
	out, err = run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "src", "--force")
	if err == nil {
		t.Fatalf("install --force should fail on a non-skill folder:\n%s", out)
	}
	if _, statErr := os.Stat(keep); statErr != nil {
		t.Fatalf("project file was deleted: %v", statErr)
	}
}

func TestAddAgentWritesAndIsUsable(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)

	out, err := run(t, "config", "add-agent", "acme", "--skills-dir", ".acme/skills",
		"--instruction-file", "ACME.md", "--project", "--use")
	if err != nil {
		t.Fatalf("add-agent: %v\n%s", err, out)
	}
	cfgPath := filepath.Join(proj, ".hikma", "config.json")
	if !strings.Contains(out, cfgPath) && !strings.Contains(out, ".hikma/config.json") {
		t.Errorf("output should name the file written:\n%s", out)
	}
	cfg, _ := os.ReadFile(cfgPath)
	for _, want := range []string{`"customAgents"`, `".acme/skills"`, `"ACME.md"`, `"agents"`, `"acme"`} {
		if !strings.Contains(string(cfg), want) {
			t.Errorf("config missing %s:\n%s", want, cfg)
		}
	}
	// The folder is not created by add-agent.
	if _, err := os.Stat(filepath.Join(proj, ".acme")); err == nil {
		t.Error("add-agent must not create the skills folder")
	}
	// Selecting it by name works, and the first install creates the folder.
	if out, err := run(t, "config", "set", "agent", "acme", "--project"); err != nil {
		t.Fatalf("set agent: %v\n%s", err, out)
	}
	if out, err := run(t, "skill", "install", fileURL(reg), "alpha"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".acme", "skills", "alpha", "SKILL.md")); err != nil {
		t.Errorf("skill not installed into the custom agent folder: %v", err)
	}
	list, _ := run(t, "config", "list")
	if !strings.Contains(list, "agent acme: .acme/skills") {
		t.Errorf("config list should show the custom agent:\n%s", list)
	}
}

func TestAddAgentRefusals(t *testing.T) {
	project(t)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"built-in name", []string{"config", "add-agent", "claude", "--skills-dir", "x/skills", "--project"}, "built-in"},
		{"escapes project", []string{"config", "add-agent", "acme", "--skills-dir", "../x", "--project"}, "must not contain .."},
		{"inside .git", []string{"config", "add-agent", "acme", "--skills-dir", ".GIT/skills", "--project"}, ".GIT"},
		{"no folder without a terminal", []string{"config", "add-agent", "acme", "--project"}, "--skills-dir is required"},
		{"no scope without a terminal", []string{"config", "add-agent", "acme", "--skills-dir", "a/skills"}, "--project or --user"},
		{"both scopes", []string{"config", "add-agent", "acme", "--skills-dir", "a/skills", "--project", "--user"}, "not both"},
	}
	for _, c := range cases {
		out, err := run(t, c.args...)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want error containing %q, got %v\n%s", c.name, c.want, err, out)
		}
	}
	if _, err := os.Stat(".hikma/config.json"); err == nil {
		t.Error("a refused add-agent must not write the config")
	}
}

func TestAddAgentDoesNotReplaceWithoutForce(t *testing.T) {
	proj := project(t)
	args := []string{"config", "add-agent", "acme", "--skills-dir", ".acme/skills", "--project"}
	if out, err := run(t, args...); err != nil {
		t.Fatalf("first add: %v\n%s", err, out)
	}
	args[4] = ".other/skills"
	if _, err := run(t, args...); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("want a --force hint, got %v", err)
	}
	cfg, _ := os.ReadFile(filepath.Join(proj, ".hikma", "config.json"))
	if !strings.Contains(string(cfg), ".acme/skills") {
		t.Fatalf("existing definition was changed:\n%s", cfg)
	}
	if out, err := run(t, append(args, "--force")...); err != nil {
		t.Fatalf("forced add: %v\n%s", err, out)
	}
	cfg, _ = os.ReadFile(filepath.Join(proj, ".hikma", "config.json"))
	if !strings.Contains(string(cfg), ".other/skills") {
		t.Fatalf("--force did not replace:\n%s", cfg)
	}
}

func TestRemoveAgentCleansReferences(t *testing.T) {
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"),
		`{"agent":"acme","agents":["claude","acme"],"customAgents":{"acme":{"skillsDir":".acme/skills"}}}`)

	if _, err := run(t, "config", "remove-agent", "acme", "--project"); err == nil ||
		!strings.Contains(err.Error(), "--yes") {
		t.Fatalf("remove without a terminal needs --yes, got %v", err)
	}
	out, err := run(t, "config", "remove-agent", "acme", "--yes")
	if err != nil {
		t.Fatalf("remove-agent: %v\n%s", err, out)
	}
	cfg, _ := os.ReadFile(filepath.Join(proj, ".hikma", "config.json"))
	if strings.Contains(string(cfg), "acme") || !strings.Contains(string(cfg), "claude") {
		t.Fatalf("references not cleaned:\n%s", cfg)
	}
	// The config must still load.
	if out, err := run(t, "config", "list"); err != nil {
		t.Fatalf("config broken after remove: %v\n%s", err, out)
	}
	if _, err := run(t, "config", "remove-agent", "acme", "--yes"); err == nil {
		t.Fatal("removing an undefined agent should fail")
	}
}

func TestOverlapWarningOnlyWhenAnAgentReadsBothFolders(t *testing.T) {
	reg := registryRepo(t)
	project(t)
	out, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "claude,copilot")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "warning:") || !strings.Contains(out, ".claude/skills") {
		t.Errorf("claude+copilot should warn:\n%s", out)
	}
	if !strings.Contains(out, "Installing") {
		t.Errorf("the install must still proceed:\n%s", out)
	}
	if n := strings.Count(out, "warning:"); n != 1 {
		t.Errorf("want one warning, got %d:\n%s", n, out)
	}

	project(t)
	out, err = run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "claude,codex")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if strings.Contains(out, "warning:") {
		t.Errorf("claude+codex is fine and must not warn:\n%s", out)
	}
}
