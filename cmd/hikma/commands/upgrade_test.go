package commands

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

type upgradeRig struct {
	ran     []string
	fetched int
}

func rigUpgrade(t *testing.T, exe string) *upgradeRig {
	t.Helper()
	r := &upgradeRig{}
	oe, of, ox := upgradeExe, upgradeFetch, upgradeExec
	upgradeExe = func() (string, error) { return exe, nil }
	upgradeFetch = func(string) ([]byte, error) { r.fetched++; return []byte("#!/bin/sh\n"), nil }
	upgradeExec = func(c *exec.Cmd) error { r.ran = append(r.ran, strings.Join(c.Args, " ")); return nil }
	t.Cleanup(func() { upgradeExe, upgradeFetch, upgradeExec = oe, of, ox })
	return r
}

func runUpgradeCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := Root()
	cmd.SetArgs(append([]string{"upgrade"}, args...))
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	return out.String(), err
}

func TestUpgradeWithoutTerminalOrYesChangesNothing(t *testing.T) {
	r := rigUpgrade(t, "/usr/local/bin/hikma")
	out, err := runUpgradeCmd(t)
	if err == nil || !strings.Contains(err.Error(), "hikma upgrade --yes") {
		t.Fatalf("want an error naming --yes, got %v", err)
	}
	if !strings.Contains(out, "This will replace") || r.fetched != 0 || len(r.ran) != 0 {
		t.Fatalf("must only print the plan: out=%q rig=%+v", out, r)
	}
}

func TestUpgradeDeclinedChangesNothing(t *testing.T) {
	r := rigUpgrade(t, "/opt/homebrew/Cellar/hikma/1.0.0/bin/hikma")
	withPrompter(t, &fakePrompter{confirms: []bool{false}})
	out, err := runUpgradeCmd(t)
	if err != nil || !strings.Contains(out, "Not upgrading") || len(r.ran) != 0 {
		t.Fatalf("out=%q err=%v ran=%v", out, err, r.ran)
	}
}

func TestUpgradeHomebrewRunsBrew(t *testing.T) {
	r := rigUpgrade(t, "/opt/homebrew/Cellar/hikma/1.0.0/bin/hikma")
	if _, err := runUpgradeCmd(t, "--yes"); err != nil {
		t.Fatal(err)
	}
	if len(r.ran) != 1 || r.ran[0] != "brew upgrade "+brewFormula || r.fetched != 0 {
		t.Fatalf("ran=%v fetched=%d", r.ran, r.fetched)
	}
}

func TestUpgradeScriptInstallRunsScriptIntoTheBinaryFolder(t *testing.T) {
	r := rigUpgrade(t, "/home/me/.local/bin/hikma")
	if _, err := runUpgradeCmd(t, "--yes"); err != nil {
		t.Fatal(err)
	}
	if r.fetched != 1 || len(r.ran) != 1 || !strings.HasPrefix(r.ran[0], "bash ") {
		t.Fatalf("ran=%v fetched=%d", r.ran, r.fetched)
	}
}
