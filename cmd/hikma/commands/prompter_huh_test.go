package commands

import (
	"os"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/ui"
)

func TestHuhIsTheDefaultPrompter(t *testing.T) {
	if _, ok := prompter.(huhPrompter); !ok {
		t.Fatalf("default prompter is %T, want huhPrompter", prompter)
	}
}

func TestAccessibleModeForDumbTerminals(t *testing.T) {
	t.Setenv("ACCESSIBLE", "")
	t.Setenv("TERM", "xterm-256color")
	if accessible() {
		t.Fatal("a normal terminal should get the full widgets")
	}
	t.Setenv("TERM", "dumb")
	if !accessible() {
		t.Fatal("TERM=dumb should get plain line prompts")
	}
	t.Setenv("TERM", "xterm")
	t.Setenv("ACCESSIBLE", "1")
	if !accessible() {
		t.Fatal("ACCESSIBLE should force plain line prompts")
	}
}

func TestNoColorFlagReachesHuhThroughNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Cleanup(func() { ui.NoColor = false })
	applyNoColor()
	if os.Getenv("NO_COLOR") != "" {
		t.Fatal("NO_COLOR must stay untouched without --no-color")
	}
	ui.NoColor = true
	applyNoColor()
	if os.Getenv("NO_COLOR") != "1" {
		t.Fatal("--no-color must set NO_COLOR so huh drops its colors")
	}
}
