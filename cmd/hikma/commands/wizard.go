package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// A wizard entry is a front end for a flag command: collect inputs, plan the
// argument list, then run the same command code a script would. It holds no
// logic of its own, so the flags stay the API.
type wizardEntry struct {
	label   string
	collect func(p Prompter) ([]string, error)
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
	picked, err := p.MultiSelect("AI agents (space to select):", labels, nil, 1)
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

func collectInstall(p Prompter) ([]string, error) {
	args, err := withSource(p, "skill", "install")(p)
	if err != nil {
		return nil, err
	}
	if !agentChosen() {
		a, err := askAgent(p)
		if err != nil {
			return nil, err
		}
		args = append(args, "--agent", a)
	}
	return args, nil
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
	{"Install a skill from your registry", collectInstall},
	{"Browse skills in your registry", withSource(nil, "skill", "list")},
	{"Update installed skills", collectUpdate},
	{"Verify installed skills", fixed("skill", "verify")},
	{"Create a new skill", askName("Skill name:", "skill", "create")},
	{"Publish a skill to your registry", collectPublish},
	{"Set up this project for your agent", collectInit},
	{"Choose your agent", collectChooseAgent},
	{"Set your registry", collectSetRegistry},
	{"Install shell completion", fixed("completion")},
	{"Check your setup", fixed("doctor")},
}

const wizardQuit = "Quit"

// runWizard shows the banner and menu, runs the chosen command through run, and
// prints the equivalent flag command so the next run can skip the menu.
func runWizard(p Prompter, out io.Writer, run func(args []string) error) error {
	fmt.Fprint(out, ui.Banner(out))
	fmt.Fprintln(out)

	labels := make([]string, 0, len(wizardEntries)+1)
	for _, e := range wizardEntries {
		labels = append(labels, e.label)
	}
	labels = append(labels, wizardQuit)

	idx, err := p.Select("What do you want to do?", labels, 0)
	if err != nil {
		return quietCancel(err)
	}
	if idx < 0 || idx >= len(wizardEntries) {
		return nil
	}
	args, err := wizardEntries[idx].collect(p)
	if err != nil {
		return quietCancel(err)
	}
	fmt.Fprintln(out)

	// Ctrl-C inside the command's own prompt is a cancel, like Ctrl-C in ours.
	runErr := quietCancel(run(args))

	fmt.Fprintf(out, "\n%s\n  %s\n", ui.Dim(out, "Equivalent command:"), ui.Cyan(out, equivalent(args)))
	return runErr
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
	return runWizard(prompter, out, func(args []string) error {
		// Root() rebinds the global flags to their defaults, so carry the
		// values the user passed over to the command the menu runs.
		keepNoInput, keepNoColor := noInput, ui.NoColor
		inner := Root()
		noInput, ui.NoColor = keepNoInput, keepNoColor
		inner.SetArgs(args)
		return inner.Execute()
	})
}
