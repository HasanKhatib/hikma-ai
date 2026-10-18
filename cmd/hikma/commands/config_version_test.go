package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSetRefusesANewerProjectConfig(t *testing.T) {
	proj := project(t)
	path := filepath.Join(proj, ".hikma", "config.json")
	body := `{"version":99,"registry":"acme/skills"}`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "config", "set", "naming", "kebab-case", "--project")
	if err == nil || !strings.Contains(out+err.Error(), "upgrade hikma") {
		t.Fatalf("want an upgrade error, got err=%v out=%s", err, out)
	}
	if got, _ := os.ReadFile(path); string(got) != body {
		t.Fatalf("config was changed: %s", got)
	}
}
