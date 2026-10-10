package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryValidateGitHubAnnotations(t *testing.T) {
	reg := registryRepo(t)
	write(t, filepath.Join(reg, "skills", "broken", "SKILL.md"), "---\nname: broken\ndescription: Replace this description. now\nwhen_to_use: x\n---\n\n- Replace with clear steps for the agent.\n")
	project(t)

	out, err := run(t, "registry", "validate", reg, "--format", "github")
	if err == nil {
		t.Fatalf("expected failure\n%s", out)
	}
	for _, want := range []string{
		"::error file=skills/broken/SKILL.md,line=3::unfilled placeholder",
		"::error file=skills/broken/SKILL.md,line=7::unfilled placeholder",
		"::warning file=skills/broken/SKILL.md::fields outside the Agent Skills spec: when_to_use",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "::error file=skills/alpha") {
		t.Errorf("clean skill got an annotation:\n%s", out)
	}
	if out, _ := run(t, "registry", "validate", reg); strings.Contains(out, "::error") {
		t.Errorf("text format printed annotations:\n%s", out)
	}
	if _, err := run(t, "registry", "validate", reg, "--format", "xml"); err == nil {
		t.Error("unknown format accepted")
	}
}

func TestEscapeAnnotationText(t *testing.T) {
	if got := escapeData("a%b\nc"); got != "a%25b%0Ac" {
		t.Errorf("data = %q", got)
	}
	if got := escapeProperty("a:b,c"); got != "a%3Ab%2Cc" {
		t.Errorf("property = %q", got)
	}
}
