package commands

import "testing"

func TestHuhIsOptIn(t *testing.T) {
	t.Setenv("HIKMA_PROMPTS", "")
	if _, ok := defaultPrompter().(surveyPrompter); !ok {
		t.Fatal("survey must stay the default until the Windows check passes")
	}
	t.Setenv("HIKMA_PROMPTS", "HUH")
	if _, ok := defaultPrompter().(huhPrompter); !ok {
		t.Fatal("HIKMA_PROMPTS=huh must select the huh prompter")
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
