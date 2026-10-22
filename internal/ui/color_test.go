package ui

import (
	"bytes"
	"os"
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

func TestNoColorFlagTurnsColorOff(t *testing.T) {
	t.Cleanup(func() { NoColor = false })
	NoColor = true
	if colorOK(os.Stdout) {
		t.Fatal("colorOK must be false when --no-color is set")
	}
}

func TestClearScreenWritesNothingOffATerminal(t *testing.T) {
	var buf bytes.Buffer
	ClearScreen(&buf)
	if buf.Len() != 0 {
		t.Errorf("captured output got %q", buf.String())
	}
}

func TestClearScreenWritesNothingInAccessibleMode(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")
	var buf bytes.Buffer
	ClearScreen(&buf)
	if buf.Len() != 0 {
		t.Errorf("got %q", buf.String())
	}
}

func TestWidthCountsColumnsNotBytes(t *testing.T) {
	if got := Width("café"); got != 4 {
		t.Errorf("Width(café) = %d, want 4", got)
	}
	if got := Width("日本"); got != 4 {
		t.Errorf("Width(日本) = %d, want 4 columns", got)
	}
	if got := Pad("café", 6); got != "café  " {
		t.Errorf("Pad = %q", got)
	}
}

func TestStrikeIsPlainOffATerminal(t *testing.T) {
	var buf bytes.Buffer
	if got := Strike(&buf, "[installed] alpha"); got != "[installed] alpha" {
		t.Errorf("captured output got %q", got)
	}
}

func TestTruncateCountsColumns(t *testing.T) {
	if got := Truncate("short", 10); got != "short" {
		t.Errorf("a short string changes: %q", got)
	}
	got := Truncate("someone-with-a-very-long-registry-name/and-a-long-repo", 20)
	if Width(got) > 20 || !strings.HasSuffix(got, "...") {
		t.Errorf("Truncate = %q (%d columns)", got, Width(got))
	}
	if got := Truncate("日本語日本語日本語", 9); Width(got) > 9 {
		t.Errorf("wide characters overflow: %q is %d columns", got, Width(got))
	}
}
