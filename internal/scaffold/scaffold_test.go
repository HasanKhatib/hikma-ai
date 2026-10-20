package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
	"gopkg.in/yaml.v3"
)

func opts(dir string, agents ...config.Agent) scaffold.Options {
	return scaffold.Options{ProjectName: "demo", Owner: "owner", Technology: "Go", Agents: agents, Path: dir}
}

func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, rel))
	return err == nil
}

func TestScaffoldClaudeWritesAgentFilesWithoutBundledSkill(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(opts(dir, config.AgentClaude))
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}
	if len(result.Created) != 1 || !exists(dir, "AGENTS.md") || exists(dir, "CLAUDE.md") {
		t.Fatalf("created = %v", result.Created)
	}
	if exists(dir, ".claude/skills") || exists(dir, ".agents") {
		t.Fatalf("scaffold must not create skill folders or other agents' files")
	}
}

func TestScaffoldNonClaudeWritesOnlyAgentsMD(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(opts(dir, config.AgentCodex)); err != nil {
		t.Fatal(err)
	}
	if !exists(dir, "AGENTS.md") || exists(dir, "CLAUDE.md") {
		t.Fatal("codex should get AGENTS.md only")
	}
}

func TestScaffoldMultipleAgentsWritesSharedFilesOnce(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(opts(dir, config.AgentClaude, config.AgentCodex))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 {
		t.Fatalf("created = %v", result.Created)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(data), "`.claude/skills`, `.agents/skills`") {
		t.Fatalf("AGENTS.md should list every skills folder:\n%s", data)
	}
}

func TestScaffoldOutputIsNeutral(t *testing.T) {
	dir := t.TempDir()
	o := opts(dir, config.AgentClaude)
	o.Registry = "someone/skills"
	if _, err := scaffold.Scaffold(o); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	text := string(data)
	if !strings.Contains(text, "someone/skills") {
		t.Fatalf("configured registry should be mentioned:\n%s", text)
	}
	for _, banned := range []string{"hasankhatib", "opencode", "hikma skill list"} {
		if strings.Contains(strings.ToLower(text), banned) {
			t.Fatalf("generated file mentions %q:\n%s", banned, text)
		}
	}
	// Without a registry the file must not mention one.
	dir2 := t.TempDir()
	if _, err := scaffold.Scaffold(opts(dir2, config.AgentClaude)); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(dir2, "AGENTS.md"))
	if strings.Contains(string(data), "project registry") {
		t.Fatalf("no registry configured, but one is mentioned:\n%s", data)
	}
}

func TestScaffoldDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	o := opts(dir, config.AgentClaude)
	o.DryRun = true
	result, err := scaffold.Scaffold(o)
	if err != nil || len(result.Created) != 1 {
		t.Fatalf("%v %v", result, err)
	}
	if exists(dir, "AGENTS.md") {
		t.Fatal("dry run wrote a file")
	}
}

func TestScaffoldRejectsBadAgents(t *testing.T) {
	if _, err := scaffold.Scaffold(opts(t.TempDir())); err == nil {
		t.Fatal("expected error for no agents")
	}
	if _, err := scaffold.Scaffold(opts(t.TempDir(), "vim")); err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestCreateSkillKeepsTheFrontmatterValidForAnyDescription(t *testing.T) {
	for _, desc := range []string{
		`Say "hello": a quote and a colon`,
		`C:\temp\files and a trailing backslash\`,
		"two\nlines and a\ttab",
		"  padded  ",
		"",
	} {
		dir := filepath.Join(t.TempDir(), "my-skill")
		if _, err := scaffold.CreateSkill(scaffold.SkillOptions{SkillName: "my-skill", Owner: `o"wner`, Team: `t\eam`, Description: desc, TargetDir: dir}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		// The template has CRLF line endings when checked out on Windows.
		_, rest, _ := strings.Cut(strings.ReplaceAll(string(data), "\r\n", "\n"), "---\n")
		front, _, _ := strings.Cut(rest, "\n---")
		var meta struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Metadata    struct {
				Owner string `yaml:"owner"`
				Team  string `yaml:"team"`
			} `yaml:"metadata"`
		}
		if err := yaml.Unmarshal([]byte(front), &meta); err != nil {
			t.Fatalf("description %q gave frontmatter that does not parse: %v\n%s", desc, err, front)
		}
		want := strings.Join(strings.Fields(desc), " ")
		if want != "" && meta.Description != want {
			t.Errorf("description %q came back as %q, want %q", desc, meta.Description, want)
		}
		if want == "" && !strings.HasPrefix(meta.Description, "Replace this description") {
			t.Errorf("no description should keep the placeholder, got %q", meta.Description)
		}
		if meta.Metadata.Owner != `o"wner` || meta.Metadata.Team != `t\eam` {
			t.Errorf("owner and team = %q, %q", meta.Metadata.Owner, meta.Metadata.Team)
		}
	}
}
