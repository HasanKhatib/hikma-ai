package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryAccessSaysWhatPublishingWillDo(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(f *fakeGH)
		want  string
	}{
		{"you can write", func(f *fakeGH) {}, "push and import commit straight to main (you can write, and no rule requires a pull request)"},
		{"another default branch name", func(f *fakeGH) { f.branch = "master" }, "commit straight to master"},
		{"a rule requires pull requests", func(f *fakeGH) { f.rules = "deletion,pull_request" }, "main requires pull requests, so push and import open one"},
		{"no write access", func(f *fakeGH) { f.canPush = "false" }, "you cannot write to it, so push and import go through your fork as a pull request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeGH()
			tc.setup(f)
			usePushFake(t, f)
			line, ok := describeRegistry("org/reg")
			if !ok || !strings.Contains(line, tc.want) {
				t.Fatalf("describeRegistry = %q, %v; want it to contain %q", line, ok, tc.want)
			}
		})
	}
}

// Without a signed-in gh GitHub does not say whether you can write, and a
// registry that is not on GitHub has no such question. Neither prints anything.
func TestRegistryAccessStaysSilentWhenItCannotTell(t *testing.T) {
	f := newFakeGH()
	f.canPush = "null"
	usePushFake(t, f)
	if line, ok := describeRegistry("org/reg"); ok {
		t.Errorf("an unknown answer must stay silent, got %q", line)
	}
	old := ghRun
	ghRun = func(string, ...string) (string, error) { return "", fmt.Errorf("gh: command not found") }
	t.Cleanup(func() { ghRun = old })
	if line, ok := describeRegistry("org/reg"); ok {
		t.Errorf("no gh must stay silent, got %q", line)
	}
	noGH(t)
	for _, registry := range []string{"./some/dir", "https://gitlab.com/team/skills.git"} {
		if line, ok := describeRegistry(registry); ok {
			t.Errorf("%s: a registry outside GitHub must stay silent, got %q", registry, line)
		}
	}
}

func TestInitReportsHowPublishingWillGoAndStoresNothingAboutIt(t *testing.T) {
	proj := project(t)
	f := newFakeGH()
	f.branch = "master"
	f.rules = "pull_request"
	usePushFake(t, f)

	out, err := run(t, "init", "--agent", "claude", "--registry", "org/reg", "--project-name", "demo", "--owner", "me", "--technology", "Go")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Registry org/reg: master requires pull requests, so push and import open one") {
		t.Errorf("init should report how publishing goes:\n%s", out)
	}
	data, err := os.ReadFile(filepath.Join(proj, ".hikma", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(data)
	for _, word := range []string{"master", "pull", "mode", "branch"} {
		if strings.Contains(cfg, word) {
			t.Errorf("project config must not store what GitHub knows (found %q):\n%s", word, cfg)
		}
	}
}

func TestInitStaysQuietWithoutGHAndDoesNotContactGitHubOnADryRun(t *testing.T) {
	project(t)
	old := ghRun
	ghRun = func(string, ...string) (string, error) { return "", fmt.Errorf("gh: not signed in") }
	t.Cleanup(func() { ghRun = old })
	out, err := run(t, "init", "--agent", "claude", "--registry", "org/reg", "--project-name", "demo", "--owner", "me", "--technology", "Go")
	if err != nil || strings.Contains(out, "Registry org/reg:") {
		t.Fatalf("init must succeed and say nothing without gh, got %v\n%s", err, out)
	}

	noGH(t)
	if out, err := run(t, "init", "--agent", "claude", "--registry", "org/reg", "--dry-run", "--project-name", "demo"); err != nil {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
}
