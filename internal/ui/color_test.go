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
