package e2e

import (
	"strings"
	"testing"
	"time"
)

const menuWait = 20 * time.Second

func repeat(k string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = k
	}
	return out
}

func downs5() []string  { return repeat(down, 5) }
func downs8() []string  { return repeat(down, 8) }
func downs11() []string { return repeat(down, 11) }

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
	tm.press(downs11()...) // the twelfth entry is Quit
	tm.press(enter)
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

func TestMenuPublishPicksALocalSkillAndPrintsTheCommand(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	sb.writeSkill(t, "alpha")
	tm := startTerm(t, sb.project, sb.env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.press(downs5()...) // Publish a skill to your registry
	tm.press(enter)
	tm.waitFor("Skill to publish:", menuWait)
	tm.waitFor("alpha", menuWait)
	tm.settle()
	tm.press(enter)
	tm.waitFor("Registry to publish to", menuWait)
	tm.settle()
	tm.write("acme/skills")
	tm.press(enter)
	tm.waitFor("hikma skill push alpha --registry acme/skills", menuWait)
	if exited, _ := tm.waitDone(30 * time.Second); !exited {
		t.Fatalf("hikma did not exit after the command; screen:\n%s", plain(tm.output()))
	}
}

func TestMenuSetRegistryWritesConfig(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	tm := startTerm(t, sb.project, sb.env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.press(downs8()...) // Set your registry
	tm.press(enter)
	tm.waitFor("Registry (owner/repo):", menuWait)
	tm.settle()
	tm.write("acme/skills")
	tm.press(enter)
	tm.waitFor("hikma config registry acme/skills", menuWait)
	if exited, err := tm.waitDone(30 * time.Second); !exited || err != nil {
		t.Fatalf("exited=%v err=%v; screen:\n%s", exited, err, plain(tm.output()))
	}
}
