package commands

import (
	"fmt"

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

	for _, name := range result.Passed {
		fmt.Printf("  ok    %s\n", name)
	}

	for _, f := range result.Failures {
		fmt.Printf("  FAIL  %s\n", f.Name)
		fmt.Printf("        %s\n", f.Err)
		for _, fix := range f.Fix {
			fmt.Printf("        fix: %s\n", fix)
		}
	}

	// Agent/path check.
	selection, err := config.ResolveSelection("")
	agentConfigOK := err == nil
	if err != nil {
		fmt.Printf("  FAIL  agent config\n")
		fmt.Printf("        %s\n", err)
		fmt.Printf("        fix: run 'hikma config agent <agent>'\n")
	} else {
		fmt.Printf("  ok    agent: %s\n", selection.Agent)
		fmt.Printf("  ok    skill path: %s\n", config.SkillBasePath(selection.Profile))
	}
	registryConfigOK := true
	activeRegistry, err := config.ResolveRegistry("")
	if err != nil {
		registryConfigOK = false
		fmt.Printf("  FAIL  registry config\n")
		fmt.Printf("        %s\n", err)
		fmt.Printf("        fix: run 'hikma config registry hasankhatib/ai'\n")
	} else {
		fmt.Printf("  ok    registry: %s\n", activeRegistry.FullName())
	}

	if !result.OK() || !agentConfigOK || !registryConfigOK {
		total := len(result.Passed) + len(result.Failures)
		passed := len(result.Passed)
		if agentConfigOK {
			total += 2
			passed += 2
		} else {
			total++
		}
		if registryConfigOK {
			total++
			passed++
		} else {
			total++
		}
		fmt.Printf("\n%d of %d checks passed. Run 'hikma doctor' again after applying fixes.\n",
			passed, total)
		return fmt.Errorf("one or more checks failed")
	}

	fmt.Println("\nAll checks passed.")
	return nil
}
