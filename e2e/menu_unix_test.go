//go:build !windows

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// slowGit is a git on PATH whose clone hangs, so a test can press Ctrl-C while a
// registry is being read. Anything else is passed to the real git.
func slowGit(t *testing.T) (pathDir, pidFile string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	pidFile = filepath.Join(dir, "clone.pid")
	script := "#!/bin/sh\n" +
		"case \" $* \" in *\" clone \"*) echo $$ > '" + pidFile + "'; exec sleep 60;; esac\n" +
		"exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, pidFile
}

func TestCtrlCDuringAClonePutsTheMenuBackAndStopsGit(t *testing.T) {
	skipShort(t)
	sb := newMenuSandbox(t)
	shim, pidFile := slowGit(t)
	env := sb.with("HIKMA_REGISTRY=acme/skills", "HIKMA_AGENT=claude", "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	tm := startTerm(t, sb.project, env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.choose("Install skills from your registry", menuWait)
	tm.waitFor("Reading acme/skills", menuWait)

	var pid int
	deadline := time.Now().Add(menuWait)
	for pid == 0 {
		if b, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
		if pid == 0 && time.Now().After(deadline) {
			t.Fatalf("the clone never started; screen:\n%s", plain(tm.output()))
		}
		time.Sleep(50 * time.Millisecond)
	}

	tm.press(ctrlC)
	for strings.Count(plain(tm.output()), "What do you want to do?") < 2 {
		if time.Now().After(deadline.Add(menuWait)) {
			t.Fatalf("Ctrl-C did not bring the menu back; screen:\n%s", plain(tm.output()))
		}
		time.Sleep(50 * time.Millisecond)
	}
	// The clone itself must be gone, not left running behind the menu.
	stopped := false
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if err := syscall.Kill(pid, 0); err != nil {
			stopped = true
			break
		}
	}
	if !stopped {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Error("git was still running after Ctrl-C")
	}
	tm.settle()
	tm.choose("Quit", menuWait)
	if exited, err := tm.waitDone(10 * time.Second); !exited || err != nil {
		t.Fatalf("Quit did not exit cleanly (%v)", err)
	}
}
