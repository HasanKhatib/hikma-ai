package commands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/source"
)

type fakePrompter struct {
	pick     int
	picks    []int // when set, successive Select calls take these in order
	scripted bool  // with picks: once they run out, the menu question cancels, which ends the loop
	quitNext bool  // answer "What next?" with Quit
	inputs   []string
	confirms []bool
	multi    []int
	err      error
	asked    []string
	options  []string
	picker   []string // the options of the last MultiSelect
	gotDef   int
	defs     []int // the cursor default of every Select, in order
	gotDefs  []int
	gotMin   int
	gotConf  bool
}

func (f *fakePrompter) Select(message string, options []string, def int) (int, error) {
	f.asked = append(f.asked, message)
	f.gotDef = def
	f.defs = append(f.defs, def)
	if message == "What next?" && f.quitNext {
		return len(options) - 1, nil // Quit is always last
	}
	if message != "What next?" {
		f.options = options
	}
	if len(f.picks) > 0 {
		p := f.picks[0]
		f.picks = f.picks[1:]
		return p, f.err
	}
	if f.scripted && message == "What do you want to do?" {
		return 0, errCancelled
	}
	return f.pick, f.err
}

func (f *fakePrompter) Spinner(message string, fn func(context.Context) error) error {
	f.asked = append(f.asked, message)
	return fn(context.Background())
}

func (f *fakePrompter) MultiSelect(message string, options []string, defaults []int, min int) ([]int, error) {
	f.asked = append(f.asked, message)
	f.options, f.picker, f.gotDefs, f.gotMin = options, options, defaults, min
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

// pathTo is the menu picks that reach an entry: one for a top-level entry, two
// for an entry under a submenu.
func pathTo(t *testing.T, label string) []int {
	t.Helper()
	for i, e := range wizardEntries {
		if e.label == label {
			return []int{i}
		}
		for j, sub := range e.sub {
			if sub.label == label {
				return []int{i, j}
			}
		}
	}
	t.Fatalf("no wizard entry %q", label)
	return nil
}

// indexOf is the top-level menu index of an entry.
func indexOf(t *testing.T, label string) int { return pathTo(t, label)[0] }

// leaves is every entry that runs something, submenus flattened.
func leaves() []wizardEntry {
	var out []wizardEntry
	for _, e := range wizardEntries {
		if e.sub == nil {
			out = append(out, e)
		}
		out = append(out, e.sub...)
	}
	return out
}

func stubSkills(t *testing.T, names ...string) {
	t.Helper()
	old := registrySkills
	registrySkills = func(context.Context, string) ([]source.Skill, error) {
		var out []source.Skill
		for _, n := range names {
			out = append(out, source.Skill{Name: n, Description: "does " + n})
		}
		return out, nil
	}
	t.Cleanup(func() { registrySkills = old })
}

func TestWizardRunsTheFlagCommandAndPrintsIt(t *testing.T) {
	project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills") // configured, so the menu has nothing to ask for install
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "alpha", "beta")
	cases := []struct {
		label  string
		inputs []string
		multi  []int
		want   []string
		equiv  string
	}{
		{"Install skills from your registry", nil, []int{1}, []string{"skill", "install", "beta"}, "hikma skill install beta"},
		{"Create a new skill", []string{" my-skill "}, nil, []string{"skill", "create", "my-skill"}, "hikma skill create my-skill"},
		{"Set your registry", []string{"acme/skills"}, nil, []string{"config", "registry", "acme/skills"}, "hikma config registry acme/skills"},
		{"Check your setup", nil, nil, []string{"doctor"}, "hikma doctor"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			p := &fakePrompter{inputs: c.inputs, multi: c.multi}
			var out bytes.Buffer
			ran, err := runEntryTo(t, c.label, p, &out)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ran, [][]string{c.want}) {
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

func TestWizardInstallTakesSeveralSkillsAsOneCommand(t *testing.T) {
	project(t)
	stubSkills(t, "alpha", "-evil", "beta", "gamma")
	p := &fakePrompter{inputs: []string{"acme/skills"}, multi: []int{0, 2}, picks: []int{agentIndex(t, "claude")}}
	ran, err := runEntryAll(t, "Install skills from your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"skill", "install", "acme/skills", "--skill=alpha", "--skill=gamma", "--agent", "claude"}}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	for _, o := range p.picker {
		if strings.HasPrefix(o, "-") {
			t.Errorf("a skill named like a flag was offered: %q", o)
		}
	}
}

func TestWizardInstallShowsAPlanAndDeclineWritesNothing(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "alpha", "beta")
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	p := &fakePrompter{multi: []int{0}, confirms: []bool{false}}
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Install skills from your registry", p, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 {
		t.Errorf("declining the plan ran %v", ran)
	}
	for _, want := range []string{"Already installed", "alpha", "Will install 1 skill from acme/skills", "  beta  does beta\n", ".claude/skills"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
	if len(p.picker) != 1 || !strings.HasPrefix(p.picker[0], "beta") {
		t.Errorf("an installed skill must not be offered, got %v", p.picker)
	}
}

func TestWizardInstallExplainsAnEmptyOrFullRegistry(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t)
	_, err := runEntry(t, "Install skills from your registry", &fakePrompter{})
	if err == nil || !strings.Contains(err.Error(), "acme/skills has no skills") || !strings.Contains(err.Error(), "hikma config registry") {
		t.Errorf("empty registry: got %v", err)
	}
	stubSkills(t, "alpha")
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	_, err = runEntry(t, "Install skills from your registry", &fakePrompter{})
	if err == nil || !strings.Contains(err.Error(), "already installed") || !strings.Contains(err.Error(), "skill update --all") {
		t.Errorf("all installed: got %v", err)
	}
}

func TestWizardRetriesOnlyTheSkillsThatFailed(t *testing.T) {
	project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "alpha", "beta", "gamma")
	p := &fakePrompter{scripted: true, multi: []int{0, 1, 2}, picks: append(pathTo(t, "Install skills from your registry"), 1, 1)}
	// "What next?" after the failure offers Back, Retry, Quit: take Retry (1).
	// After the retry works it offers Back, Quit: take Quit (1).
	var ran [][]string
	var out bytes.Buffer
	err := runWizard(p, &out, func(_ context.Context, a []string) error {
		ran = append(ran, a)
		if len(ran) == 1 {
			return &installFailed{Failed: []string{"beta"}, Total: 3}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"skill", "install", "--skill=alpha", "--skill=beta", "--skill=gamma"},
		{"skill", "install", "--skill=beta"},
	}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

func TestInstallPlanAndResultRenderAsPlainText(t *testing.T) {
	plan := renderInstallPlan(installPlan{Source: "acme/skills", Skills: []planSkill{{"foo", "Lint a repo"}, {"bar", ""}, {"café", "Accents count as one column"}}, Agents: []string{"claude"}, Dirs: []string{".claude/skills"}})
	if want := "Will install 3 skills from acme/skills\n  foo   Lint a repo\n  bar\n  café  Accents count as one column\nInto .claude/skills (agent: claude)\n"; plan != want {
		t.Errorf("plan:\n%q\nwant\n%q", plan, want)
	}
	res := renderResult([]skillResult{{Name: "foo"}, {Name: "bar"}, {Name: "baz", Err: errors.New("not found in acme/skills\nmore")}})
	want := "Installed 2 of 3\n  ok      foo\n  ok      bar\n  failed  baz  not found in acme/skills; more\n"
	if res != want {
		t.Errorf("result:\n%q\nwant\n%q", res, want)
	}
}

func TestWizardInstallFailureShowsTheErrorAndReturnsToTheMenu(t *testing.T) {
	project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	old := registrySkills
	registrySkills = func(context.Context, string) ([]source.Skill, error) { return nil, errors.New("clone failed") }
	t.Cleanup(func() { registrySkills = old })
	var out bytes.Buffer
	_, err := runEntryTo(t, "Install skills from your registry", &fakePrompter{}, &out)
	if err == nil || !strings.Contains(err.Error(), "clone failed") {
		t.Fatalf("got %v, want the fetch error shown", err)
	}
}

func TestWizardStaysUntilTheUserQuits(t *testing.T) {
	project(t)
	doctor := pathTo(t, "Check your setup")
	picks := append(append([]int{}, doctor...), 0) // run doctor, back to the menu
	picks = append(picks, doctor...)
	picks = append(picks, 0, len(wizardEntries)) // run it again, then Quit from the menu
	p := &fakePrompter{scripted: true, picks: picks}
	var out bytes.Buffer
	runs := 0
	if err := runWizard(p, &out, func(context.Context, []string) error { runs++; return nil }); err != nil {
		t.Fatal(err)
	}
	if runs != 2 {
		t.Errorf("ran %d commands, want 2 before quitting", runs)
	}
	if n := strings.Count(out.String(), "skills follow you"); n != 3 {
		t.Errorf("menu drawn %d times, want 3", n)
	}
}

func TestWizardCtrlCInsideAnEntryGoesBackToTheMenu(t *testing.T) {
	project(t)
	// Input cancels, as Ctrl-C at the name prompt would.
	cp := &cancelInputPrompter{&fakePrompter{scripted: true, picks: []int{indexOf(t, "Create a new skill"), len(wizardEntries)}}}
	var out bytes.Buffer
	if err := runWizard(cp, &out, func(context.Context, []string) error { t.Fatal("ran"); return nil }); err != nil {
		t.Fatalf("got %v, want a clean exit through the menu", err)
	}
	if n := strings.Count(out.String(), "skills follow you"); n != 2 {
		t.Errorf("menu drawn %d times, want 2 (cancel returns to it)", n)
	}
}

type cancelInputPrompter struct{ *fakePrompter }

func (cancelInputPrompter) Input(string, func(string) error) (string, error) { return "", errCancelled }

func TestWizardQuitRunsNothing(t *testing.T) {
	p := &fakePrompter{pick: len(wizardEntries)}
	var out bytes.Buffer
	called := false
	if err := runWizard(p, &out, func(context.Context, []string) error { called = true; return nil }); err != nil {
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
	if err := runWizard(p, &out, func(context.Context, []string) error { t.Fatal("ran"); return nil }); err != nil {
		t.Fatalf("Ctrl-C must exit quietly, got %v", err)
	}
}

func TestWizardShowsACommandErrorAndKeepsGoing(t *testing.T) {
	boom := errors.New("boom")
	p := &fakePrompter{scripted: true, picks: append(pathTo(t, "Check your setup"), 0, len(wizardEntries))}
	var out bytes.Buffer
	if err := runWizard(p, &out, func(context.Context, []string) error { return boom }); err != nil {
		t.Fatalf("a failed command must not end the menu, got %v", err)
	}
	if !strings.Contains(out.String(), "error: boom") {
		t.Errorf("the error must be shown:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Equivalent command") {
		t.Error("the equivalent command is only printed for a task that worked")
	}
}

func TestWizardRejectsBadNames(t *testing.T) {
	for in, want := range map[string]string{
		"two words": "no spaces",
		"   ":       "enter a value",
		"--yes":     "dash",
		" -h":       "dash",
	} {
		ran, err := runEntryAll(t, "Create a new skill", &fakePrompter{inputs: []string{in}})
		if len(ran) != 0 {
			t.Errorf("%q ran %v", in, ran)
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want an error containing %q", in, err, want)
		}
	}
}

func TestWizardCtrlCInsideTheCommandIsQuiet(t *testing.T) {
	p := &fakePrompter{scripted: true, picks: append(pathTo(t, "Install shell completion"), 0, len(wizardEntries))}
	var out bytes.Buffer
	err := runWizard(p, &out, func(context.Context, []string) error { return errCancelled })
	if err != nil {
		t.Fatalf("got %v, want a quiet exit", err)
	}
	if strings.Contains(out.String(), "Equivalent command") {
		t.Error("a cancelled task prints no equivalent command")
	}
}

func TestEveryWizardEntryIsARealCommand(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	stubSkills(t, "alpha", "beta") // alpha is installed below, so the picker offers beta
	oldT := stdinIsTerminal
	stdinIsTerminal = func() bool { return true } // completion asks for its shell itself
	t.Cleanup(func() { stdinIsTerminal = oldT })
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	root := Root()
	for _, e := range leaves() {
		cmds, err := e.collect(&fakePrompter{inputs: []string{"acme/skills"}, picks: []int{0}, multi: []int{0}}, io.Discard)
		if err != nil {
			t.Fatalf("%s: %v", e.label, err)
		}
		for _, args := range cmds {
			cmd, _, err := root.Find(args)
			if err != nil || cmd == root {
				t.Errorf("%s: %v does not resolve to a subcommand (%v)", e.label, args, err)
			}
			// Every planned command must pass the command's own argument check:
			// this is what failed when install was planned with no skill name.
			rest := args[len(strings.Fields(cmd.CommandPath()))-1:]
			if err := cmd.ParseFlags(rest); err != nil {
				t.Errorf("%s: %v has a flag its command rejects: %v", e.label, args, err)
			} else if err := cmd.ValidateArgs(cmd.Flags().Args()); err != nil {
				t.Errorf("%s: %v is rejected by its own command: %v", e.label, args, err)
			}
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

// runEntryTo drives one menu entry with a fake prompter, then quits at "What
// next?". It returns every command the entry ran, and any error the menu showed.
func runEntryTo(t *testing.T, label string, p *fakePrompter, out *bytes.Buffer) ([][]string, error) {
	t.Helper()
	// The first Select picks the entry; later ones answer the entry's own
	// questions, and the fake quits at "What next?". A flow that cancels early
	// runs out of picks, and the menu ends.
	p.scripted, p.quitNext = true, true
	p.picks = append(pathTo(t, label), p.picks...)
	var ran [][]string
	if err := runWizard(p, out, func(_ context.Context, a []string) error { ran = append(ran, a); return nil }); err != nil {
		return ran, err
	}
	if _, msg, ok := strings.Cut(out.String(), "error: "); ok {
		return ran, errors.New(strings.SplitN(msg, "\n", 2)[0])
	}
	return ran, nil
}

func runEntryAll(t *testing.T, label string, p *fakePrompter) ([][]string, error) {
	t.Helper()
	var out bytes.Buffer
	return runEntryTo(t, label, p, &out)
}

// runEntry returns the first command the entry ran.
func runEntry(t *testing.T, label string, p *fakePrompter) ([]string, error) {
	t.Helper()
	all, err := runEntryAll(t, label, p)
	if len(all) == 0 {
		return nil, err
	}
	return all[0], err
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
	stubSkills(t, "alpha")
	p := &fakePrompter{inputs: []string{" acme/skills "}, picks: []int{agentIndex(t, "claude")}, multi: []int{0}}
	ran, err := runEntry(t, "Install skills from your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "install", "acme/skills", "alpha", "--agent", "claude"}; !reflect.DeepEqual(ran, want) {
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

func TestCtrlCAtWhatNextGoesBackToTheMenuNotOut(t *testing.T) {
	project(t)
	p := &cancelNextPrompter{fakePrompter: &fakePrompter{scripted: true, picks: append(pathTo(t, "Check your setup"), len(wizardEntries))}}
	var out bytes.Buffer
	if err := runWizard(p, &out, func(context.Context, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out.String(), "skills follow you"); n != 2 {
		t.Errorf("menu drawn %d times, want 2: Ctrl-C at What next? goes back, then Quit leaves", n)
	}
}

// cancelNextPrompter cancels (Ctrl-C) at "What next?" once, then behaves.
type cancelNextPrompter struct {
	*fakePrompter
	done bool
}

func (c *cancelNextPrompter) Select(message string, options []string, def int) (int, error) {
	if message == "What next?" && !c.done {
		c.done = true
		return 0, errCancelled
	}
	return c.fakePrompter.Select(message, options, def)
}

func TestMenuRemembersTheCursor(t *testing.T) {
	project(t)
	verify := indexOf(t, "Verify installed skills")
	// Verify, then "Back to the menu", then Quit from the menu.
	p := &fakePrompter{scripted: true, picks: []int{verify, 0, len(wizardEntries)}}
	var out bytes.Buffer
	if err := runWizard(p, &out, func(context.Context, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	// Selects, in order: menu, "What next?", menu.
	if len(p.defs) != 3 || p.defs[0] != 0 || p.defs[2] != verify {
		t.Errorf("cursor defaults %v: want the first draw at 0 and the second on the entry last used (%d)", p.defs, verify)
	}
}

func TestInstallAsksForAnotherRegistryAfterATypoAndKeepsTheAgent(t *testing.T) {
	project(t)
	calls := 0
	old := registrySkills
	registrySkills = func(_ context.Context, repo string) ([]source.Skill, error) {
		calls++
		if repo == "acme/skils" {
			return nil, errors.New("cannot read acme/skils")
		}
		return []source.Skill{{Name: "alpha"}}, nil
	}
	t.Cleanup(func() { registrySkills = old })
	p := &fakePrompter{inputs: []string{"acme/skils", "acme/skills"}, picks: []int{agentIndex(t, "claude")}, multi: []int{0}}
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Install skills from your registry", p, &out)
	// The typo's error is shown, not returned; the helper reports any shown error.
	if err != nil && !strings.Contains(err.Error(), "acme/skils") {
		t.Fatal(err)
	}
	want := [][]string{{"skill", "install", "acme/skills", "alpha", "--agent", "claude"}}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if calls != 2 || !strings.Contains(out.String(), "cannot read acme/skils") {
		t.Errorf("want the typo's error shown and a second try; calls=%d\n%s", calls, out.String())
	}
	agentQuestions := 0
	for _, m := range p.asked {
		if m == "AI agent:" {
			agentQuestions++
		}
	}
	if agentQuestions != 1 {
		t.Errorf("the agent was asked %d times, want once", agentQuestions)
	}
}

func TestInstallOffersASkillInstalledOnlyForAnotherAgent(t *testing.T) {
	dir := project(t)
	stubSkills(t, "alpha")
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	p := &fakePrompter{inputs: []string{"acme/skills"}, picks: []int{agentIndex(t, "codex")}, multi: []int{0}}
	ran, err := runEntry(t, "Install skills from your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "install", "acme/skills", "alpha", "--agent", "codex"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v: claude's copy must not hide it from codex", ran, want)
	}
}

// A cancelled spinner leaves the work running; its result must not be shared
// with the caller, or the two race. Run this package with -race to see it.
func TestReadRegistryDoesNotRaceWithACancelledSpinner(t *testing.T) {
	release := make(chan struct{})
	finished := make(chan struct{})
	old := registrySkills
	registrySkills = func(_ context.Context, _ string) ([]source.Skill, error) {
		defer close(finished)
		<-release
		return []source.Skill{{Name: "late"}}, nil
	}
	t.Cleanup(func() { registrySkills = old })
	p := &abandonSpinner{fakePrompter: &fakePrompter{}}
	if _, err := readRegistry(p, "acme/skills", ""); !errors.Is(err, errCancelled) {
		t.Fatalf("got %v, want errCancelled", err)
	}
	close(release)
	<-finished
}

type abandonSpinner struct{ *fakePrompter }

func (abandonSpinner) Spinner(_ string, fn func(context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	go fn(ctx)
	cancel()
	return errCancelled
}

func TestPartialInstallShowsWhatRanAndHowToRetryTheRest(t *testing.T) {
	project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "alpha", "beta")
	var out bytes.Buffer
	p := &fakePrompter{multi: []int{0, 1}}
	_, err := func() ([][]string, error) {
		p.scripted, p.quitNext = true, true
		p.picks = pathTo(t, "Install skills from your registry")
		return nil, runWizard(p, &out, func(context.Context, []string) error {
			return &installFailed{Failed: []string{"beta"}, Total: 2}
		})
	}()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"error: 1 of 2 skills were not installed",
		"Equivalent command:\n  hikma skill install --skill=alpha --skill=beta",
		"Retry only the ones that failed:\n  hikma skill install --skill=beta",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
}

func TestAccessibleModeDrawsTheBannerOnce(t *testing.T) {
	project(t)
	t.Setenv("ACCESSIBLE", "1")
	// Check your setup, back; Verify, back; then Quit from the menu.
	picks := append(pathTo(t, "Check your setup"), 0, indexOf(t, "Verify installed skills"), 0, len(wizardEntries))
	p := &fakePrompter{scripted: true, picks: picks}
	var out bytes.Buffer
	if err := runWizard(p, &out, func(context.Context, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out.String(), "skills follow you"); n != 1 {
		t.Errorf("banner drawn %d times in accessible mode, want once", n)
	}
	if n := strings.Count(out.String(), "registry:"); n < 2 {
		t.Errorf("the context line should still be drawn each time, got %d", n)
	}
}

func TestLongRegistryNamesAreShortened(t *testing.T) {
	long := "someone-with-a-very-long-name/a-registry-with-an-equally-long-name"
	if got := shortLabel(long); len([]rune(got)) != 40 || !strings.HasSuffix(got, "...") {
		t.Errorf("shortLabel = %q", got)
	}
	if got := shortLabel("acme/skills"); got != "acme/skills" {
		t.Errorf("a short name changes: %q", got)
	}
}
