package commands

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

type upgradeRig struct {
	ran       []string
	fetched   []string
	installed string // what the check after the upgrade reports
}

// rigUpgrade stands in for the machine: the running binary, the latest release
// GitHub reports, what ends up installed, and every process hikma would start.
func rigUpgrade(t *testing.T, exe, current, latest string) *upgradeRig {
	t.Helper()
	r := &upgradeRig{installed: latest}
	oe, of, ox, oi, ov := upgradeExe, upgradeFetch, upgradeExec, upgradeInstalled, version
	upgradeExe = func() (string, error) { return exe, nil }
	upgradeFetch = func(url string) ([]byte, error) {
		r.fetched = append(r.fetched, url)
		if url == latestReleaseURL {
			if latest == "" {
				return nil, errors.New("rate limited")
			}
			return []byte(`{"tag_name":"v` + latest + `"}`), nil
		}
		return []byte("#!/bin/sh\n"), nil
	}
	upgradeExec = func(c *exec.Cmd) error { r.ran = append(r.ran, strings.Join(c.Args, " ")); return nil }
	upgradeInstalled = func(bool, string) (string, error) { return r.installed, nil }
	version = current
	t.Cleanup(func() { upgradeExe, upgradeFetch, upgradeExec, upgradeInstalled, version = oe, of, ox, oi, ov })
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

// noQuestions fails the test if anything is asked.
type noQuestions struct{ *fakePrompter }

func (noQuestions) Confirm(string, bool) (bool, error) { panic("upgrade must not ask") }

const brewExe = "/opt/homebrew/Cellar/hikma/1.2.0/bin/hikma"

func TestUpgradeShowsBothVersionsAndUpgradesWithoutAsking(t *testing.T) {
	r := rigUpgrade(t, brewExe, "1.2.0", "1.3.0")
	withPrompter(t, noQuestions{&fakePrompter{}})
	out, err := runUpgradeCmd(t)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Current version: 1.2.0", "Latest release:  1.3.0", "1.2.0 -> 1.3.0", "Done."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if want := []string{"brew update --quiet", "brew upgrade " + brewFormula}; strings.Join(r.ran, "|") != strings.Join(want, "|") {
		t.Errorf("ran %v, want %v", r.ran, want)
	}
}

func TestUpgradeWithoutATerminalDoesNotWaitOrRefuse(t *testing.T) {
	r := rigUpgrade(t, brewExe, "1.2.0", "1.3.0")
	oldT := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = oldT })
	if _, err := runUpgradeCmd(t); err != nil {
		t.Fatalf("a script running hikma upgrade should upgrade: %v", err)
	}
	if len(r.ran) != 2 {
		t.Errorf("ran %v", r.ran)
	}
}

func TestUpgradeYesIsStillAcceptedAndDoesNothingExtra(t *testing.T) {
	r := rigUpgrade(t, brewExe, "1.2.0", "1.3.0")
	if _, err := runUpgradeCmd(t, "--yes"); err != nil {
		t.Fatalf("scripts that pass --yes must keep working: %v", err)
	}
	if len(r.ran) != 2 {
		t.Errorf("ran %v", r.ran)
	}
}

func TestUpgradeWhenAlreadyCurrentChangesNothing(t *testing.T) {
	for _, c := range []struct{ current, latest, want string }{
		{"1.3.0", "1.3.0", "is up to date"},
		{"1.4.0", "1.3.0", "newer than the latest release"},
	} {
		r := rigUpgrade(t, brewExe, c.current, c.latest)
		out, err := runUpgradeCmd(t)
		if err != nil || !strings.Contains(out, c.want) || len(r.ran) != 0 {
			t.Errorf("%s vs %s: out=%q err=%v ran=%v", c.current, c.latest, out, err, r.ran)
		}
	}
}

func TestUpgradeDryRunShowsThePlanAndChangesNothing(t *testing.T) {
	r := rigUpgrade(t, brewExe, "1.2.0", "1.3.0")
	out, err := runUpgradeCmd(t, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Would upgrade 1.2.0 -> 1.3.0 with Homebrew") || !strings.Contains(out, "Nothing was changed") {
		t.Errorf("plan missing:\n%s", out)
	}
	if len(r.ran) != 0 || len(r.fetched) != 1 {
		t.Errorf("dry run must only ask for the latest release: ran=%v fetched=%v", r.ran, r.fetched)
	}
}

func TestUpgradeSaysWhenHomebrewHasNotCaughtUpWithTheRelease(t *testing.T) {
	r := rigUpgrade(t, brewExe, "1.2.0", "1.3.0")
	r.installed = "1.2.0" // brew ran, but the tap still has the old formula
	out, err := runUpgradeCmd(t)
	if err == nil || !strings.Contains(err.Error(), "Homebrew still provides 1.2.0") || !strings.Contains(err.Error(), "try again") {
		t.Fatalf("want the stale tap named, got %v", err)
	}
	if strings.Contains(out, "Done.") {
		t.Errorf("must not claim success:\n%s", out)
	}
}

func TestUpgradeScriptInstallRunsScriptIntoTheBinaryFolder(t *testing.T) {
	r := rigUpgrade(t, "/home/me/.local/bin/hikma", "1.2.0", "1.3.0")
	out, err := runUpgradeCmd(t)
	if err != nil {
		t.Fatal(err)
	}
	scripts := 0
	for _, u := range r.fetched {
		if u == installScriptURL {
			scripts++
		}
	}
	if scripts != 1 || len(r.ran) != 1 || !strings.HasPrefix(r.ran[0], "bash ") {
		t.Fatalf("ran=%v fetched=%v", r.ran, r.fetched)
	}
	if !strings.Contains(out, "1.2.0 -> 1.3.0") {
		t.Errorf("missing the change:\n%s", out)
	}
}

func TestUpgradeCannotReachGitHubSaysWhatToRunInstead(t *testing.T) {
	r := rigUpgrade(t, brewExe, "1.2.0", "")
	_, err := runUpgradeCmd(t)
	if err == nil || !strings.Contains(err.Error(), "check the latest release") || !strings.Contains(err.Error(), "brew upgrade "+brewFormula) {
		t.Fatalf("got %v", err)
	}
	if len(r.ran) != 0 {
		t.Errorf("nothing should run when the latest release is unknown: %v", r.ran)
	}
}

func TestUpgradeFromADevelopmentBuildProceeds(t *testing.T) {
	r := rigUpgrade(t, "/home/me/go/bin/hikma", "dev", "1.3.0")
	if _, err := runUpgradeCmd(t); err != nil {
		t.Fatal(err)
	}
	if len(r.ran) != 1 {
		t.Errorf("a dev build has no version to compare, so it upgrades: ran=%v", r.ran)
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		{"1.2.0", "1.3.0", -1, true},
		{"v1.10.0", "1.9.9", 1, true},
		{"1.3.0", "v1.3.0", 0, true},
		{"1.3.0-rc.1", "1.3.0", 0, true},
		{"dev", "1.3.0", 0, false},
		{"1.3", "1.3.0", 0, false},
	} {
		got, ok := compareVersions(c.a, c.b)
		if got != c.want || ok != c.ok {
			t.Errorf("compareVersions(%q, %q) = %d, %v; want %d, %v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
	if got := newestVersion([]string{"hikma", "1.2.0", "1.10.0", "1.9.0"}); got != "1.10.0" {
		t.Errorf("newestVersion = %q", got)
	}
}
