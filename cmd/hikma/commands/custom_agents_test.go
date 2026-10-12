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
