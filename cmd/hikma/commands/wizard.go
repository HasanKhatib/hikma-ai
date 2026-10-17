package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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

var wizardEntries = []wizardEntry{
	{"Install a skill from your registry", fixed("skill", "install")},
	{"Browse skills in your registry", fixed("skill", "list")},
	{"Update installed skills", fixed("skill", "update")},
	{"Verify installed skills", fixed("skill", "verify")},
	{"Create a new skill", askName("Skill name:", "skill", "create")},
	{"Publish a skill to your registry", askName("Skill to publish:", "skill", "push")},
	{"Set up this project for your agent", fixed("init")},
	{"Choose your agent", fixed("config", "agent")},
	{"Set your registry", askName("Registry (owner/repo):", "config", "registry")},
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

	idx, err := p.Select("What do you want to do?", labels)
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
	runErr := quietCancel(cancelled(run(args)))

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
	return runWizard(surveyPrompter{}, out, func(args []string) error {
		inner := Root()
		inner.SetArgs(args)
		return inner.Execute()
	})
}
