package commands

import (
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/ui"
)

func TestGlobalFlagsAreParsedBeforeAnyCommandRuns(t *testing.T) {
	t.Cleanup(func() { noInput, ui.NoColor = false, false })
	if _, err := run(t, "--no-input", "--no-color", "completion", "bash"); err != nil {
		t.Fatal(err)
	}
	if !noInput || !ui.NoColor {
		t.Fatalf("noInput=%v NoColor=%v, want both true", noInput, ui.NoColor)
	}
}

func TestNoInputMakesEveryPromptSiteFailWithTheFlagToPass(t *testing.T) {
	t.Cleanup(func() { noInput = false })
	noInput = true
	if isInteractive() {
		t.Fatal("isInteractive must be false under --no-input, even on a terminal")
	}
	fakeHome(t)
	_, err := runCompletion(t, "--no-input", "zsh", "--install")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("want an error naming --yes, got %v", err)
	}
}

func TestNoInputBareHikmaPrintsHelp(t *testing.T) {
	t.Cleanup(func() { noInput = false })
	out, err := run(t, "--no-input")
	if err != nil || !strings.Contains(out, "Usage:") {
		t.Fatalf("want help, got %q (%v)", out, err)
	}
}
