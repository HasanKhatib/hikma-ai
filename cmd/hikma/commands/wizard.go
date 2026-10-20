package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// A wizard entry is a front end for a flag command: collect inputs, plan the
// argument list, then run the same command code a script would. It holds no
// logic of its own, so the flags stay the API.
//
// collect returns the commands to run, in order; one entry may run several, such
// as installing every skill the user ticked.
type wizardEntry struct {
	label   string
	collect func(p Prompter, out io.Writer) ([][]string, error)
	sub     []wizardEntry // set on a submenu; collect is unused then
}

// one adapts a collector that plans a single command.
func one(f func(Prompter) ([]string, error)) func(Prompter, io.Writer) ([][]string, error) {
	return func(p Prompter, _ io.Writer) ([][]string, error) {
		args, err := f(p)
		if err != nil {
			return nil, err
		}
		return [][]string{args}, nil
	}
}

func fixed(args ...string) func(Prompter) ([]string, error) {
	return func(Prompter) ([]string, error) { return args, nil }
}

func askName(message string, args ...string) func(Prompter) ([]string, error) {
	return func(p Prompter) ([]string, error) {
		name, err := p.Input(message, func(s string) error {
			s = strings.TrimSpace(s)
			switch {
			case s == "":
				return errors.New("enter a value")
			case strings.HasPrefix(s, "-"):
				// The value becomes a command argument, so a leading dash
				// would be read as a flag.
				return errors.New("cannot start with a dash")
			case strings.ContainsAny(s, " \t"):
				return errors.New("use one word, with no spaces")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return append(append([]string{}, args...), strings.TrimSpace(name)), nil
	}
}

// The menu asks for what a command needs, so the equivalent command it prints is
// complete and can be pasted into a script. Anything already configured is left
// to the command; only what is missing is asked.

func registryConfigured() bool {
	e, err := config.Lookup(config.KeyRegistry)
	return err == nil && e.Value != ""
}

func agentChosen() bool {
	e, err := config.Lookup(config.KeyAgent)
	return err == nil && e.Source != config.SourceDefault
}

func validRegistry(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("enter owner/repo")
	}
	if strings.HasPrefix(s, "-") {
		return errors.New("cannot start with a dash")
	}
	_, err := config.ParseRegistry(s)
	return err
}

func askRegistry(p Prompter, message string) (string, error) {
	r, err := p.Input(message, validRegistry)
	return strings.TrimSpace(r), err
}

// agentChoices lists the agents the user can pick, with a label for each.
func agentChoices() ([]config.Agent, []string, error) {
	set, err := config.Layouts()
	if err != nil {
		return nil, nil, err
	}
	names := set.Names()
	labels := make([]string, len(names))
	for i, a := range names {
		labels[i] = labelForAgent(set, a)
	}
	return names, labels, nil
}

func askAgent(p Prompter) (string, error) {
	names, labels, err := agentChoices()
	if err != nil {
		return "", err
	}
	idx, err := p.Select("AI agent:", labels, 0)
	if err != nil {
		return "", err
	}
	return string(names[idx]), nil
}

func askAgents(p Prompter) (string, error) {
	names, labels, err := agentChoices()
	if err != nil {
		return "", err
	}
	picked, err := p.MultiSelect("AI agents:", labels, nil, 1)
	if err != nil {
		return "", err
	}
	chosen := make([]string, len(picked))
	for i, idx := range picked {
		chosen[i] = string(names[idx])
	}
	return strings.Join(chosen, ","), nil
}

// projectSkillNames lists the skills in this project's agent folders, once each.
func projectSkillNames() ([]string, error) {
	rows, err := listInstalledSkills(projectScope())
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, r := range rows {
		if !seen[r.Name] {
			seen[r.Name] = true
			names = append(names, r.Name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func askLocalSkill(p Prompter, message string) (string, error) {
	names, err := projectSkillNames()
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", errors.New("no skills in this project's agent folders yet: create one first (hikma skill create <name>) or install one")
	}
	idx, err := p.Select(message, names, 0)
	if err != nil {
		return "", err
	}
	return names[idx], nil
}

// withSource adds the registry to a command that reads one, when none is configured.
func withSource(p Prompter, args ...string) func(Prompter) ([]string, error) {
	return func(p Prompter) ([]string, error) {
		out := append([]string{}, args...)
		if !registryConfigured() {
			r, err := askRegistry(p, "No registry is set. Registry to use (owner/repo):")
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	}
}

// registrySkills lists the skills a source offers. It is a variable so tests do
// not clone anything. Cancelling ctx stops the clone.
var registrySkills = func(ctx context.Context, repo string) ([]source.Skill, error) {
	src, co, cleanup, err := openSource(ctx, io.Discard, repo, "", "")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return discoverSkills(src, co)
}

// readRegistry lists a registry's skills behind a spinner. The work reports
// through a buffered channel, never through variables the caller shares, so a
// spinner that is cancelled and abandoned cannot race with the caller.
func readRegistry(p Prompter, label, repo string) ([]source.Skill, error) {
	type result struct {
		skills []source.Skill
		err    error
	}
	ch := make(chan result, 1)
	err := p.Spinner("Reading "+shortLabel(label)+"...", func(ctx context.Context) error {
		skills, err := registrySkills(ctx, repo)
		ch <- result{skills, err}
		return err
	})
	if err != nil {
		return nil, err
	}
	r := <-ch
	return r.skills, r.err
}

// registryLabel names the registry an install will read, for messages.
func registryLabel(repo string) string {
	if repo != "" {
		return repo
	}
	if e, err := config.Lookup(config.KeyRegistry); err == nil && e.Value != "" {
		return e.Value
	}
	return "the registry"
}

// shortLabel keeps a long registry name from wrapping a narrow terminal.
func shortLabel(s string) string {
	const max = 40
	if r := []rune(s); len(r) > max {
		return string(r[:max-3]) + "..."
	}
	return s
}

// installedFor reports whether name is already installed in this project for any
// of the agents the install would write to. Another agent's copy does not count.
func installedFor(targets []config.Selection, name string) bool {
	sc := projectScope()
	for _, t := range targets {
		if dirExists(sc.skillDir(t, name)) {
			return true
		}
	}
	return false
}

// planSkill is one skill on the install screen.
type planSkill struct{ Name, Description string }

// installPlan is what the install screen shows before anything is written.
type installPlan struct {
	Source string
	Skills []planSkill
	Agents []string
	Dirs   []string
}

// renderInstallPlan is the one-screen summary the user confirms.
func renderInstallPlan(p installPlan) string {
	noun := "skill"
	if len(p.Skills) != 1 {
		noun = "skills"
	}
	width := 0
	for _, s := range p.Skills {
		width = max(width, ui.Width(s.Name))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Will install %d %s from %s\n", len(p.Skills), noun, shortLabel(p.Source))
	for _, s := range p.Skills {
		line := "  " + s.Name
		if s.Description != "" {
			line = "  " + ui.Pad(s.Name, width) + "  " + truncate(s.Description, 50)
		}
		b.WriteString(line + "\n")
	}
	if len(p.Dirs) > 0 {
		fmt.Fprintf(&b, "Into %s (agent: %s)\n", strings.Join(p.Dirs, ", "), strings.Join(p.Agents, ", "))
	}
	return b.String()
}

// collectInstall is the whole install flow in the menu: pick the registry if none
// is set, read what it offers, let the user tick skills, show a plan, and confirm.
// It returns one complete command, so the equivalent command printed afterwards
// can be pasted into a script. A registry that cannot be read (a typo) asks for
// another one without asking for the agent again.
func collectInstall(p Prompter, out io.Writer) ([][]string, error) {
	askRepo := !registryConfigured()
	repo := ""
	var agentArgs []string
	agentFlag := ""
	agentAsked := false

	var skills []source.Skill
	var label string
	for {
		if askRepo {
			r, err := askRegistry(p, "No registry is set. Registry to use (owner/repo):")
			if err != nil {
				return nil, err
			}
			repo = r
		}
		if !agentAsked && !agentChosen() {
			a, err := askAgent(p)
			if err != nil {
				return nil, err
			}
			agentArgs, agentFlag, agentAsked = []string{"--agent", a}, a, true
		}
		label = registryLabel(repo)
		var err error
		skills, err = readRegistry(p, label, repo)
		if err == nil {
			break
		}
		if !askRepo || errors.Is(err, errCancelled) {
			return nil, err
		}
		fmt.Fprintf(out, "\nerror: %s\n\n", err)
	}

	targets, err := config.ResolveTargets(agentFlag)
	if err != nil {
		return nil, fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
	}
	if _, err := projectSkillNames(); err != nil {
		return nil, err
	}

	// A name from a registry is untrusted: one that starts with a dash would be
	// read as a flag, so it is left out of the menu.
	var fresh []source.Skill
	var already []string
	total := 0
	for _, s := range skills {
		if strings.HasPrefix(s.Name, "-") {
			continue
		}
		total++
		if installedFor(targets, s.Name) {
			already = append(already, s.Name)
			continue
		}
		fresh = append(fresh, s)
	}
	switch {
	case total == 0:
		return nil, fmt.Errorf("%s has no skills. Check the name, or set another registry: hikma config registry <owner/repo>", label)
	case len(fresh) == 0:
		return nil, fmt.Errorf("every skill in %s is already installed here. To refresh them: hikma skill update --all", label)
	}
	// Installing over an installed skill needs --force, which would replace local
	// edits, so the menu keeps those out of the picker and points at Update.
	if len(already) > 0 {
		fmt.Fprintf(out, "Already installed (to refresh them, use Update installed skills): %s\n\n", strings.Join(already, ", "))
	}

	labels := make([]string, len(fresh))
	for i, s := range fresh {
		labels[i] = s.Name
		if s.Description != "" {
			labels[i] += " - " + truncate(s.Description, 70)
		}
	}
	picked, err := p.MultiSelect("Skills to install:", labels, nil, 1)
	if err != nil {
		return nil, err
	}
	plan := installPlan{Source: label}
	names := make([]string, len(picked))
	for i, idx := range picked {
		names[i] = fresh[idx].Name
		plan.Skills = append(plan.Skills, planSkill{fresh[idx].Name, fresh[idx].Description})
	}
	for _, t := range targets {
		plan.Agents = append(plan.Agents, string(t.Agent))
		plan.Dirs = append(plan.Dirs, t.Layout.SkillsDir)
	}
	fmt.Fprintln(out)
	fmt.Fprint(out, renderInstallPlan(plan))
	ok, err := p.Confirm("Install these?", true)
	if err != nil {
		return nil, err
	}
	if !ok {
		fmt.Fprintf(out, "\n%s\n", ui.Dim(out, "Nothing was installed."))
		return nil, errCancelled
	}

	args := []string{"skill", "install"}
	if repo != "" {
		args = append(args, repo)
	}
	if len(names) == 1 {
		args = append(args, names[0])
	} else {
		for _, n := range names {
			args = append(args, "--skill="+n)
		}
	}
	return [][]string{append(args, agentArgs...)}, nil
}

func collectUpdate(p Prompter) ([]string, error) {
	names, err := projectSkillNames()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, errors.New("no skills are installed in this project yet: install one first")
	}
	idx, err := p.Select("Update which skill?", append([]string{"All installed skills"}, names...), 0)
	if err != nil {
		return nil, err
	}
	if idx == 0 {
		return []string{"skill", "update", "--all"}, nil
	}
	return []string{"skill", "update", names[idx-1]}, nil
}

func collectPublish(p Prompter) ([]string, error) {
	name, err := askLocalSkill(p, "Skill to publish:")
	if err != nil {
		return nil, err
	}
	args := []string{"skill", "push", name}
	if !registryConfigured() {
		r, err := askRegistry(p, "No registry is set. Registry to publish to (owner/repo):")
		if err != nil {
			return nil, err
		}
		args = append(args, "--registry", r)
	}
	return args, nil
}

func collectInit(p Prompter) ([]string, error) {
	agents, err := askAgents(p)
	if err != nil {
		return nil, err
	}
	args := []string{"init", "--agent", agents}
	if !registryConfigured() {
		r, err := p.Input("Registry for this project (owner/repo, or leave empty):", func(s string) error {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			return validRegistry(s)
		})
		if err != nil {
			return nil, err
		}
		if r = strings.TrimSpace(r); r != "" {
			args = append(args, "--registry", r)
		}
	}
	return args, nil
}

func collectChooseAgent(p Prompter) ([]string, error) {
	a, err := askAgent(p)
	if err != nil {
		return nil, err
	}
	return []string{"config", "agent", a}, nil
}

func collectSetRegistry(p Prompter) ([]string, error) {
	r, err := askRegistry(p, "Registry (owner/repo):")
	if err != nil {
		return nil, err
	}
	return []string{"config", "registry", r}, nil
}

var wizardEntries = []wizardEntry{
	{label: "Install skills from your registry", collect: collectInstall},
	{label: "Browse skills in your registry", collect: one(withSource(nil, "skill", "list"))},
	{label: "Update installed skills", collect: one(collectUpdate)},
	{label: "Verify installed skills", collect: one(fixed("skill", "verify"))},
	{label: "Create a new skill", collect: one(askName("Skill name:", "skill", "create"))},
	{label: "Publish a skill to your registry", collect: one(collectPublish)},
	{label: "Setup and settings", sub: setupEntries},
}

var setupEntries = []wizardEntry{
	{label: "Set up this project for your agent", collect: one(collectInit)},
	{label: "Choose your agent", collect: one(collectChooseAgent)},
	{label: "Set your registry", collect: one(collectSetRegistry)},
	{label: "Install shell completion", collect: one(fixed("completion"))},
	{label: "Check your setup", collect: one(fixed("doctor"))},
}

const (
	wizardQuit  = "Quit"
	wizardBack  = "Back to the menu"
	wizardRetry = "Retry the ones that failed"
)

// wizardContext is the one line under the banner: what the next task will use.
func wizardContext() string {
	registry := "not set"
	if e, err := config.Lookup(config.KeyRegistry); err == nil && e.Value != "" {
		registry = e.Value
	}
	agent := "not chosen"
	if agentChosen() {
		if e, err := config.Lookup(config.KeyAgent); err == nil {
			agent = e.Value
		}
	}
	n := 0
	if names, err := projectSkillNames(); err == nil {
		n = len(names)
	}
	return fmt.Sprintf("registry: %s | agent: %s | %d installed here", registry, agent, n)
}

func entryLabels(entries []wizardEntry, last string) []string {
	labels := make([]string, 0, len(entries)+1)
	for _, e := range entries {
		labels = append(labels, e.label)
		if e.sub != nil {
			labels[len(labels)-1] += " >"
		}
	}
	return append(labels, last)
}

// withInterrupt runs fn with a context that Ctrl-C cancels, so the work stops
// instead of ending hikma. The terminal sends the interrupt to the whole
// foreground group, so a child such as git stops too, and commands that take a
// context stop between steps. Once the first Ctrl-C has been taken, the default
// behaviour is back, so a second one ends hikma if the work will not stop. A
// prompt on screen has the terminal in raw mode and reads the key itself, so this
// never sees that case.
func withInterrupt(fn func(ctx context.Context) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()
	err := fn(ctx)
	if err != nil && ctx.Err() != nil {
		return errCancelled
	}
	return err
}

// retryArgs is args with the --skill flags replaced by the skills that failed, so
// "retry" asks for exactly those.
func retryArgs(args, failed []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--skill=") {
			out = append(out, a)
		}
	}
	for _, n := range failed {
		out = append(out, "--skill="+n)
	}
	return out
}

func printEquivalent(out io.Writer, heading string, args []string) {
	fmt.Fprintf(out, "\n%s\n  %s\n", ui.Dim(out, heading), ui.Cyan(out, equivalent(args)))
}

// execute runs each planned command, shows what happened, and returns the command
// that would retry the skills that failed, or nil.
func execute(out io.Writer, cmds [][]string, run func(ctx context.Context, args []string) error) []string {
	for _, args := range cmds {
		err := withInterrupt(func(ctx context.Context) error { return run(ctx, args) })
		switch {
		case errors.Is(err, errCancelled):
			fmt.Fprintf(out, "\n%s\n", ui.Dim(out, "Cancelled."))
			return nil
		case err != nil:
			fmt.Fprintf(out, "\nerror: %s\n", err)
			var f *installFailed
			if errors.As(err, &f) {
				// Some skills went in: say what ran, and how to redo only the rest.
				retry := retryArgs(args, f.Failed)
				printEquivalent(out, "Equivalent command:", args)
				printEquivalent(out, "Retry only the ones that failed:", retry)
				return retry
			}
			return nil
		}
		// The equivalent command comes last, and only for a task that worked.
		printEquivalent(out, "Equivalent command:", args)
	}
	return nil
}

// runWizard is a menu you stay in: it clears the screen, shows the banner and
// menu, runs the chosen command through run, prints the equivalent flag command,
// and returns to the menu until the user quits. Ctrl-C or Esc inside a task, and
// at "What next?", goes back to the menu; at the menu they leave. The cursor stays
// on the entry last used.
func runWizard(p Prompter, out io.Writer, run func(ctx context.Context, args []string) error) error {
	cursor := 0
	for first := true; ; first = false {
		ui.ClearScreen(out)
		// A screen reader hears the whole screen again on every draw, so the
		// banner is read once.
		if first || !accessible() {
			fmt.Fprint(out, ui.Banner(out))
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, ui.Dim(out, wizardContext()))
		fmt.Fprintln(out, ui.Dim(out, "Ctrl-C or Esc inside a task comes back here. At this menu they quit."))
		fmt.Fprintln(out)

		idx, err := p.Select("What do you want to do?", entryLabels(wizardEntries, wizardQuit), cursor)
		if err != nil {
			return quietCancel(err)
		}
		if idx < 0 || idx >= len(wizardEntries) {
			return nil
		}
		cursor = idx
		entry := wizardEntries[idx]
		if entry.sub != nil {
			sidx, err := p.Select(entry.label+":", entryLabels(entry.sub, wizardBack), 0)
			if errors.Is(err, errCancelled) || (err == nil && (sidx < 0 || sidx >= len(entry.sub))) {
				continue
			}
			if err != nil {
				return err
			}
			entry = entry.sub[sidx]
		}

		cmds, err := entry.collect(p, out)
		if errors.Is(err, errCancelled) {
			continue
		}
		var retry []string
		if err != nil {
			fmt.Fprintf(out, "\nerror: %s\n", err)
		} else {
			fmt.Fprintln(out)
			retry = execute(out, cmds, run)
		}

		// Hold the result on screen: the next loop clears it.
		for {
			opts := []string{wizardBack}
			if retry != nil {
				opts = append(opts, wizardRetry)
			}
			opts = append(opts, wizardQuit)
			next, err := p.Select("What next?", opts, 0)
			if errors.Is(err, errCancelled) {
				break // Ctrl-C or Esc is "back" everywhere but the menu
			}
			if err != nil || opts[next] == wizardQuit {
				return err
			}
			if opts[next] != wizardRetry {
				break
			}
			fmt.Fprintln(out)
			retry = execute(out, [][]string{retry}, run)
		}
	}
}

func quietCancel(err error) error {
	if errors.Is(err, errCancelled) {
		return nil
	}
	return err
}

// equivalent renders args as the hikma command a user could type, quoting
// anything the shell would split.
func equivalent(args []string) string {
	parts := []string{"hikma"}
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\"'$`\\") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && (isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()))
}

// rootRun is what bare `hikma` does: open the menu on a terminal, print help
// everywhere else so scripts and CI never wait for input.
func rootRun(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	if !isInteractive() || !isTerminalWriter(out) {
		return cmd.Help()
	}
	return runWizard(prompter, out, func(ctx context.Context, args []string) error {
		// Root() rebinds the global flags to their defaults, so carry the
		// values the user passed over to the command the menu runs.
		keepNoInput, keepNoColor := noInput, ui.NoColor
		inner := Root()
		noInput, ui.NoColor = keepNoInput, keepNoColor
		inner.SetArgs(args)
		return inner.ExecuteContext(ctx)
	})
}
