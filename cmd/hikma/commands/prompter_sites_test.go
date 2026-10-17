package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withPrompter pretends to be on a terminal and routes every prompt to f.
func withPrompter(t *testing.T, f Prompter) {
	t.Helper()
	oldP, oldT := prompter, stdinIsTerminal
	prompter, stdinIsTerminal = f, func() bool { return true }
	t.Cleanup(func() { prompter, stdinIsTerminal = oldP, oldT })
}

func TestConfirmationsDefaultToNoAndDeclineStopsTheWrite(t *testing.T) {
	f := &fakePrompter{confirms: []bool{false}}
	withPrompter(t, f)
	err := askGlobal(nil)
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("want a cancelled error, got %v", err)
	}
	if f.gotConf {
		t.Fatal("the home-directory confirmation must default to No")
	}
}

func TestCompletionInstallDeclinedWritesNothing(t *testing.T) {
	home := fakeHome(t)
	f := &fakePrompter{confirms: []bool{false}}
	withPrompter(t, f)
	out, err := runCompletion(t, "zsh", "--install")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Skipped") {
		t.Fatalf("want a skip message, got %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".zshrc")); statErr == nil {
		t.Fatal("wrote after the user said no")
	}
}

func TestCompletionInstallAcceptedThroughThePrompter(t *testing.T) {
	home := fakeHome(t)
	withPrompter(t, &fakePrompter{confirms: []bool{true}})
	if _, err := runCompletion(t, "zsh", "--install"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); err != nil {
		t.Fatalf("expected .zshrc: %v", err)
	}
}

func TestCancellingAPromptSurfacesAsCancelled(t *testing.T) {
	fakeHome(t)
	withPrompter(t, &fakePrompter{err: errCancelled})
	_, err := runCompletion(t, "zsh", "--install")
	if !errors.Is(err, errCancelled) {
		t.Fatalf("want errCancelled, got %v", err)
	}
}

func TestNoInputBeatsATerminal(t *testing.T) {
	f := &fakePrompter{}
	withPrompter(t, f)
	t.Cleanup(func() { noInput = false })
	fakeHome(t)
	_, err := runCompletion(t, "--no-input", "zsh", "--install")
	if err == nil || !strings.Contains(err.Error(), "--yes") || len(f.asked) != 0 {
		t.Fatalf("err=%v asked=%v: --no-input must fail without asking", err, f.asked)
	}
}
