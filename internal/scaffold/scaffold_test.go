package scaffold_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
)

func TestScaffoldClaudeWritesAgentFilesWithoutBundledSkill(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(scaffold.Options{
		ProjectName: "demo",
		Owner:       "owner",
		Technology:  "Go",
		Profile:     config.ProfileClaude,
		Path:        dir,
	})
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}
	if len(result.Created) == 0 {
		t.Fatalf("Scaffold() created no files")
	}
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills")); !os.IsNotExist(err) {
		t.Fatalf("scaffold should not create bundled skill registry content")
	}
}
