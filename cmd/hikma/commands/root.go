// Package commands contains all Cobra subcommands for the hikma CLI.
package commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
)

// version is set at build time by GoReleaser via ldflags:
//
//	-X github.com/hasankhatib/hikma-ai/cmd/hikma/commands.version={{.Version}}
//
// Local builds (go build, go run) will show "dev".
var version = "dev"

// Root returns the root Cobra command.
func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hikma",
		Short:   "Skills follow you: publish to your registry, install into any agent",
		Version: version,
		Long: `hikma moves your agent skills around. Define your source once, publish a skill
from any project to a registry repo you own, then install, update, and verify
it in whichever agent you use: Claude, Codex, Copilot, OpenCode, or a custom
layout you define. One CLI, plain Git.

Install from any repo with: hikma skill install owner/repo my-skill
Set a registry with: hikma config registry owner/registry

Run hikma with no arguments in a terminal for a menu that walks you through
the common tasks and prints the command it ran.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          rootRun,
	}

	cmd.PersistentFlags().BoolVar(&noInput, "no-input", false, "never prompt; fail and name the flag to pass instead")
	cmd.PersistentFlags().BoolVar(&ui.NoColor, "no-color", false, "turn off color (same as setting NO_COLOR)")

	cmd.AddCommand(
		newConfigCmd(),
		newDoctorCmd(),
		newScaffoldCmd(),
		newSkillCmd(),
		newRegistryCmd(),
		newSyncCmd(),
		newUpgradeCmd(),
		newCompletionCmd(),
	)

	friendlyErrors(cmd)
	return cmd
}

func newCompletionCmd() *cobra.Command {
	shells := []string{"bash", "zsh", "fish", "powershell"}
	var install, assumeYes bool
	cmd := &cobra.Command{
		Annotations: map[string]string{ownArgErrors: "true"},
		Use:         "completion [bash|zsh|fish|powershell]",
		Short:       "Install or print shell completion",
		Long: `Install shell completion for hikma, or print the script.

With no shell, hikma detects yours, shows exactly what it would change, and
asks before installing (--yes skips the question). With a shell name it prints
the script, or installs it with --install.

Bash:
  source <(hikma completion bash)
  # To persist across sessions:
  hikma completion bash > /etc/bash_completion.d/hikma

Zsh (add to ~/.zshrc, after compinit):
  source <(hikma completion zsh)

Fish:
  hikma completion fish > ~/.config/fish/completions/hikma.fish

PowerShell:
  hikma completion powershell | Out-String | Invoke-Expression`,
		ValidArgs: shells,
		Args: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				if install || assumeYes || isInteractive() {
					return nil
				}
				return fmt.Errorf("which shell? Run one of: hikma completion %s\nOr install it for your shell: hikma completion --install\nSee 'hikma completion --help' for details", strings.Join(shells, "|"))
			case len(args) > 1:
				return fmt.Errorf("completion takes one shell name, got %d: %s", len(args), strings.Join(args, " "))
			case !slices.Contains(shells, args[0]):
				msg := fmt.Sprintf("unknown shell %q. Supported: %s", args[0], strings.Join(shells, ", "))
				if s := closest(args[0], shells); s != "" {
					msg += fmt.Sprintf(". Did you mean: hikma completion %s?", s)
				}
				return errors.New(msg)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				shell, err := chooseShell(shells)
				if err != nil {
					return err
				}
				return installCompletion(cmd, shell, assumeYes)
			}
			if install {
				return installCompletion(cmd, args[0], assumeYes)
			}
			return generateCompletion(cmd, args[0], cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install completion for the shell instead of printing it")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "skip the confirmation question")
	return cmd
}
