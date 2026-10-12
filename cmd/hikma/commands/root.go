// Package commands contains all Cobra subcommands for the hikma CLI.
package commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

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
Set a registry with: hikma config registry owner/registry`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newConfigCmd(),
		newDoctorCmd(),
		newScaffoldCmd(),
		newSkillCmd(),
		newRegistryCmd(),
		newSyncCmd(),
		newUpdateCmd(),
		newCompletionCmd(),
	)

	friendlyErrors(cmd)
	return cmd
}

func newCompletionCmd() *cobra.Command {
	shells := []string{"bash", "zsh", "fish", "powershell"}
	return &cobra.Command{
		Annotations: map[string]string{ownArgErrors: "true"},
		Use:         "completion [bash|zsh|fish|powershell]",
		Short:       "Generate shell completion script",
		Long: `Generate shell completion script for hikma.

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
				return fmt.Errorf("which shell? Run one of: hikma completion %s\nSee 'hikma completion --help' for how to install it", strings.Join(shells, "|"))
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
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(out)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			case "fish":
				return cmd.Root().GenFishCompletion(out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(out)
			}
			return nil
		},
	}
}
