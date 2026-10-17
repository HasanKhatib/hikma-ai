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
	tm.write(downs(11) + enter) // the twelfth entry is Quit
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
	tm.write(downs(5) + enter) // Publish a skill to your registry
	tm.waitFor("Skill to publish:", menuWait)
	tm.waitFor("alpha", menuWait)
	tm.write(enter)
	tm.waitFor("Registry to publish to", menuWait)
	tm.write("acme/skills" + enter)
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
	tm.write(downs(8) + enter) // Set your registry
	tm.waitFor("Registry (owner/repo):", menuWait)
	tm.write("acme/skills" + enter)
	tm.waitFor("hikma config registry acme/skills", menuWait)
	if exited, err := tm.waitDone(30 * time.Second); !exited || err != nil {
		t.Fatalf("exited=%v err=%v; screen:\n%s", exited, err, plain(tm.output()))
	}
}
