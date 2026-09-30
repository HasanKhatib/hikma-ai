package commands

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage hikma user configuration",
		Long:  `Manage hikma user-level configuration stored at os.UserConfigDir()/hikma/config.json.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown config command %q - use 'hikma config agent' or 'hikma config registry'", args[0])
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newConfigAgentCmd(), newConfigRegistryCmd())
	return cmd
}

func newConfigRegistryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "registry [owner/repo|url]",
		Short: "Get or set the active skill registry",
		Args:  cobra.MaximumNArgs(1),
		Long: `Get or set the active skill registry.

The registry must be a GitHub repository that contains reusable skills. Accepted
formats include owner/repo, https://github.com/owner/repo.git, and

Examples:
  hikma config registry hasankhatib/ai
  hikma config registry https://github.com/hasankhatib/ai.git`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return runConfigRegistrySet(cmd, args[0])
			}
			c, err := config.Load()
			if err != nil {
				return err
			}
			if c.Registry == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No registry configured.")
				fmt.Fprintln(cmd.OutOrStdout(), "Set one with: hikma config registry hasankhatib/ai")
				return nil
			}
			r, err := config.ParseRegistry(c.Registry)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Registry: %s\n", r.FullName())
			fmt.Fprintf(cmd.OutOrStdout(), "Clone URL: %s\n", r.CloneURL())
			return nil
		},
	}
}

func runConfigRegistrySet(cmd *cobra.Command, val string) error {
	c, err := config.ConfigForRegistry(val)
	if err != nil {
		return err
	}
	if err := config.Save(c); err != nil {
		return err
	}
	r, _ := config.ParseRegistry(val)
	fmt.Fprintf(cmd.OutOrStdout(), "Registry set to: %s\n", r.FullName())
	return nil
}

func newConfigAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "agent [value]",
		Short:     "Get or set the active AI agent (copilot, codex, opencode, claude)",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"copilot", "codex", "opencode", "claude"},
		Long: `Get or set the active AI agent.

The agent controls which paths hikma uses for skills and scaffold output:

  copilot    .agents/skills/   AGENTS.md only          GitHub Copilot
  codex      .agents/skills/   AGENTS.md only          Codex CLI
  opencode   .agents/skills/   AGENTS.md only          OpenCode
  claude     .claude/skills/   AGENTS.md + CLAUDE.md   Claude Code

The active agent can also be overridden per-command with --agent,
or set for CI via the HIKMA_AGENT environment variable.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return runConfigAgentSet(cmd, args[0])
			}
			return runConfigAgentInteractive(cmd)
		},
	}
}

// runConfigAgentSet handles: hikma config agent <value>
func runConfigAgentSet(cmd *cobra.Command, val string) error {
	a := config.Agent(val)
	c, err := config.ConfigForAgent(a)
	if err != nil {
		return fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
	}
	if err := config.Save(c); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Active agent set to: %s\n", a)
	fmt.Fprintf(cmd.OutOrStdout(), "Skill path: %s\n", config.SkillBasePath(config.ProfileForAgent(a)))
	fmt.Fprintf(cmd.OutOrStdout(), "Scaffold output: %s\n", scaffoldOutput(config.ProfileForAgent(a)))
	return nil
}

// runConfigAgentInteractive handles: hikma config agent (no argument)
func runConfigAgentInteractive(cmd *cobra.Command) error {
	selection, err := config.ResolveSelection("")
	if err != nil {
		selection = config.Selection{Agent: config.AgentCopilot, Profile: config.ProfileDefault}
	}

	labelFor := func(a config.Agent) string {
		return fmt.Sprintf("%s - %s", a, config.AgentDescriptions[a])
	}

	options := []string{
		labelFor(config.AgentCopilot),
		labelFor(config.AgentCodex),
		labelFor(config.AgentOpenCode),
		labelFor(config.AgentClaude),
	}

	labelToAgent := map[string]config.Agent{
		labelFor(config.AgentCopilot):  config.AgentCopilot,
		labelFor(config.AgentCodex):    config.AgentCodex,
		labelFor(config.AgentOpenCode): config.AgentOpenCode,
		labelFor(config.AgentClaude):   config.AgentClaude,
	}

	defaultLabel := labelFor(config.AgentCopilot)
	for label, a := range labelToAgent {
		if a == selection.Agent {
			defaultLabel = label
			break
		}
	}

	var selected string
	prompt := &survey.Select{
		Message: "AI agent:",
		Options: options,
		Default: defaultLabel,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	chosen := labelToAgent[selected]
	c, err := config.ConfigForAgent(chosen)
	if err != nil {
		return err
	}
	if err := config.Save(c); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Active agent set to: %s\n", chosen)
	fmt.Fprintf(cmd.OutOrStdout(), "Skill path: %s\n", config.SkillBasePath(config.ProfileForAgent(chosen)))
	fmt.Fprintf(cmd.OutOrStdout(), "Scaffold output: %s\n", scaffoldOutput(config.ProfileForAgent(chosen)))
	return nil
}
