package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestDoctorColorsOnATerminalAndNotUnderNoColor(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)

	colored := startTerm(t, sb.project, sb.with("TERM=xterm-256color"), "doctor")
	if exited, _ := colored.waitDone(30 * time.Second); !exited {
		t.Fatalf("doctor did not finish; screen:\n%s", plain(colored.output()))
	}
	if !strings.Contains(colored.output(), "\x1b[32m") {
		t.Errorf("expected green ok on a terminal, got %q", colored.output())
	}

	nocolor := startTerm(t, sb.project, sb.with("TERM=xterm-256color", "NO_COLOR=1"), "doctor")
	if exited, _ := nocolor.waitDone(30 * time.Second); !exited {
		t.Fatalf("doctor did not finish under NO_COLOR; screen:\n%s", plain(nocolor.output()))
	}
	if strings.Contains(nocolor.output(), "\x1b[") {
		t.Errorf("NO_COLOR output has escape codes: %q", nocolor.output())
	}
}

func TestBareHikmaPrintsHelpWhenNotATerminal(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	out, err := runPlain(t, sb, "--help")
	if err != nil || !strings.Contains(out, "Usage:") {
		t.Fatalf("err=%v out=%s", err, out)
	}
}
