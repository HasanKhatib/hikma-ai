package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const acmeLayout = `{"layouts":{"acme":{"skillsDir":".acme/skills","instructionFile":"ACME.md"}}}`

func TestInitAndInstallUseACustomLayout(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"), acmeLayout)

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
		t.Errorf("init dropped the layouts key:\n%s", cfg)
	}
}

func TestSyncRestoresACustomLayout(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"), acmeLayout)
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

func TestSyncRejectsACustomFolderWithoutItsLayout(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	cfgPath := filepath.Join(proj, ".hikma", "config.json")
	write(t, cfgPath, acmeLayout)
	if out, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "acme"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	// The layout is gone, so the lock entry points outside every known skills folder.
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
		t.Fatal("sync wrote into a folder no layout declares")
	}
}

func TestUnknownAgentListsCustomLayouts(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"), acmeLayout)
	_, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "nope")
	if err == nil || !strings.Contains(err.Error(), "acme") {
		t.Fatalf("error should list acme, got %v", err)
	}
}

func TestInvalidLayoutInConfigFailsLoudly(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"),
		`{"layouts":{"evil":{"skillsDir":"../outside"}}}`)
	_, err := run(t, "skill", "install", fileURL(reg), "alpha", "--agent", "claude")
	if err == nil || !strings.Contains(err.Error(), "layouts.evil.skillsDir") {
		t.Fatalf("want a layout error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(proj, ".claude")); statErr == nil {
		t.Fatal("nothing should be installed when the config is invalid")
	}
}

func TestForcedSyncNeverReplacesAFolderThatIsNotASkill(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	write(t, filepath.Join(proj, ".hikma", "config.json"),
		`{"layouts":{"src":{"skillsDir":"src"}}}`)
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
