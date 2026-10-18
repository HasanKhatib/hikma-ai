package commands

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestGhSkillInterop is a canary, not a unit test: it runs the real `gh skill
// install` and checks hikma still reads what it writes. gh skill is a preview
// and may change without notice, so the Canary workflow runs this weekly against
// the latest gh. It needs the network and a GH_TOKEN, so it only runs when
// HIKMA_GH_CANARY=1.
func TestGhSkillInterop(t *testing.T) {
	if os.Getenv("HIKMA_GH_CANARY") != "1" {
		t.Skip("set HIKMA_GH_CANARY=1 to run the gh skill canary (needs the network and GH_TOKEN)")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		t.Fatal("gh is not on PATH")
	}
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		t.Fatal("GH_TOKEN is not set")
	}
	proj := project(t)
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "canary@example.com"}, {"config", "user.name", "canary"}} {
		if out, err := exec.Command("git", append([]string{"-C", proj}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if v, err := exec.Command("gh", "--version").Output(); err == nil {
		t.Logf("gh: %s", strings.SplitN(string(v), "\n", 2)[0])
	}

	// A public skill; the canary tests gh's metadata format, not this repository.
	install := exec.Command("gh", "skill", "install", "anthropics/skills", "skill-creator", "--agent", "claude-code", "--scope", "project")
	install.Dir = proj
	install.Env = append(os.Environ(), "GH_TOKEN="+token, "GH_PROMPT_DISABLED=1")
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("gh skill install failed (did gh skill change?): %v\n%s", err, out)
	}

	out, err := run(t, "skill", "list", "--installed")
	if err != nil {
		t.Fatalf("list --installed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "installed by gh skill") || !strings.Contains(out, "hikma skill adopt skill-creator") {
		t.Fatalf("hikma no longer recognizes a gh skill install (metadata format changed?):\n%s", out)
	}
	if out, err := run(t, "skill", "adopt", "skill-creator", "--agent", "claude"); err != nil {
		t.Fatalf("adopt: %v\n%s", err, out)
	}
	if out, err := run(t, "skill", "verify"); err != nil {
		t.Fatalf("verify after adopt: %v\n%s", err, out)
	}
}
