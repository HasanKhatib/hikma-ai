package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/lock"
)

// threeSkillRepo is a registry with alpha, beta, and gamma under skills/.
func threeSkillRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git(t, dir, "init", "--quiet", "-b", "main")
	for _, n := range []string{"alpha", "beta", "gamma"} {
		write(t, filepath.Join(dir, "skills", n, "SKILL.md"), strings.NewReplacer("%s", n).Replace(skillMD))
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "--quiet", "-m", "init")
	return dir
}

func installed(proj, name string) bool {
	_, err := os.Stat(filepath.Join(proj, ".claude", "skills", name, "SKILL.md"))
	return err == nil
}

func TestInstallSeveralSkillsOpensTheSourceOnce(t *testing.T) {
	reg := threeSkillRepo(t)
	proj := project(t)

	out, err := run(t, "skill", "install", reg, "--skill", "alpha", "--skill=gamma", "--agent", "claude")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !installed(proj, "alpha") || !installed(proj, "gamma") || installed(proj, "beta") {
		t.Fatalf("wanted alpha and gamma only; output:\n%s", out)
	}
	if n := strings.Count(out, "Source: "); n != 1 {
		t.Errorf("the source was opened %d times, want once:\n%s", n, out)
	}
	if !strings.Contains(out, "Installed 2 of 2") {
		t.Errorf("missing the summary:\n%s", out)
	}
	lf, err := lock.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"alpha", "gamma"} {
		if _, ok := lf.Skills[".claude/skills/"+n]; !ok {
			t.Errorf("no lock entry for %s", n)
		}
	}
}

func TestInstallSkillFlagUsesTheConfiguredRegistry(t *testing.T) {
	reg := threeSkillRepo(t)
	proj := project(t)
	t.Setenv("HIKMA_REGISTRY", reg)
	if out, err := run(t, "skill", "install", "--skill", "beta", "--skill", "alpha", "--agent", "claude"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !installed(proj, "alpha") || !installed(proj, "beta") {
		t.Fatal("both skills should be installed")
	}
}

func TestInstallUnknownSkillWritesNothing(t *testing.T) {
	reg := threeSkillRepo(t)
	proj := project(t)
	out, err := run(t, "skill", "install", reg, "--skill", "alpha", "--skill", "nope", "--agent", "claude")
	if err == nil || !strings.Contains(err.Error(), "nope") || !strings.Contains(err.Error(), "nothing was installed") {
		t.Fatalf("got %v\n%s", err, out)
	}
	if installed(proj, "alpha") {
		t.Error("a typo must not install the other skills")
	}
}

func TestInstallSeveralReportsWhichFailed(t *testing.T) {
	reg := threeSkillRepo(t)
	proj := project(t)
	if out, err := run(t, "skill", "install", reg, "beta", "--agent", "claude"); err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	out, err := run(t, "skill", "install", reg, "--skill", "alpha", "--skill", "beta", "--skill", "gamma", "--agent", "claude")
	var f *installFailed
	if !errors.As(err, &f) || len(f.Failed) != 1 || f.Failed[0] != "beta" {
		t.Fatalf("got %v, want beta reported as failed\n%s", err, out)
	}
	if !installed(proj, "alpha") || !installed(proj, "gamma") {
		t.Error("the skills that could install should be installed")
	}
	if !strings.Contains(out, "Installed 2 of 3") || !strings.Contains(out, "failed  beta") || !strings.Contains(out, "--force") {
		t.Errorf("summary should name beta and say how to reinstall:\n%s", out)
	}
}

func TestInstallRejectsANameAndSkillFlagTogether(t *testing.T) {
	reg := threeSkillRepo(t)
	project(t)
	for _, args := range [][]string{
		{"skill", "install", reg, "alpha", "--skill", "beta"},
		{"skill", "install", "alpha", "--skill", "beta"},
	} {
		if _, err := run(t, append(args, "--agent", "claude")...); err == nil || !strings.Contains(err.Error(), "not both") {
			t.Errorf("%v: got %v, want a usage error", args, err)
		}
	}
}

func TestInstallWithoutNameOrSkillStillNeedsAnArgument(t *testing.T) {
	project(t)
	if _, err := run(t, "skill", "install"); err == nil || !strings.Contains(err.Error(), "missing argument") {
		t.Fatalf("got %v", err)
	}
}

func TestRenderResultNamesEveryState(t *testing.T) {
	got := renderResult([]skillResult{{Name: "a"}, {Name: "bb", Err: errors.New("boom\nsecond line")}})
	want := "Installed 1 of 2\n  ok      a\n  failed  bb  boom; second line\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestInstallStopsBetweenSkillsWhenCancelled(t *testing.T) {
	reg := threeSkillRepo(t)
	proj := project(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Ctrl-C had already been taken before the first skill
	root := Root()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"skill", "install", reg, "--skill", "alpha", "--skill", "beta", "--agent", "claude"})
	err := root.ExecuteContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want the cancel", err)
	}
	if installed(proj, "alpha") || installed(proj, "beta") {
		t.Error("nothing should be installed after a cancel")
	}
	if !strings.Contains(out.String(), "not started: cancelled") {
		t.Errorf("the summary should say what did not run:\n%s", out.String())
	}
}

func TestInstallFailedErrorDoesNotRepeatTheSummary(t *testing.T) {
	e := &installFailed{Failed: []string{"beta"}, Total: 3}
	if e.Error() != "1 of 3 skills were not installed" {
		t.Errorf("got %q", e.Error())
	}
}

func TestListMarksInstalledSkillsInTheTableOnly(t *testing.T) {
	reg := threeSkillRepo(t)
	project(t)
	t.Setenv("HIKMA_AGENT", "claude")
	if out, err := run(t, "skill", "install", reg, "beta"); err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	out, err := run(t, "skill", "list", reg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[installed] beta") {
		t.Errorf("beta is installed and should be marked:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "alpha") && strings.Contains(line, "[installed]") {
			t.Errorf("alpha is not installed but is marked: %q", line)
		}
	}
	// Machine output is unchanged.
	js, err := run(t, "skill", "list", reg, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(js, "[installed]") {
		t.Errorf("--json must not carry display markers:\n%s", js)
	}
}

func TestListHasNoMarkerColumnWhenNothingIsInstalled(t *testing.T) {
	reg := threeSkillRepo(t)
	project(t)
	t.Setenv("HIKMA_AGENT", "claude")
	out, err := run(t, "skill", "list", reg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[installed]") {
		t.Errorf("nothing is installed:\n%s", out)
	}
	if !strings.Contains(out, "NAME ") || strings.Contains(out, "  NAME") {
		t.Errorf("the table should not be indented when nothing is marked:\n%s", out)
	}
}

func TestInstallPickerMarksInstalledSkills(t *testing.T) {
	reg := threeSkillRepo(t)
	project(t)
	t.Setenv("HIKMA_AGENT", "claude")
	if out, err := run(t, "skill", "install", reg, "gamma"); err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	f := &fakePrompter{pick: 0}
	withPrompter(t, f)
	if out, err := run(t, "skill", "install", reg); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	var marked []string
	for _, o := range f.options {
		if strings.Contains(o, "[installed]") {
			marked = append(marked, o)
		}
	}
	if len(marked) != 1 || !strings.Contains(marked[0], "gamma") {
		t.Errorf("only gamma is installed; picker options: %q", f.options)
	}
}
