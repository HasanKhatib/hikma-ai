package e2e

import (
	"strings"
	"testing"
	"time"
)

const menuWait = 20 * time.Second

func TestMenuDrawsWithoutTerminalReplies(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	tm := startTerm(t, sb.project, sb.env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.waitFor("Check your setup", menuWait)
	tm.settle()
	tm.write(ctrlC)
	if exited, _ := tm.waitDone(10 * time.Second); !exited {
		t.Fatalf("Ctrl-C did not leave the menu; screen:\n%s", plain(tm.output()))
	}
}

func TestMenuQuitExitsCleanly(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	tm := startTerm(t, sb.project, sb.env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.choose("Quit", menuWait)
	exited, err := tm.waitDone(10 * time.Second)
	if !exited {
		t.Fatalf("Quit did not exit; screen:\n%s", plain(tm.output()))
	}
	if err != nil {
		t.Fatalf("Quit exited with %v", err)
	}
	if strings.Contains(plain(tm.output()), "Equivalent command") {
		t.Error("Quit must not print an equivalent command")
	}
}
