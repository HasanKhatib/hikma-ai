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
	multis   [][]int // when set, successive MultiSelect calls take these in order
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
	if len(f.multis) > 0 {
		m := f.multis[0]
		f.multis = f.multis[1:]
		return m, f.err
	}
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
		{"Create a new skill", []string{" my-skill ", ""}, nil, []string{"skill", "create", "my-skill"}, "hikma skill create my-skill"},
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
	for _, want := range []string{"Will install 1 skill from acme/skills", "  beta  does beta\n", ".claude/skills"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
	if len(p.picker) != 2 || !strings.HasPrefix(p.picker[0], "beta") || !strings.HasPrefix(p.picker[1], "[installed] alpha") {
		t.Errorf("want the new skill first and the installed one last, marked in words; got %v", p.picker)
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

	// Everything installed is not a failure: the list is shown marked, with what to
	// do next, and no "error:".
	stubSkills(t, "alpha", "beta")
	for _, n := range []string{"alpha", "beta"} {
		write(t, filepath.Join(dir, ".claude/skills/"+n+"/SKILL.md"), "---\nname: "+n+"\ndescription: d\n---\n")
	}
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Install skills from your registry", &fakePrompter{}, &out)
	if err != nil {
		t.Fatalf("all installed must not be shown as an error, got %v", err)
	}
	if len(ran) != 0 {
		t.Errorf("ran %v", ran)
	}
	for _, want := range []string{"[installed] alpha", "[installed] beta", "Everything in acme/skills is already installed here", "hikma skill update --all"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "error:") {
		t.Errorf("no 'error:' for a state that is not a failure:\n%s", out.String())
	}
}

// ticksThen answers the install picker with each list in turn.
type ticksThen struct {
	*fakePrompter
	answers [][]int
	asked   int
}

func (p *ticksThen) MultiSelect(message string, options []string, defaults []int, atLeast int) ([]int, error) {
	p.fakePrompter.MultiSelect(message, options, defaults, atLeast)
	a := p.answers[min(p.asked, len(p.answers)-1)]
	p.asked++
	return a, nil
}

func TestTickingAnInstalledSkillIsIgnoredAndAsksAgain(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "alpha", "beta", "gamma")
	write(t, filepath.Join(dir, ".claude/skills/alpha/SKILL.md"), "---\nname: alpha\ndescription: d\n---\n")
	// The list is beta, gamma, then [installed] alpha. First only alpha is ticked,
	// then alpha and gamma.
	fp := &fakePrompter{scripted: true, quitNext: true, picks: pathTo(t, "Install skills from your registry")}
	p := &ticksThen{fakePrompter: fp, answers: [][]int{{2}, {2, 1}}}
	var out bytes.Buffer
	var got [][]string
	if err := runWizard(p, &out, func(_ context.Context, a []string) error { got = append(got, a); return nil }); err != nil {
		t.Fatal(err)
	}
	if p.asked != 2 {
		t.Errorf("the picker was shown %d times, want 2: the first tick was only an installed skill", p.asked)
	}
	if !strings.Contains(out.String(), "Those are already installed") {
		t.Errorf("the user should be told why the picker came back:\n%s", out.String())
	}
	if want := [][]string{{"skill", "install", "gamma"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("ran %v, want %v: the installed alpha must be ignored", got, want)
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
			return &installFailed{Failed: []string{"beta"}, Retryable: []string{"beta"}, Total: 3}
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
	// After the retry worked, the equivalent command is the whole install, not the
	// one skill that was retried.
	if !strings.Contains(out.String(), "Equivalent command for the whole install:\n  hikma skill install --skill=alpha --skill=beta --skill=gamma") {
		t.Errorf("want the whole install as the equivalent:\n%s", out.String())
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
		cmds, err := e.collect(&fakePrompter{inputs: []string{"acme/skills", "a description"}, picks: []int{0}, multi: []int{0}, confirms: []bool{true}}, io.Discard)
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

func skillFile(t *testing.T, dir, folder, name string) {
	t.Helper()
	write(t, filepath.Join(dir, folder, name, "SKILL.md"), "---\nname: "+name+"\ndescription: d\n---\n")
}

func TestWizardPublishTakesSeveralSkillsAsOneCommandEach(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t) // nothing published yet, so nothing is marked
	for _, n := range []string{"alpha", "beta", "gamma"} {
		skillFile(t, dir, ".claude/skills", n)
	}
	p := &fakePrompter{multi: []int{0, 2}, confirms: []bool{true}}
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Publish skills to your registry", p, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"skill", "push", "alpha", "--yes"}, {"skill", "push", "gamma", "--yes"}}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v: one push, so one commit, per skill", ran, want)
	}
	if !reflect.DeepEqual(p.picker, []string{"alpha", "beta", "gamma"}) {
		t.Errorf("picker options = %v", p.picker)
	}
	if !strings.Contains(out.String(), "Will publish 2 skills to acme/skills") {
		t.Errorf("missing plan:\n%s", out.String())
	}
}

func TestWizardPublishDeclineWritesNothing(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	stubSkills(t)
	skillFile(t, dir, ".claude/skills", "alpha")
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Publish skills to your registry", &fakePrompter{multi: []int{0}, confirms: []bool{false}}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 || !strings.Contains(out.String(), "Nothing was published.") {
		t.Errorf("ran %v:\n%s", ran, out.String())
	}
}

func TestWizardPublishOffersEveryAgentFolderAndPassesTheAgentForOtherFolders(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "codex") // .agents/skills is active
	stubSkills(t)
	skillFile(t, dir, ".agents/skills", "mine")
	skillFile(t, dir, ".claude/skills", "video-pixel-scene")
	p := &fakePrompter{multi: []int{0, 1}, confirms: []bool{true}}
	ran, err := runEntryAll(t, "Publish skills to your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	wantOpts := []string{"mine  (.agents/skills)", "video-pixel-scene  (.claude/skills)"}
	if !reflect.DeepEqual(p.picker, wantOpts) {
		t.Errorf("picker options = %q, want %q: the folder is shown when there are several", p.options, wantOpts)
	}
	want := [][]string{
		{"skill", "push", "mine", "--yes"},
		{"skill", "push", "video-pixel-scene", "--agent", "claude", "--yes"},
	}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

func TestWizardPublishRefusesTheSameSkillFromTwoFolders(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t)
	skillFile(t, dir, ".agents/skills", "twin")
	skillFile(t, dir, ".claude/skills", "twin")
	var out bytes.Buffer
	_, err := runEntryTo(t, "Publish skills to your registry", &fakePrompter{multis: [][]int{{0, 1}, {1}}, confirms: []bool{true}}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "twin is ticked from two folders") {
		t.Errorf("want the duplicate named:\n%s", out.String())
	}
}

func TestWizardPublishMarksTheSkillsThatAlreadyExistInTheRegistry(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "Beta", "other") // the registry has Beta (any letter case) and a skill we do not have
	for _, n := range []string{"alpha", "beta"} {
		skillFile(t, dir, ".claude/skills", n)
	}
	p := &fakePrompter{multi: []int{1}, confirms: []bool{true}}
	ran, err := runEntryAll(t, "Publish skills to your registry", p)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"skill", "push", "beta", "--yes"}}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	want := []string{"         alpha", "[exists] beta"}
	if !reflect.DeepEqual(p.picker, want) {
		t.Errorf("picker options = %q, want %q: marked in words, names aligned", p.options, want)
	}
}

func TestWizardPublishStillWorksWhenTheRegistryCannotBeRead(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	old := registrySkills
	registrySkills = func(context.Context, string) ([]source.Skill, error) { return nil, errors.New("repository is empty") }
	t.Cleanup(func() { registrySkills = old })
	skillFile(t, dir, ".claude/skills", "alpha")
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Publish skills to your registry", &fakePrompter{multi: []int{0}, confirms: []bool{true}}, &out)
	if err != nil {
		t.Fatalf("an unreadable registry must not stop a publish: %v", err)
	}
	if want := [][]string{{"skill", "push", "alpha", "--yes"}}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if !strings.Contains(out.String(), "nothing is marked") || !strings.Contains(out.String(), "repository is empty") {
		t.Errorf("the user should be told why nothing is marked:\n%s", out.String())
	}
}

func TestPushFindsASkillInAnotherAgentsFolderAndSaysSo(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_AGENT", "codex")
	skillFile(t, dir, ".claude/skills", "video-pixel-scene")
	err := skillNotFound("video-pixel-scene", filepath.Join(".agents", "skills", "video-pixel-scene"))
	if err == nil || !strings.Contains(err.Error(), "hikma skill push video-pixel-scene --agent claude") {
		t.Errorf("want the --agent that finds it, got %v", err)
	}
	if strings.Contains(err.Error(), "skill create") {
		t.Errorf("a scaffold would create a second copy: %v", err)
	}
	err = skillNotFound("nope", filepath.Join(".agents", "skills", "nope"))
	if err == nil || !strings.Contains(err.Error(), "hikma skill create nope") {
		t.Errorf("a missing skill still points at create, got %v", err)
	}
}

func TestSetupIsOfferedOnceInAProjectThatHasNone(t *testing.T) {
	project(t)
	var ran [][]string
	run := func(_ context.Context, a []string) error { ran = append(ran, a); return nil }
	var out bytes.Buffer
	p := &fakePrompter{picks: []int{agentIndex(t, "claude"), 0}, multi: []int{agentIndex(t, "claude")}, inputs: []string{"acme/skills"}}
	quit, err := offerSetup(p, &out, run)
	if err != nil || quit {
		t.Fatalf("quit=%v err=%v", quit, err)
	}
	want := [][]string{{"init", "--agent", "claude", "--registry", "acme/skills"}}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if !strings.Contains(out.String(), "not set up for hikma yet") {
		t.Errorf("missing explanation:\n%s", out.String())
	}
}

func TestSetupOfferCanBeDeclinedAndIsSkippedWhenSetUp(t *testing.T) {
	dir := project(t)
	var out bytes.Buffer
	quit, err := offerSetup(&fakePrompter{confirms: []bool{false}}, &out, nil)
	if err != nil || quit {
		t.Fatalf("declining goes to the menu: quit=%v err=%v", quit, err)
	}
	write(t, filepath.Join(dir, ".hikma", "config.json"), "{\"version\":1}")
	out.Reset()
	p := &fakePrompter{}
	if quit, err := offerSetup(p, &out, nil); quit || err != nil || len(p.asked) != 0 || out.Len() != 0 {
		t.Errorf("a set-up project is not asked: quit=%v err=%v asked=%v out=%q", quit, err, p.asked, out.String())
	}
}

func TestSetupIsNotOfferedOutsideAGitRepository(t *testing.T) {
	project(t)
	t.Chdir(t.TempDir())
	if needsSetup() {
		t.Error("a folder that is not a git repository must not be offered setup")
	}
}

func TestVerifyEntryExplainsWhatItChecks(t *testing.T) {
	project(t)
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Check installed skills for local edits", &fakePrompter{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"skill", "verify"}}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if !strings.Contains(out.String(), ".hikma/lock.json") || !strings.Contains(out.String(), "changes nothing") {
		t.Errorf("explain the check before running it:\n%s", out.String())
	}
}

func TestAFailedCommandSaysWhichOnesDidNotRun(t *testing.T) {
	var out bytes.Buffer
	run := func(_ context.Context, a []string) error { return errors.New("boom") }
	execute(&out, [][]string{{"skill", "push", "a"}, {"skill", "push", "b"}}, run, nil)
	if !strings.Contains(out.String(), "did not run") || !strings.Contains(out.String(), "hikma skill push b") {
		t.Errorf("want the skipped command listed:\n%s", out.String())
	}
}

func TestMarkLabelsAddsNoColumnWhenNothingIsMarked(t *testing.T) {
	got := markLabels([]string{"a", "bb"}, func(string) bool { return false }, "exists")
	if !reflect.DeepEqual(got, []string{"a", "bb"}) {
		t.Errorf("got %q", got)
	}
	got = markLabels([]string{"a", "bb"}, nil, "exists")
	if !reflect.DeepEqual(got, []string{"a", "bb"}) {
		t.Errorf("nil marker: got %q", got)
	}
}

func TestWizardPublishAndUpdateSayWhatToDoWithNoSkills(t *testing.T) {
	project(t)
	for _, label := range []string{"Publish skills to your registry", "Update installed skills"} {
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
	verify := indexOf(t, "Check installed skills for local edits")
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
			return &installFailed{Failed: []string{"beta"}, Retryable: []string{"beta"}, Total: 2}
		})
	}()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"error: 1 of 2 skills were not installed",
		"Equivalent command:\n  hikma skill install --skill=alpha --skill=beta",
		"Retry only the ones that can still work:\n  hikma skill install --skill=beta",
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
	picks := append(pathTo(t, "Check your setup"), 0, indexOf(t, "Check installed skills for local edits"), 0, len(wizardEntries))
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

func TestWizardCreateAsksForADescriptionAndPointsAtTheGuide(t *testing.T) {
	project(t)
	p := &fakePrompter{inputs: []string{"deploy-helper", `  Deploy a service and "verify" it.
Use when asked to roll out.  `}}
	var out bytes.Buffer
	ran, err := runEntryTo(t, "Create a new skill", p, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"skill", "create", "deploy-helper", `--description=Deploy a service and "verify" it. Use when asked to roll out.`}}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if !strings.Contains(out.String(), skillGuideURL) {
		t.Errorf("the prompt text should link the guide:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "--description=") {
		t.Errorf("the equivalent command should carry the description:\n%s", out.String())
	}
}

func TestWizardCreateDescriptionIsOptionalAndBounded(t *testing.T) {
	if err := validDescription(""); err != nil {
		t.Errorf("an empty description must be allowed: %v", err)
	}
	if err := validDescription(strings.Repeat("x", 301)); err == nil {
		t.Error("a description over 300 characters should be refused")
	}
	// A description that starts with a dash is a value, not a flag.
	ran, err := runEntry(t, "Create a new skill", &fakePrompter{inputs: []string{"s", "-not a flag"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"skill", "create", "s", "--description=-not a flag"}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

func TestRetryIsNotOfferedForSkillsThatAreAlreadyInstalled(t *testing.T) {
	project(t)
	t.Setenv("HIKMA_REGISTRY", "acme/skills")
	t.Setenv("HIKMA_AGENT", "claude")
	stubSkills(t, "alpha", "beta")
	p := &fakePrompter{scripted: true, multi: []int{0, 1}, picks: append(pathTo(t, "Install skills from your registry"), 0, len(wizardEntries))}
	var out bytes.Buffer
	err := runWizard(p, &out, func(context.Context, []string) error {
		// Both failed, but only because they are already installed: trying again,
		// without --force, would fail the same way.
		return &installFailed{Failed: []string{"alpha", "beta"}, Retryable: nil, Total: 2}
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range p.options {
		if o == wizardRetry {
			t.Errorf("a retry that cannot succeed was offered: %v", p.options)
		}
	}
	if strings.Contains(out.String(), "Retry only") {
		t.Errorf("no retry command should be printed:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "2 already installed, so trying again would not help") {
		t.Errorf("the user should be told why, and what to do:\n%s", out.String())
	}
}

func TestContextLineSaysWhenTheSkillsFolderCannotBeRead(t *testing.T) {
	dir := project(t)
	t.Setenv("HIKMA_AGENT", "claude")
	if got := wizardContext(); !strings.Contains(got, "0 installed here") {
		t.Fatalf("an empty project is 0 installed: %q", got)
	}
	// A skills folder that is a file cannot be listed. That is not "0 installed".
	write(t, filepath.Join(dir, ".claude", "skills"), "not a folder")
	got := wizardContext()
	if strings.Contains(got, "0 installed") {
		t.Skipf("this system lists an unreadable folder as empty: %q", got)
	}
	if !strings.Contains(got, "unknown") {
		t.Errorf("want the unreadable folder named as unknown, got %q", got)
	}
}
