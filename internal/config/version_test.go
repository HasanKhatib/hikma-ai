package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileTreatsAMissingVersionAsOneAndSaveStampsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"registry":"acme/skills"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := loadFile(path)
	if err != nil || c.Registry != "acme/skills" {
		t.Fatalf("loadFile() = %+v, %v", c, err)
	}
	if err := saveFile(path, c); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"version": 1`) {
		t.Fatalf("saved config has no version: %s", data)
	}
}

func TestANewerConfigIsRefusedAndNeverRewritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{"version":99,"registry":"acme/skills","future":true}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var tooNew *TooNewError
	if _, err := loadFile(path); !errors.As(err, &tooNew) || tooNew.Found != 99 {
		t.Fatalf("loadFile() error = %v", err)
	}
	if err := saveFile(path, Config{Registry: "other/repo"}); !errors.As(err, &tooNew) {
		t.Fatalf("saveFile() error = %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != body {
		t.Fatalf("config was changed: %s", got)
	}
}
