// Package commands contains all Cobra subcommands for the hikma CLI.
package commands

import (
	"os"

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
		Short:   "Scaffold AI agent configurations and manage skills from your registry",
		Version: version,
		Long: `hikma helps teams scaffold AI agent configuration files and install reusable
skills from a user-configured registry repository.

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

	return cmd
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for hikma.

Bash:
  source <(hikma completion bash)
  # To persist across sessions:
  hikma completion bash > /etc/bash_completion.d/hikma

Zsh:
  hikma completion zsh > "${fpath[1]}/_hikma"
  # Then restart your shell or run:
  autoload -U compinit && compinit

Fish:
  hikma completion fish > ~/.config/fish/completions/hikma.fish

PowerShell:
  hikma completion powershell | Out-String | Invoke-Expression`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}
