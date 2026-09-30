package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
	"github.com/spf13/cobra"
)

func newScaffoldCmd() *cobra.Command {
	var path string
	var force bool
	var dryRun bool
	var flagProjectName, flagOwner, flagTechnology string
	var flagAgent, flagRegistry string

	cmd := &cobra.Command{
		Use:     "scaffold",
		Aliases: []string{"init"},
		Short:   "Scaffold AI agent configuration into the current repo",
		Args:    cobra.NoArgs,
		Long: `Scaffold writes AI agent configuration files into a repository.

Flags control what is written:
  --agent <value>         AI agent: copilot, codex, opencode, claude (default: from hikma config)
  --registry <owner/repo> skill registry to save in user config
  --path <dir>            target directory (default: current working directory)
  --force                 overwrite existing files
  --dry-run               print what would be written without writing anything
  --project-name <name>   project name (skips prompt)
  --owner <owner>         owner / team name (skips prompt)
  --technology <tech>     technology stack (skips prompt)

Agent controls which files are written and where skills are installed:
  copilot    AGENTS.md only          skills -> .agents/skills/
  codex      AGENTS.md only          skills -> .agents/skills/
  opencode   AGENTS.md only          skills -> .agents/skills/
  claude     AGENTS.md + CLAUDE.md   skills -> .claude/skills/

Set your default agent once with: hikma config agent
Set your registry once with: hikma config registry hasankhatib/ai

When --project-name, --owner, and --technology are all provided, scaffold runs
non-interactively without requiring a TTY.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve target path.
			if path == "" {
				var err error
				path, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine working directory: %w", err)
				}
			} else {
				abs, err := filepath.Abs(path)
				if err != nil {
					return fmt.Errorf("invalid path: %w", err)
				}
				path = abs
			}

			// Non-interactive mode: all three value flags provided - no prompts needed.
			nonInteractive := flagProjectName != "" && flagOwner != "" && flagTechnology != ""

			// Guard: stdin must be an interactive terminal before we attempt prompts,
			// unless all required values are provided via flags.
			if !nonInteractive && !dryRun {
				fi, err := os.Stdin.Stat()
				if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
					return fmt.Errorf(
						"hikma scaffold requires an interactive terminal\n" +
							"  Prompts cannot be answered when stdin is not a TTY\n" +
							"  To use non-interactively, provide: --project-name, --owner, --technology",
					)
				}
			}

			// Resolve agent/layout - print as first output line before any prompts.
			isAgentOverride := flagAgent != ""
			selection, err := config.ResolveSelection(flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w\nrun 'hikma doctor' to check your environment", err)
			}
			printActiveSelection(selection, isAgentOverride)
			fmt.Printf("Scaffold output: %s\n", scaffoldOutput(selection.Profile))

			// Check for a git repository by walking up the directory tree.
			_, isGitRepo := scaffold.FindGitRoot(path)

			var reader *bufio.Reader
			if !nonInteractive && !dryRun {
				reader = bufio.NewReader(os.Stdin)

				if !isGitRepo {
					fmt.Fprintf(os.Stderr, "warning: this directory is not a git repository\n")
					fmt.Fprintf(os.Stderr, "  Files written here will not be version-controlled.\n")
					ans, err := prompt(reader, "Proceed anyway? [y/N]: ", "n")
					if err != nil {
						return err
					}
					if strings.ToLower(strings.TrimSpace(ans)) != "y" {
						fmt.Println("Aborted.")
						return nil
					}
				} else if gr, _ := scaffold.FindGitRoot(path); gr != path {
					fmt.Printf("note: git root is %s - scaffolding into %s\n", gr, path)
				}
			}

			projectName, owner, technology, err := resolveScaffoldValues(
				reader, path, flagProjectName, flagOwner, flagTechnology, dryRun,
			)
			if err != nil {
				return err
			}

			if !nonInteractive && !dryRun {
				// 4th prompt: AI agent (skipped when --agent is set).
				if !isAgentOverride {
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
					agentSelect := &survey.Select{
						Message: "AI agent:",
						Options: options,
						Default: defaultLabel,
					}
					if err := survey.AskOne(agentSelect, &selected); err != nil {
						return err
					}
					chosen := labelToAgent[selected]
					selection = config.Selection{Agent: chosen, Profile: config.ProfileForAgent(chosen)}
				}
			}

			opts := scaffold.Options{
				ProjectName: projectName,
				Owner:       owner,
				Technology:  technology,
				Profile:     selection.Profile,
				Path:        path,
				Force:       force,
				DryRun:      dryRun,
			}

			if dryRun {
				fmt.Printf("Dry run - nothing will be written.\n\n")
			} else {
				fmt.Printf("\nScaffolding %s...\n", projectName)
			}

			sr, err := scaffold.Scaffold(opts)
			if err != nil {
				return err
			}

			if dryRun {
				for _, p := range sr.Created {
					fmt.Printf("  would write %s\n", relOrFull(path, p))
				}
				return nil
			}

			for _, p := range sr.Created {
				fmt.Printf("  Created %s\n", relOrFull(path, p))
			}
			for _, p := range sr.Overwritten {
				fmt.Printf("  Overwrote %s\n", relOrFull(path, p))
			}
			for _, p := range sr.Skipped {
				rel := relOrFull(path, p)
				fmt.Printf("  Skipped %-30s (already exists - use --force to overwrite)\n", rel)
			}

			if len(sr.Created) > 0 || len(sr.Skipped) > 0 {
				if flagRegistry != "" {
					if err := runConfigRegistrySet(cmd, flagRegistry); err != nil {
						return err
					}
				} else {
					fmt.Printf("\nRun 'hikma config registry hasankhatib/ai' to choose a registry.\n")
				}
				fmt.Printf("Run 'hikma skill list' to browse configured registry skills.\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "target directory (default: current working directory)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be written without writing anything")
	cmd.Flags().StringVar(&flagProjectName, "project-name", "", "project name (skips interactive prompt)")
	cmd.Flags().StringVar(&flagOwner, "owner", "", "owner / team name (skips interactive prompt)")
	cmd.Flags().StringVar(&flagTechnology, "technology", "", "technology stack (skips interactive prompt)")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "AI agent (copilot, codex, opencode, claude)")
	cmd.Flags().StringVar(&flagRegistry, "registry", "", "registry to save for skill commands (owner/repo or GitHub URL)")

	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func resolveScaffoldValues(reader *bufio.Reader, path, projectName, owner, technology string, dryRun bool) (string, string, string, error) {
	if dryRun {
		if projectName == "" {
			projectName = filepath.Base(path)
		}
		return projectName, owner, technology, nil
	}

	var err error
	if projectName == "" {
		defaultName := filepath.Base(path)
		projectName, err = prompt(reader, fmt.Sprintf("Project name [%s]: ", defaultName), defaultName)
		if err != nil {
			return "", "", "", err
		}
	}
	if owner == "" {
		owner, err = prompt(reader, "Owner / team name: ", "")
		if err != nil {
			return "", "", "", err
		}
	}
	if technology == "" {
		technology, err = prompt(reader, "Technology (language and framework): ", "")
		if err != nil {
			return "", "", "", err
		}
	}
	return projectName, owner, technology, nil
}

// prompt prints a prompt and reads a line from r.
// If the user presses enter with no input, defaultVal is returned.
func prompt(r *bufio.Reader, message, defaultVal string) (string, error) {
	fmt.Print(message)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// relOrFull returns p relative to base if possible, otherwise p as-is.
func relOrFull(base, p string) string {
	rel, err := filepath.Rel(base, strings.TrimSuffix(p, "/"))
	if err != nil {
		return p
	}
	if strings.HasSuffix(p, "/") {
		return rel + "/"
	}
	return rel
}
