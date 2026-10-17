package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCompletion(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := Root()
	cmd.SetArgs(append([]string{"completion"}, args...))
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	return out.String(), err
}

func fakeHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestCompletionInstallZshIsIdempotent(t *testing.T) {
	home := fakeHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("export A=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runCompletion(t, "zsh", "--install", "--yes"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(rc)
	if want := "export A=1\n" + completionMarker + "\n" + guardedSource("zsh") + "\n"; string(got) != want {
		t.Fatalf("zshrc = %q, want %q", got, want)
	}
	out, err := runCompletion(t, "zsh", "--install", "--yes")
	if err != nil || !strings.Contains(out, "already") {
		t.Fatalf("second run: %q, %v", out, err)
	}
	if again, _ := os.ReadFile(rc); !bytes.Equal(again, got) {
		t.Fatal("second install changed the file")
	}
}

func TestCompletionInstallFish(t *testing.T) {
	home := fakeHome(t)
	if _, err := runCompletion(t, "fish", "--install", "--yes"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(home, ".config", "fish", "completions", "hikma.fish"))
	if err != nil || !strings.Contains(string(b), "hikma") {
		t.Fatalf("fish file missing or empty: %v", err)
	}
}

func TestCompletionInstallNeedsConfirmationWhenNotInteractive(t *testing.T) {
	home := fakeHome(t)
	_, err := runCompletion(t, "zsh", "--install")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("want confirmation error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".zshrc")); statErr == nil {
		t.Fatal("wrote without confirmation")
	}
}

func TestHasActiveLineIgnoresComments(t *testing.T) {
	line := guardedSource("zsh")
	if hasActiveLine("# "+line+"\n", line) {
		t.Error("commented-out line counted as installed")
	}
	if !hasActiveLine("a\n  "+line+"  \nb", line) {
		t.Error("active line not found")
	}
}
