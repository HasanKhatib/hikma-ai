package commands

import (
	"fmt"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage hikma user configuration",
		Long: `Manage hikma configuration.

Values are resolved in this order: command flag, HIKMA_* environment variable,
project config (.hikma/config.json, commit it to share with your team), then
user config (os.UserConfigDir()/hikma/config.json).

Keys:
  agent     your default agent: copilot, codex, opencode, or claude
  agents    agents a project sets up, comma-separated (project config)
  registry  where bare skill names install from and where push publishes
  naming    skill name rule for create and push: loose (default) or kebab-case`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown config command %q - use 'hikma config list', 'get', 'set', 'unset' or 'path'", args[0])
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newConfigListCmd(), newConfigGetCmd(), newConfigSetCmd(), newConfigUnsetCmd(), newConfigPathCmd(),
		newConfigAgentCmd(), newConfigRegistryCmd(),
	)
	return cmd
}

func newConfigRegistryCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "registry [owner/repo|url]",
		Hidden: true, // alias for 'config get|set registry'

		Short: "Get or set the active skill registry",
		Args:  cobra.MaximumNArgs(1),
		Long: `Get or set the active skill registry.

The registry is where bare skill names are installed from and where 'skill push'
publishes. Accepted formats: owner/repo, a git URL, or a local path (push needs
a GitHub owner/repo).

Examples:
  hikma config registry owner/registry
  hikma config registry https://github.com/example/registry.git`,
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
				fmt.Fprintln(cmd.OutOrStdout(), "Set one with: hikma config registry owner/registry")
				return nil
			}
			src, err := source.Parse(c.Registry)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Registry: %s\n", src.Display)
			if src.CloneURL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Clone URL: %s\n", src.CloneURL)
			}
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
	src, _ := source.Parse(val)
	fmt.Fprintf(cmd.OutOrStdout(), "Registry set to: %s\n", src.Display)
	return nil
}

func newConfigAgentCmd() *cobra.Command {
	return &cobra.Command{
		Hidden:    true, // alias for 'config get|set agent'
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
	fmt.Fprintf(cmd.OutOrStdout(), "Skill path: %s\n", config.LayoutFor(a).SkillsDir)
	fmt.Fprintf(cmd.OutOrStdout(), "Scaffold output: %s\n", strings.Join(config.LayoutFor(a).InstructionFiles, ", "))
	return nil
}

// runConfigAgentInteractive handles: hikma config agent (no argument)
func runConfigAgentInteractive(cmd *cobra.Command) error {
	selection, err := config.ResolveSelection("")
	if err != nil {
		selection = config.SelectionFor(config.AgentCopilot)
	}

	labelFor := func(a config.Agent) string {
		return fmt.Sprintf("%s - %s", a, config.LayoutFor(a).Description)
	}
	var options []string
	labelToAgent := map[string]config.Agent{}
	defaultLabel := labelFor(config.AgentCopilot)
	for _, a := range config.ValidAgents {
		options = append(options, labelFor(a))
		labelToAgent[labelFor(a)] = a
		if a == selection.Agent {
			defaultLabel = labelFor(a)
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
	fmt.Fprintf(cmd.OutOrStdout(), "Skill path: %s\n", config.LayoutFor(chosen).SkillsDir)
	fmt.Fprintf(cmd.OutOrStdout(), "Scaffold output: %s\n", strings.Join(config.LayoutFor(chosen).InstructionFiles, ", "))
	return nil
}

func scopeFromFlag(project bool) config.Scope {
	if project {
		return config.ScopeProject
	}
	return config.ScopeUser
}

func newConfigListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show every config value and where it comes from",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := config.List()
			if err != nil {
				return err
			}
			if asJSON {
				type row struct {
					Key    string `json:"key"`
					Value  string `json:"value"`
					Source string `json:"source"`
				}
				rows := make([]row, len(entries))
				for i, e := range entries {
					rows[i] = row{e.Key, e.Value, describeSource(e)}
				}
				return printJSON(rows)
			}
			for _, e := range entries {
				v := e.Value
				if v == "" {
					v = "(not set)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-9s %s  [%s]\n", e.Key, v, describeSource(e))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func describeSource(e config.Entry) string {
	if e.Source == config.SourceEnv {
		return "env " + config.EnvVar(e.Key)
	}
	return string(e.Source)
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "get <key>",
		Short:     "Print the effective value of a config key",
		Args:      cobra.ExactArgs(1),
		ValidArgs: config.Keys,
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := config.Lookup(args[0])
			if err != nil {
				return err
			}
			if e.Value == "" {
				return fmt.Errorf("%s is not set", e.Key)
			}
			fmt.Fprintln(cmd.OutOrStdout(), e.Value)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	var project bool
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value (user config, or project config with --project)",
		Args:  cobra.ExactArgs(2),
		Example: `  hikma config set registry owner/registry
  hikma config set agent claude --project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := scopeFromFlag(project)
			if err := config.Set(scope, args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s (%s)\n", args[0], args[1], scope)
			return nil
		},
	}
	cmd.Flags().BoolVar(&project, "project", false, "write to .hikma/config.json in this repository")
	return cmd
}

func newConfigUnsetCmd() *cobra.Command {
	var project bool
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a config value from user config, or project config with --project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := scopeFromFlag(project)
			if err := config.Unset(scope, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unset %s (%s)\n", args[0], scope)
			return nil
		},
	}
	cmd.Flags().BoolVar(&project, "project", false, "remove from .hikma/config.json in this repository")
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the user and project config file locations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := config.ConfigFilePath()
			if err != nil {
				return err
			}
			project, err := config.ProjectConfigPath()
			if err != nil {
				return err
			}
			if project == "" {
				project = "(none found)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "user:    %s\nproject: %s\n", user, project)
			return nil
		},
	}
}
