package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/config"
)

// isolate points user config and the working directory at temp dirs.
func isolate(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("HIKMA_AGENT", "")
	t.Setenv("HIKMA_REGISTRY", "")
	t.Setenv("HIKMA_NAMING", "")
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	t.Setenv("USERPROFILE", home)
	repo = t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return home, repo
}

func TestLookupPrecedence(t *testing.T) {
	isolate(t)

	e, _ := config.Lookup(config.KeyAgent)
	if e.Value != "copilot" || e.Source != config.SourceDefault {
		t.Fatalf("default = %+v", e)
	}

	if err := config.Set(config.ScopeUser, "registry", "user/reg"); err != nil {
		t.Fatal(err)
	}
	e, _ = config.Lookup(config.KeyRegistry)
	if e.Value != "user/reg" || e.Source != config.SourceUser {
		t.Fatalf("user = %+v", e)
	}

	if err := config.Set(config.ScopeProject, "registry", "proj/reg"); err != nil {
		t.Fatal(err)
	}
	e, _ = config.Lookup(config.KeyRegistry)
	if e.Value != "proj/reg" || e.Source != config.SourceProject {
		t.Fatalf("project = %+v", e)
	}

	t.Setenv("HIKMA_REGISTRY", "env/reg")
	e, _ = config.Lookup(config.KeyRegistry)
	if e.Value != "env/reg" || e.Source != config.SourceEnv {
		t.Fatalf("env = %+v", e)
	}

	r, err := config.ResolveRegistry("flag/reg")
	if err != nil || r.FullName() != "flag/reg" {
		t.Fatalf("flag = %+v, %v", r, err)
	}
}

func TestProjectConfigFoundFromSubdirectory(t *testing.T) {
	_, repo := isolate(t)
	if err := config.Set(config.ScopeProject, "agent", "claude"); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	e, _ := config.Lookup(config.KeyAgent)
	if e.Value != "claude" || e.Source != config.SourceProject {
		t.Fatalf("lookup = %+v", e)
	}
}

func TestSetValidatesAndUnsetRemoves(t *testing.T) {
	isolate(t)
	if err := config.Set(config.ScopeUser, "agent", "vim"); err == nil {
		t.Fatal("expected invalid agent error")
	}
	if err := config.Set(config.ScopeUser, "registry", "nope"); err == nil {
		t.Fatal("expected invalid registry error")
	}
	if err := config.Set(config.ScopeUser, "color", "red"); err == nil {
		t.Fatal("expected unknown key error")
	}
	if err := config.Set(config.ScopeUser, "agent", "claude"); err != nil {
		t.Fatal(err)
	}
	if err := config.Unset(config.ScopeUser, "agent"); err != nil {
		t.Fatal(err)
	}
	e, _ := config.Lookup(config.KeyAgent)
	if e.Source != config.SourceDefault {
		t.Fatalf("after unset = %+v", e)
	}
}

func TestResolveRegistryErrorsWhenUnset(t *testing.T) {
	isolate(t)
	if _, err := config.ResolveRegistry(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestNamingDefaultsToLooseAndValidates(t *testing.T) {
	isolate(t)
	n, err := config.NamingConvention()
	if err != nil || n != config.NamingLoose {
		t.Fatalf("default = %q, %v", n, err)
	}
	if err := config.Set(config.ScopeUser, "naming", "camel"); err == nil {
		t.Fatal("expected invalid naming error")
	}
	if err := config.Set(config.ScopeProject, "naming", "kebab-case"); err != nil {
		t.Fatal(err)
	}
	if n, _ := config.NamingConvention(); n != config.NamingKebabCase {
		t.Fatalf("project naming = %q", n)
	}
}

func TestCheckSkillName(t *testing.T) {
	tests := []struct {
		naming, name string
		ok           bool
	}{
		{config.NamingLoose, "My_Skill v2", true},
		{config.NamingLoose, "../evil", false},
		{config.NamingLoose, "a/b", false},
		{config.NamingLoose, `a\b`, false},
		{config.NamingLoose, "..", false},
		{config.NamingLoose, "", false},
		{config.NamingKebabCase, "my-skill-2", true},
		{config.NamingKebabCase, "My_Skill", false},
	}
	for _, tt := range tests {
		err := config.CheckSkillName(tt.naming, tt.name)
		if (err == nil) != tt.ok {
			t.Errorf("CheckSkillName(%q, %q) err = %v, want ok=%v", tt.naming, tt.name, err, tt.ok)
		}
	}
}

func TestUnsetRemovesEmptyProjectFile(t *testing.T) {
	_, repo := isolate(t)
	if err := config.Set(config.ScopeProject, "agent", "claude"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, ".hikma", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := config.Unset(config.ScopeProject, "agent"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err = %v", err)
	}
}

func TestProjectLookupStopsAtHome(t *testing.T) {
	home, _ := isolate(t)
	if err := os.MkdirAll(filepath.Join(home, ".hikma"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".hikma", "config.json"), []byte(`{"agent":"claude"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(home, "work", "proj")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	if p, _ := config.ProjectConfigPath(); p != "" {
		t.Fatalf("project path = %q, want none", p)
	}
}

func TestAgentsListAndResolveTargets(t *testing.T) {
	isolate(t)
	if err := config.Set(config.ScopeProject, "agents", "claude, codex,opencode"); err != nil {
		t.Fatal(err)
	}
	if err := config.Set(config.ScopeProject, "agents", "claude,vim"); err == nil {
		t.Fatal("expected invalid agent in list")
	}
	e, _ := config.Lookup(config.KeyAgents)
	if e.Value != "claude,codex,opencode" || e.Source != config.SourceProject {
		t.Fatalf("lookup = %+v", e)
	}

	// codex and opencode share .agents/skills, so they collapse into one target.
	targets, err := config.ResolveTargets("")
	if err != nil || len(targets) != 2 || targets[0].Agent != config.AgentClaude || targets[1].Layout.SkillsDir != ".agents/skills" {
		t.Fatalf("targets = %+v, %v", targets, err)
	}

	// The flag wins over config.
	targets, _ = config.ResolveTargets("codex")
	if len(targets) != 1 || targets[0].Agent != config.AgentCodex {
		t.Fatalf("flag targets = %+v", targets)
	}
	if _, err := config.ResolveTargets("claude,nope"); err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestResolveTargetsFallsBackToSingleAgent(t *testing.T) {
	isolate(t)
	targets, _ := config.ResolveTargets("")
	if len(targets) != 1 || targets[0].Agent != config.AgentCopilot {
		t.Fatalf("default = %+v", targets)
	}
	if err := config.Set(config.ScopeUser, "agent", "claude"); err != nil {
		t.Fatal(err)
	}
	targets, _ = config.ResolveTargets("")
	if len(targets) != 1 || targets[0].Layout.SkillsDir != ".claude/skills" {
		t.Fatalf("configured = %+v", targets)
	}
}

func TestInstructionFilesAreDeduplicated(t *testing.T) {
	got := config.InstructionFiles([]config.Agent{config.AgentClaude, config.AgentCodex})
	if len(got) != 1 || got[0] != "AGENTS.md" {
		t.Fatalf("files = %v", got)
	}
}
