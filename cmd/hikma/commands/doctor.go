package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/preflight"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that all required tools are available and configured",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	result := preflight.Check()
	failed := 0

	for _, name := range result.Passed {
		fmt.Printf("  %s    %s\n", ui.Green(os.Stdout, "ok"), name)
	}
	for _, f := range result.Failures {
		label := ui.Red(os.Stdout, "FAIL")
		if strings.HasPrefix(f.Name, "gh ") {
			label = ui.Yellow(os.Stdout, "warn") // gh is only needed for 'skill push' and some private repos
		} else {
			failed++
		}
		fmt.Printf("  %s  %s\n", label, f.Name)
		fmt.Printf("        %s\n", f.Err)
		for _, fix := range f.Fix {
			fmt.Printf("        %s\n", ui.Dim(os.Stdout, "fix: "+fix))
		}
	}

	if targets, err := config.ResolveTargets(""); err != nil {
		failed++
		fmt.Printf("  %s  agent config\n        %s\n        %s\n", ui.Red(os.Stdout, "FAIL"), err, ui.Dim(os.Stdout, "fix: run 'hikma config set agent <agent>'"))
	} else {
		for _, t := range targets {
			fmt.Printf("  %s    agent: %s (skills in %s)\n", ui.Green(os.Stdout, "ok"), t.Agent, t.Layout.SkillsDir)
		}
	}

	if src, origin, err := resolveSource(""); err != nil {
		// A registry is optional: skills can always be installed from an explicit owner/repo.
		fmt.Printf("  %s  registry not configured\n        %s\n", ui.Yellow(os.Stdout, "warn"), ui.Dim(os.Stdout, "fix: run 'hikma config set registry <owner/repo>' (needed for bare skill names and push)"))
	} else {
		fmt.Printf("  %s    registry: %s (%s)\n", ui.Green(os.Stdout, "ok"), src.Display, origin)
		if line, ok := describeRegistry(src.Raw); ok {
			fmt.Printf("        %s\n", ui.Dim(os.Stdout, line))
		}
	}

	if gh := ghManagedUntracked(projectScope()); len(gh) > 0 {
		fmt.Printf("  %s  skills installed by gh skill and not tracked by hikma: %s\n        %s\n", ui.Yellow(os.Stdout, "warn"), adoptHint(gh), ui.Dim(os.Stdout, "fix: run 'hikma skill adopt <name>' so update, sync, and remove manage them"))
	}

	if sc, err := globalScope(); err == nil && !sameDir(".", sc.root) {
		var names []string
		for n := range installedNames(projectScope()) {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, w := range projectScope().shadowWarnings(names) {
			fmt.Printf("  %s  %s\n", ui.Yellow(os.Stdout, "warn"), strings.TrimPrefix(w, "warning: "))
		}
	}

	if failed > 0 {
		fmt.Printf("\n%d check(s) failed. Run 'hikma doctor' again after applying fixes.\n", failed)
		return fmt.Errorf("one or more checks failed")
	}
	fmt.Println("\nAll required checks passed.")
	return nil
}
