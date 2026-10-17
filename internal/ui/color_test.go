package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNoColorForCapturedOutput(t *testing.T) {
	var b bytes.Buffer
	if got := Yellow(&b, "warn"); got != "warn" {
		t.Fatalf("captured output must stay plain, got %q", got)
	}
	PrintWarning(&b, "warning: x")
	if strings.Contains(b.String(), "\x1b") || b.String() != "warning: x\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBannerIsPlainOffATerminal(t *testing.T) {
	var buf bytes.Buffer
	got := Banner(&buf)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("banner has escape codes for a non-terminal writer: %q", got)
	}
	for _, want := range []string{">_", "hikma", "skills follow you"} {
		if !strings.Contains(got, want) {
			t.Errorf("banner missing %q:\n%s", want, got)
		}
	}
}

func TestColorHelpersArePlainOffATerminal(t *testing.T) {
	var buf bytes.Buffer
	for _, s := range []string{Green(&buf, "a"), Cyan(&buf, "a"), Bold(&buf, "a"), Dim(&buf, "a")} {
		if s != "a" {
			t.Errorf("got %q, want plain text", s)
		}
	}
}
