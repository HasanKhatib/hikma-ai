package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	tm.waitFor("Setup and settings", menuWait)
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

func TestMenuClearsAndComesBackAfterACommand(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	tm := startTerm(t, sb.project, sb.env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.choose("Setup and settings >", menuWait)
	tm.waitFor("Check your setup", menuWait)
	tm.settle()
	tm.choose("Check your setup", menuWait)
	tm.waitFor("What next?", menuWait)
	tm.settle()
	tm.choose("Back to the menu", menuWait)
	tm.waitFor("Install skills from your registry", menuWait)
	tm.settle()
	if n := strings.Count(tm.output(), "\x1b[2J"); n < 2 {
		t.Errorf("the screen was cleared %d times, want it cleared on each menu draw", n)
	}
	if strings.Contains(tm.output(), "\x1b[3J") {
		t.Error("the menu must not erase the user's scrollback (ESC[3J)")
	}
	tm.choose("Quit", menuWait)
	if exited, err := tm.waitDone(10 * time.Second); !exited || err != nil {
		t.Fatalf("Quit did not exit cleanly (%v); screen:\n%s", err, plain(tm.output()))
	}
}

// registryRepo is a local git repository with two skills, used as the registry.
func registryRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "--quiet", "-b", "main")
	for _, n := range []string{"alpha", "beta"} {
		body := "---\nname: " + n + "\ndescription: the " + n + " skill\n---\n# " + n + "\n"
		if err := os.MkdirAll(filepath.Join(dir, "skills", n), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "skills", n, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "--quiet", "-m", "init")
	return dir
}

func TestMenuInstallsASkillEndToEnd(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	reg := registryRepo(t)
	tm := startTerm(t, sb.project, sb.with("HIKMA_REGISTRY="+reg, "HIKMA_AGENT=claude"))
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.choose("Install skills from your registry", menuWait)
	tm.waitFor("Skills to install:", menuWait)
	tm.settle()
	tm.tick("✓ alpha") // tick the first skill, and see it ticked before confirming
	tm.press(enter)
	tm.waitFor("Will install 1 skill", menuWait)
	tm.settle()
	tm.press(enter) // the plan's question defaults to yes
	tm.waitFor("Equivalent command", menuWait)
	tm.waitFor("What next?", menuWait)
	if _, err := os.Stat(filepath.Join(sb.project, ".claude", "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatalf("the skill was not installed: %v\nscreen:\n%s", err, plain(tm.output()))
	}
	if !strings.Contains(plain(tm.output()), "hikma skill install alpha") {
		t.Errorf("the equivalent command should be complete:\n%s", plain(tm.output()))
	}
	tm.settle()
	tm.choose("Quit", menuWait)
	if exited, err := tm.waitDone(10 * time.Second); !exited || err != nil {
		t.Fatalf("Quit did not exit cleanly (%v)", err)
	}
}

func TestEscInsideATaskComesBackToTheMenu(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	reg := registryRepo(t)
	tm := startTerm(t, sb.project, sb.with("HIKMA_REGISTRY="+reg, "HIKMA_AGENT=claude"))
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.choose("Install skills from your registry", menuWait)
	tm.waitFor("Skills to install:", menuWait)
	tm.settle()
	tm.press("\x1b") // Esc backs out of the picker without installing
	deadline := time.Now().Add(menuWait)
	for strings.Count(plain(tm.output()), "What do you want to do?") < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("Esc did not return to the menu; screen:\n%s", plain(tm.output()))
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(sb.project, ".claude", "skills", "alpha")); err == nil {
		t.Error("Esc must not install anything")
	}
	tm.settle()
	tm.choose("Quit", menuWait)
	if exited, err := tm.waitDone(10 * time.Second); !exited || err != nil {
		t.Fatalf("Quit did not exit cleanly (%v)", err)
	}
}

// clearsOnStart counts the clear-screen sequences on the screen once the menu is
// up. Some terminal layers (Windows ConPTY) paint a clear of their own, so a test
// compares runs instead of expecting zero everywhere.
func clearsOnStart(t *testing.T, env []string, dir string) int {
	t.Helper()
	tm := startTerm(t, dir, env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	n := strings.Count(tm.output(), "\x1b[2J")
	tm.write(ctrlC)
	tm.waitDone(10 * time.Second)
	return n
}

func TestAccessibleModeUsesNumberedPromptsAndNeverClears(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	normal := clearsOnStart(t, sb.env, sb.project)
	accessible := clearsOnStart(t, sb.with("ACCESSIBLE=1"), sb.project)
	if accessible >= normal {
		t.Errorf("accessible mode cleared the screen %d times, the normal menu %d: it should clear less", accessible, normal)
	}
	if runtime.GOOS != "windows" && accessible != 0 {
		t.Errorf("accessible mode cleared the screen %d times, want none", accessible)
	}

	tm := startTerm(t, sb.project, sb.with("ACCESSIBLE=1"))
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.write("8" + enter) // the last entry, Quit
	if exited, err := tm.waitDone(10 * time.Second); !exited || err != nil {
		t.Fatalf("Quit by number did not exit cleanly (%v); screen:\n%s", err, plain(tm.output()))
	}
}
