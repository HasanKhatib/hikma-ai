package source_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/source"
)

func checkContent(t *testing.T, name, content string) []source.Issue {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return source.CheckSkill(dir, name, false)
}

func hasIssue(issues []source.Issue, severity, substr string) bool {
	for _, i := range issues {
		if i.Severity == severity && strings.Contains(i.Message, substr) {
			return true
		}
	}
	return false
}

func TestCheckSkillSpecClean(t *testing.T) {
	issues := checkContent(t, "good-skill", "---\nname: good-skill\ndescription: Does a thing. Use when asked.\nmetadata:\n  owner: me\n---\n# body\n")
	if len(issues) != 0 {
		t.Fatalf("issues = %v", issues)
	}
}

func TestCheckSkillSpecRules(t *testing.T) {
	long := strings.Repeat("x", 1025)
	tests := []struct {
		name, fm, severity, want string
	}{
		{"bad-metadata", "name: bad-metadata\ndescription: d\nmetadata:\n  last_validated: 2026-10-10\n", "error", "metadata.last_validated must be a string"},
		{"bad-metadata", "name: bad-metadata\ndescription: d\nmetadata:\n  email:\n", "error", "metadata.email must be a string"},
		{"long-desc", "name: long-desc\ndescription: " + long + "\n", "error", "description is longer than 1024"},
		{"compat", "name: compat\ndescription: d\ncompatibility: " + strings.Repeat("y", 501) + "\n", "error", "compatibility must be"},
		{"extra", "name: extra\ndescription: d\nwhen_to_use: now\nmodel: x\n", "warning", "model, when_to_use"},
		{"Bad_Name", "name: Bad_Name\ndescription: d\n", "warning", "outside the Agent Skills spec"},
		{"double--hyphen", "name: double--hyphen\ndescription: d\n", "warning", "outside the Agent Skills spec"},
	}
	for _, tc := range tests {
		issues := checkContent(t, tc.name, "---\n"+tc.fm+"---\nbody\n")
		if !hasIssue(issues, tc.severity, tc.want) {
			t.Errorf("%s: want %s containing %q, got %v", tc.name, tc.severity, tc.want, issues)
		}
	}
}

func TestCheckSkillLongBodyWarns(t *testing.T) {
	body := strings.Repeat("line\n", 600)
	issues := checkContent(t, "big", "---\nname: big\ndescription: d\n---\n"+body)
	if !hasIssue(issues, "warning", "recommends under 500") || source.HasErrors(issues) {
		t.Fatalf("issues = %v", issues)
	}
}

func TestReadProvenance(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: pdf\ndescription: d\nmetadata:\n    github-path: skills/pdf\n    github-ref: refs/tags/v1.2.0\n    github-repo: https://github.com/anthropics/skills\n    github-tree-sha: abc\n---\n"), 0o644)
	p, ok := source.ReadProvenance(dir)
	if !ok || p.Repo != "https://github.com/anthropics/skills" || p.TreeSHA != "abc" {
		t.Fatalf("p = %+v ok = %v", p, ok)
	}
	if src, ref := p.SourceRef(); src != p.Repo || ref != "v1.2.0" {
		t.Fatalf("source %q ref %q", src, ref)
	}
	p.Pinned = "deadbeef"
	if _, ref := p.SourceRef(); ref != "deadbeef" {
		t.Fatalf("pinned ref = %q", ref)
	}

	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: x\ndescription: d\n---\n"), 0o644)
	if _, ok := source.ReadProvenance(dir); ok {
		t.Fatal("no metadata should not count")
	}
}
