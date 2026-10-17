package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/source"
)

func TestPlaceholderSeverity(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "draft")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := "---\nname: draft\ndescription: A draft skill for testing.\n---\n# Draft\n\n- Replace with clear steps for the agent.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		sev     string
		wantErr bool
	}{{source.SeverityError, true}, {source.SeverityWarning, false}} {
		issues, err := validateSkillDirSeverity(dir, "draft", false, tc.sev)
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, i := range issues {
			if strings.HasPrefix(i.Message, "unfilled placeholder") {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: placeholder not reported: %v", tc.sev, issues)
		}
		if source.HasErrors(issues) != tc.wantErr {
			t.Errorf("%s: HasErrors = %v, want %v", tc.sev, !tc.wantErr, tc.wantErr)
		}
	}
}

func TestNoProvenanceErrorOffersWaysForward(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HIKMA_REGISTRY", "")
	msg := noProvenanceError("vault-note", ".claude/skills/vault-note").Error()
	for _, want := range []string{
		"was not installed by gh skill",
		"hikma skill adopt vault-note --source <owner/repo>",
		"hikma skill push vault-note",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
}
