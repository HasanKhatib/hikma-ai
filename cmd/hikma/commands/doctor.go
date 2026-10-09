package commands

import (
	"fmt"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/preflight"
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
		fmt.Printf("  ok    %s\n", name)
	}
	for _, f := range result.Failures {
		label := "FAIL"
		if strings.HasPrefix(f.Name, "gh ") {
			label = "warn" // gh is only needed for 'skill push' and some private repos
		} else {
			failed++
		}
		fmt.Printf("  %s  %s\n", label, f.Name)
		fmt.Printf("        %s\n", f.Err)
		for _, fix := range f.Fix {
			fmt.Printf("        fix: %s\n", fix)
		}
	}

	if targets, err := config.ResolveTargets(""); err != nil {
		failed++
		fmt.Printf("  FAIL  agent config\n        %s\n        fix: run 'hikma config set agent <agent>'\n", err)
	} else {
		for _, t := range targets {
			fmt.Printf("  ok    agent: %s (skills in %s)\n", t.Agent, t.Layout.SkillsDir)
		}
	}

	if src, origin, err := resolveSource(""); err != nil {
		// A registry is optional: skills can always be installed from an explicit owner/repo.
		fmt.Printf("  warn  registry not configured\n        fix: run 'hikma config set registry <owner/repo>' (needed for bare skill names and push)\n")
	} else {
		fmt.Printf("  ok    registry: %s (%s)\n", src.Display, origin)
	}

	if gh := ghManagedUntracked(); len(gh) > 0 {
		fmt.Printf("  warn  skills installed by gh skill and not tracked by hikma: %s\n        fix: run 'hikma skill adopt <name>' so update, sync, and remove manage them\n", adoptHint(gh))
	}

	if failed > 0 {
		fmt.Printf("\n%d check(s) failed. Run 'hikma doctor' again after applying fixes.\n", failed)
		return fmt.Errorf("one or more checks failed")
	}
	fmt.Println("\nAll required checks passed.")
	return nil
}
