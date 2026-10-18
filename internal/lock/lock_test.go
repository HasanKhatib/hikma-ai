package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, body string) string {
	t.Helper()
	p := Path(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadTreatsAMissingVersionAsOne(t *testing.T) {
	root := t.TempDir()
	write(t, root, `{"skills":{}}`)
	f, err := Load(root)
	if err != nil || f.Version != 1 {
		t.Fatalf("Load() = %+v, %v", f, err)
	}
}

func TestLoadRefusesANewerLockfile(t *testing.T) {
	root := t.TempDir()
	write(t, root, `{"version":99,"skills":{},"future":true}`)
	_, err := Load(root)
	var tooNew *TooNewError
	if !errors.As(err, &tooNew) || tooNew.Found != 99 {
		t.Fatalf("Load() error = %v, want TooNewError for 99", err)
	}
}

func TestLoadRefusesANegativeVersion(t *testing.T) {
	root := t.TempDir()
	write(t, root, `{"version":-1,"skills":{}}`)
	if _, err := Load(root); err == nil {
		t.Fatal("expected an error for a negative version")
	}
}

func TestSaveNeverOverwritesANewerLockfile(t *testing.T) {
	root := t.TempDir()
	body := `{"version":99,"skills":{},"future":true}`
	p := write(t, root, body)
	err := Save(root, File{Skills: map[string]Entry{}})
	var tooNew *TooNewError
	if !errors.As(err, &tooNew) {
		t.Fatalf("Save() error = %v, want TooNewError", err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != body {
		t.Fatalf("lockfile was changed: %s", got)
	}
}

func TestSaveWritesTheCurrentVersion(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, File{Skills: map[string]Entry{}}); err != nil {
		t.Fatal(err)
	}
	f, err := Load(root)
	if err != nil || f.Version != Version {
		t.Fatalf("Load() = %+v, %v", f, err)
	}
}
