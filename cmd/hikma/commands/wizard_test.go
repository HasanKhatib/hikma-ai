package commands

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakePrompter struct {
	pick     int
	picks    []int // when set, successive Select calls take these in order
	inputs   []string
	confirms []bool
	multi    []int
	err      error
	asked    []string
	options  []string
	gotDef   int
	gotDefs  []int
	gotMin   int
	gotConf  bool
}

func (f *fakePrompter) Select(message string, options []string, def int) (int, error) {
	f.asked = append(f.asked, message)
	f.options = options
	f.gotDef = def
	if len(f.picks) > 0 {
		p := f.picks[0]
		f.picks = f.picks[1:]
		return p, f.err
	}
	return f.pick, f.err
}

func (f *fakePrompter) MultiSelect(message string, options []string, defaults []int, min int) ([]int, error) {
	f.asked = append(f.asked, message)
	f.options, f.gotDefs, f.gotMin = options, defaults, min
	return f.multi, f.err
}

func (f *fakePrompter) Confirm(message string, def bool) (bool, error) {
	f.asked = append(f.asked, message)
	f.gotConf = def
	if f.err != nil {
		return false, f.err
	}
	if len(f.confirms) == 0 {
		return def, nil
	}
	ok := f.confirms[0]
	f.confirms = f.confirms[1:]
	return ok, nil
}

func (f *fakePrompter) Input(message string, validate func(string) error) (string, error) {
	f.asked = append(f.asked, message)
	if f.err != nil {
		return "", f.err
	}
	in := f.inputs[0]
	f.inputs = f.inputs[1:]
	if validate != nil {
		if err := validate(in); err != nil {
			return "", err
		}
	}
	return in, nil
}

func indexOf(t *testing.T, label string) int {
	t.Helper()
	for i, e := range wizardEntries {
		if e.label == label {
			return i
		}
	}
	t.Fatalf("no wizard entry %q", label)
	return -1
}

func TestWizardRunsTheFlagCommandAndPrintsIt(t *testing.T) {
	project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills") // configured, so the menu has nothing to ask for install
	t.Setenv("HIKMA_AGENT", "claude")
	cases := []struct {
		label  string
		inputs []string
		want   []string
		equiv  string
	}{
		{"Install a skill from your registry", nil, []string{"skill", "install"}, "hikma skill install"},
		{"Create a new skill", []string{" my-skill "}, []string{"skill", "create", "my-skill"}, "hikma skill create my-skill"},
		{"Set your registry", []string{"acme/skills"}, []string{"config", "registry", "acme/skills"}, "hikma config registry acme/skills"},
		{"Check your setup", nil, []string{"doctor"}, "hikma doctor"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			p := &fakePrompter{pick: indexOf(t, c.label), inputs: c.inputs}
			var out bytes.Buffer
			var ran []string
			if err := runWizard(p, &out, func(a []string) error { ran = a; return nil }); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ran, c.want) {
				t.Errorf("ran %v, want %v", ran, c.want)
			}
			if !strings.Contains(out.String(), "Equivalent command:\n  "+c.equiv+"\n") {
				t.Errorf("missing equivalent command %q in:\n%s", c.equiv, out.String())
			}
			if strings.Contains(out.String(), "\x1b") {
				t.Errorf("escape codes in non-terminal output: %q", out.String())
			}
		})
	}
}

func TestWizardQuitRunsNothing(t *testing.T) {
	p := &fakePrompter{pick: len(wizardEntries)}
	var out bytes.Buffer
	called := false
	if err := runWizard(p, &out, func([]string) error { called = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("Quit must not run a command")
	}
	if strings.Contains(out.String(), "Equivalent command") {
		t.Error("Quit must not print an equivalent command")
	}
	if last := p.options[len(p.options)-1]; last != wizardQuit {
		t.Errorf("last option is %q, want %q", last, wizardQuit)
	}
}

func TestWizardCancelIsQuiet(t *testing.T) {
	p := &fakePrompter{err: errCancelled}
	var out bytes.Buffer
	if err := runWizard(p, &out, func([]string) error { t.Fatal("ran"); return nil }); err != nil {
		t.Fatalf("Ctrl-C must exit quietly, got %v", err)
	}
}

func TestWizardPassesCommandErrorThrough(t *testing.T) {
	boom := errors.New("boom")
	p := &fakePrompter{pick: indexOf(t, "Check your setup")}
	var out bytes.Buffer
	err := runWizard(p, &out, func([]string) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("got %v, want the command's error", err)
	}
	if !strings.Contains(out.String(), "hikma doctor") {
		t.Error("the equivalent command is printed even when the command fails")
	}
}

func TestWizardRejectsBadNames(t *testing.T) {
	for in, want := range map[string]string{
		"two words": "no spaces",
		"   ":       "enter a value",
		"--yes":     "dash",
		" -h":       "dash",
	} {
		p := &fakePrompter{pick: indexOf(t, "Create a new skill"), inputs: []string{in}}
		var out bytes.Buffer
		err := runWizard(p, &out, func([]string) error { t.Fatalf("%q ran", in); return nil })
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want an error containing %q", in, err, want)
		}
	}
}

func TestWizardCtrlCInsideTheCommandIsQuiet(t *testing.T) {
	p := &fakePrompter{pick: indexOf(t, "Install shell completion")}
	var out bytes.Buffer
	err := runWizard(p, &out, func([]string) error { return errCancelled })
	if err != nil {
		t.Fatalf("got %v, want a quiet exit", err)
	}
}

func TestEveryWizardEntryIsARealCommand(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	root := Root()
	for _, e := range wizardEntries {
		args, err := e.collect(&fakePrompter{inputs: []string{"acme/skills"}, picks: []int{0}, multi: []int{0}})
		if err != nil {
			t.Fatalf("%s: %v", e.label, err)
		}
		cmd, _, err := root.Find(args)
		if err != nil || cmd == root {
			t.Errorf("%s: %v does not resolve to a subcommand (%v)", e.label, args, err)
		}
	}
}

func TestEquivalentQuotesWhatTheShellWouldSplit(t *testing.T) {
	got := equivalent([]string{"skill", "create", "it's here"})
	if got != `hikma skill create 'it'\''s here'` {
		t.Errorf("got %s", got)
	}
}

func TestBareHikmaPrintsHelpWithoutATerminal(t *testing.T) {
	root := Root()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(nil)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Usage:") || strings.Contains(out.String(), "What do you want to do?") {
		t.Errorf("expected help text, got:\n%s", out.String())
	}
}

// runEntry drives one menu entry with a fake prompter and returns the command it ran.
func runEntry(t *testing.T, label string, p *fakePrompter) ([]string, error) {
	t.Helper()
	// The first Select picks the entry; later ones answer the entry's own questions.
	p.picks = append([]int{indexOf(t, label)}, p.picks...)
	var out bytes.Buffer
	var ran []string
	err := runWizard(p, &out, func(a []string) error { ran = a; return nil })
	return ran, err
}

func agentIndex(t *testing.T, a string) int {
	t.Helper()
	names, _, err := agentChoices()
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range names {
		if string(n) == a {
			return i
		}
	}
	t.Fatalf("no agent %q", a)
	return -1
}

func TestWizardInstallAsksForRegistryAndAgentWhenUnset(t *testing.T) {
	project(t)
	p := &fakePrompter{inputs: []string{" acme/skills "}, picks: []int{agentIndex(t, "claude")}}
	ran, err := runEntry(t, "Install a skill from your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "install", "acme/skills", "--agent", "claude"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

func TestWizardBrowseAsksForARegistryOnlyWhenNoneIsSet(t *testing.T) {
	project(t)
	ran, err := runEntry(t, "Browse skills in your registry", &fakePrompter{inputs: []string{"acme/skills"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "list", "acme/skills"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	ran, _ = runEntry(t, "Browse skills in your registry", &fakePrompter{})
	if want := []string{"skill", "list"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("with a registry set, ran %v, want %v", ran, want)
	}
}

func TestWizardRejectsABadRegistry(t *testing.T) {
	project(t)
	for _, in := range []string{"", "-x", "not a repo"} {
		if _, err := runEntry(t, "Set your registry", &fakePrompter{inputs: []string{in}}); err == nil {
			t.Errorf("registry %q was accepted", in)
		}
	}
}

func TestWizardPublishPicksALocalSkill(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	write(t, filepath.Join(dir, ".claude/skills/beta/SKILL.md"), "---\nname: beta\ndescription: d\n---\n")
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	p := &fakePrompter{picks: []int{1}} // names are sorted: alpha, beta
	ran, err := runEntry(t, "Publish a skill to your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "push", "beta"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if !reflect.DeepEqual(p.options, []string{"alpha", "beta"}) {
		t.Errorf("picker options = %v", p.options)
	}
}

func TestWizardPublishAsksForARegistryWhenNoneIsSet(t *testing.T) {
	dir := project(t)
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	ran, err := runEntry(t, "Publish a skill to your registry", &fakePrompter{inputs: []string{"acme/skills"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "push", "alpha", "--registry", "acme/skills"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

func TestWizardPublishAndUpdateSayWhatToDoWithNoSkills(t *testing.T) {
	project(t)
	for _, label := range []string{"Publish a skill to your registry", "Update installed skills"} {
		_, err := runEntry(t, label, &fakePrompter{})
		if err == nil || !strings.Contains(err.Error(), "first") {
			t.Errorf("%s: want an error saying what to do first, got %v", label, err)
		}
	}
}

func TestWizardUpdateOffersAllOrOneSkill(t *testing.T) {
	dir := project(t)
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	ran, err := runEntry(t, "Update installed skills", &fakePrompter{picks: []int{0}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "update", "--all"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	ran, _ = runEntry(t, "Update installed skills", &fakePrompter{picks: []int{1}})
	if want := []string{"skill", "update", "alpha"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

func TestWizardInitAsksForAgentsAndAnOptionalRegistry(t *testing.T) {
	project(t)
	claude, codex := agentIndex(t, "claude"), agentIndex(t, "codex")
	p := &fakePrompter{multi: []int{codex, claude}, inputs: []string{"acme/skills"}}
	ran, err := runEntry(t, "Set up this project for your agent", p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"init", "--agent", "codex,claude", "--registry", "acme/skills"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if p.gotMin != 1 {
		t.Errorf("must pick at least one agent, min = %d", p.gotMin)
	}
	ran, _ = runEntry(t, "Set up this project for your agent", &fakePrompter{multi: []int{claude}, inputs: []string{""}})
	if want := []string{"init", "--agent", "claude"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("empty registry: ran %v, want %v", ran, want)
	}
}

func TestWizardChooseAgentPrintsTheAgent(t *testing.T) {
	project(t)
	ran, err := runEntry(t, "Choose your agent", &fakePrompter{picks: []int{agentIndex(t, "claude")}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"config", "agent", "claude"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}
