package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCustomAgents(t *testing.T) {
	ok := map[string]CustomAgent{
		"acme": {SkillsDir: ".acme/skills", InstructionFile: "ACME.md"},
		"min":  {SkillsDir: "tools/skills"},
	}
	if err := ValidateCustomAgents(ok); err != nil {
		t.Fatalf("valid custom agents rejected: %v", err)
	}
	bad := map[string]CustomAgent{
		"claude":        {SkillsDir: "x/skills"},
		"Acme":          {SkillsDir: "x/skills"},
		"empty":         {},
		"parent":        {SkillsDir: "../skills"},
		"nested-parent": {SkillsDir: "a/../../skills"},
		"abs":           {SkillsDir: "/etc/skills"},
		"drive":         {SkillsDir: "C:/skills"},
		"backslash":     {SkillsDir: `a\skills`},
		"dot":           {SkillsDir: "."},
		"unclean":       {SkillsDir: "a//skills"},
		"git":           {SkillsDir: ".git/skills"},
		"git-case":      {SkillsDir: ".GIT/skills"},
		"git-deep":      {SkillsDir: "a/.git/hooks"},
		"git-dots":      {SkillsDir: ".git./skills"},
		"hikma":         {SkillsDir: ".hikma/skills"},
		"instr-parent":  {SkillsDir: "ok/skills", InstructionFile: "../X.md"},
		"instr-git":     {SkillsDir: "ok/skills", InstructionFile: ".git/config"},
	}
	for name, l := range bad {
		if err := ValidateCustomAgents(map[string]CustomAgent{name: l}); err == nil {
			t.Errorf("custom agent %q %+v: expected an error", name, l)
		}
	}
}

func TestLoadFileValidatesCustomAgents(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	body := `{"customAgents":{"evil":{"skillsDir":"../outside"}}}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadFile(p)
	if err == nil || !strings.Contains(err.Error(), "customAgents.evil.skillsDir") {
		t.Fatalf("want a customAgents.evil.skillsDir error, got %v", err)
	}
	good := `{"customAgents":{"acme":{"skillsDir":".acme/skills","instructionFile":"ACME.md"}}}`
	if err := os.WriteFile(p, []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := loadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.CustomAgents["acme"].SkillsDir != ".acme/skills" {
		t.Fatalf("custom agent not loaded: %+v", c.CustomAgents)
	}
	if c.isEmpty() {
		t.Fatal("config with custom agents must not be empty")
	}
}
