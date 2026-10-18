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

func TestMenuPublishPicksALocalSkillAndPrintsTheCommand(t *testing.T) {
	skipShort(t)
	sb := newSandbox(t)
	sb.writeSkill(t, "alpha")
	tm := startTerm(t, sb.project, sb.env)
	tm.waitFor("What do you want to do?", menuWait)
	tm.settle()
	tm.choose("Publish a skill to your registry", menuWait)
	tm.waitFor("Skill to publish:", menuWait)
	tm.waitFor("alpha", menuWait)
	tm.settle()
	tm.press(enter) // the picker has one skill, already selected
	tm.waitFor("Registry to publish to", menuWait)
	tm.settle()
	tm.typeText("acme/skills", menuWait)
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
	tm.choose("Set your registry", menuWait)
	tm.waitFor("Registry (owner/repo):", menuWait)
	tm.settle()
	tm.typeText("acme/skills", menuWait)
	tm.press(enter)
	tm.waitFor("hikma config registry acme/skills", menuWait)
	if exited, err := tm.waitDone(30 * time.Second); !exited || err != nil {
		t.Fatalf("exited=%v err=%v; screen:\n%s", exited, err, plain(tm.output()))
	}
}
