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
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

func newScaffoldCmd() *cobra.Command {
	var path string
	var force bool
	var dryRun bool
	var flagProjectName, flagOwner, flagTechnology string
	var flagAgent, flagRegistry string
	var flagSkills []string

	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"scaffold"},
		Short:   "Set up AI agent instructions and skills in the current repo",
		Args:    cobra.NoArgs,
		Long: `Set up a repository for one or more AI agents.

init writes each selected agent's instruction files, records the agents (and an
optional registry) in .hikma/config.json so teammates share them, and can install
skills into every selected agent's skills folder.

Flags:
  --agent <list>          agents, comma-separated: copilot, codex, opencode, claude
  --registry <source>     project registry: owner/repo, git URL, or local path
  --skill <name>          install this skill from the registry (repeatable)
  --path <dir>            target directory (default: current working directory)
  --force                 overwrite existing files
  --dry-run               print what would happen without changing anything
  --project-name <name>   project name (skips prompt)
  --owner <owner>         owner / team name (skips prompt)
  --technology <tech>     technology stack (skips prompt)

Where each agent reads from:
  copilot    AGENTS.md              skills -> .agents/skills/
  codex      AGENTS.md              skills -> .agents/skills/
  opencode   AGENTS.md              skills -> .agents/skills/
  claude     AGENTS.md + CLAUDE.md  skills -> .claude/skills/

Agents that share a folder are written once. Without a terminal, init runs
non-interactively using the flags and defaults.

Examples:
  hikma init --agent claude --registry owner/registry --skill my-skill
  hikma init --agent claude,codex --project-name demo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("could not determine working directory: %w", err)
			}
			if path == "" {
				path = cwd
			} else {
				abs, err := filepath.Abs(path)
				if err != nil {
					return fmt.Errorf("invalid path: %w", err)
				}
				path = abs
			}
			if len(flagSkills) > 0 && path != cwd {
				return fmt.Errorf("--skill installs into the current directory; run hikma init from %s", path)
			}

			// Prompts only when attached to a terminal and not everything came from flags.
			allFlags := flagProjectName != "" && flagOwner != "" && flagTechnology != ""
			interactive := isInteractive() && !allFlags && !dryRun

			// Check for a git repository by walking up the directory tree.
			gitRoot, isGitRepo := scaffold.FindGitRoot(path)

			var reader *bufio.Reader
			if interactive {
				reader = bufio.NewReader(os.Stdin)
				if !isGitRepo {
					fmt.Fprintf(os.Stderr, "warning: this directory is not a git repository\n")
					fmt.Fprintf(os.Stderr, "  Files written here will not be version-controlled.\n")
					ans, err := prompt(reader, "Proceed anyway? [y/N]: ", "n")
					if err != nil {
						return err
					}
					if strings.ToLower(strings.TrimSpace(ans)) != "y" {
						fmt.Fprintln(out, "Aborted.")
						return nil
					}
				} else if gitRoot != path {
					fmt.Fprintf(out, "note: git root is %s - scaffolding into %s\n", gitRoot, path)
				}
			}

			projectName, owner, technology, err := resolveScaffoldValues(
				reader, path, flagProjectName, flagOwner, flagTechnology, !interactive,
			)
			if err != nil {
				return err
			}

			agents, err := chooseAgents(flagAgent, interactive)
			if err != nil {
				return err
			}
			targets, err := config.ResolveTargets(joinAgents(agents))
			if err != nil {
				return err
			}
			printTargets(out, targets, flagAgent != "")
			fmt.Fprintf(out, "Instruction files: %s\n", strings.Join(config.InstructionFiles(agents), ", "))

			// Resolve the registry up front so a bad --skill request fails before any file is written.
			registryValue := flagRegistry
			if registryValue == "" {
				if e, err := config.Lookup(config.KeyRegistry); err == nil {
					registryValue = e.Value
				}
			}
			if flagRegistry != "" {
				if _, err := source.Parse(flagRegistry); err != nil {
					return err
				}
			}
			if len(flagSkills) > 0 && registryValue == "" {
				return fmt.Errorf("--skill needs a registry: pass --registry <owner/repo> or run 'hikma config set registry <owner/repo>'")
			}

			if dryRun {
				fmt.Fprintf(out, "Dry run - nothing will be written.\n\n")
			} else {
				fmt.Fprintf(out, "\nSetting up %s...\n", projectName)
			}

			sr, err := scaffold.Scaffold(scaffold.Options{
				ProjectName: projectName,
				Owner:       owner,
				Technology:  technology,
				Agents:      agents,
				Registry:    registryValue,
				Path:        path,
				Force:       force,
				DryRun:      dryRun,
			})
			if err != nil {
				return err
			}

			if dryRun {
				for _, p := range sr.Created {
					fmt.Fprintf(out, "  would write %s\n", relOrFull(path, p))
				}
				fmt.Fprintf(out, "  would write .hikma/config.json (agents: %s)\n", joinAgents(agents))
				for _, name := range flagSkills {
					fmt.Fprintf(out, "  would install skill %s from %s\n", name, registryValue)
				}
				return nil
			}

			for _, p := range sr.Created {
				fmt.Fprintf(out, "  Created %s\n", relOrFull(path, p))
			}
			for _, p := range sr.Overwritten {
				fmt.Fprintf(out, "  Overwrote %s\n", relOrFull(path, p))
			}
			for _, p := range sr.Skipped {
				fmt.Fprintf(out, "  Skipped %-30s (already exists - use --force to overwrite)\n", relOrFull(path, p))
			}

			// Share the agents (and registry) with teammates through project config.
			projectCfg := config.ProjectConfigFile(path)
			if err := config.SetFile(projectCfg, config.KeyAgents, joinAgents(agents)); err != nil {
				return err
			}
			if flagRegistry != "" {
				if err := config.SetFile(projectCfg, config.KeyRegistry, flagRegistry); err != nil {
					return err
				}
			}
			fmt.Fprintf(out, "  Wrote %s\n", relOrFull(path, projectCfg))

			if len(flagSkills) > 0 {
				if err := installInitSkills(cmd, flagSkills, flagRegistry, targets, force); err != nil {
					return err
				}
			}

			fmt.Fprintln(out)
			if registryValue == "" {
				fmt.Fprintln(out, "Next: hikma config set registry <owner/repo> --project, then hikma skill install <name>")
			} else if len(flagSkills) == 0 {
				fmt.Fprintln(out, "Next: hikma skill list, then hikma skill install <name>")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "target directory (default: current working directory)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files and reinstall skills")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen without changing anything")
	cmd.Flags().StringVar(&flagProjectName, "project-name", "", "project name (skips interactive prompt)")
	cmd.Flags().StringVar(&flagOwner, "owner", "", "owner / team name (skips interactive prompt)")
	cmd.Flags().StringVar(&flagTechnology, "technology", "", "technology stack (skips interactive prompt)")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	cmd.Flags().StringVar(&flagRegistry, "registry", "", "project registry: owner/repo, git URL, or local path")
	cmd.Flags().StringArrayVar(&flagSkills, "skill", nil, "skill to install from the registry (repeatable)")

	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func joinAgents(agents []config.Agent) string {
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = string(a)
	}
	return strings.Join(names, ",")
}

// chooseAgents returns the agents to set up: the --agent flag, a prompt when
// interactive, or the already-configured agents.
func chooseAgents(flagAgent string, interactive bool) ([]config.Agent, error) {
	if flagAgent != "" {
		return config.ParseAgents(flagAgent)
	}
	current, err := config.ResolveTargets("")
	if err != nil {
		return nil, err
	}
	var defaults []string
	for _, t := range current {
		defaults = append(defaults, labelForAgent(t.Agent))
	}
	if !interactive {
		agents := make([]config.Agent, len(current))
		for i, t := range current {
			agents[i] = t.Agent
		}
		return agents, nil
	}

	var options []string
	byLabel := map[string]config.Agent{}
	for _, a := range config.ValidAgents {
		options = append(options, labelForAgent(a))
		byLabel[labelForAgent(a)] = a
	}
	var selected []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "AI agents (space to select):",
		Options: options,
		Default: defaults,
	}, &selected, survey.WithValidator(survey.MinItems(1))); err != nil {
		return nil, err
	}
	agents := make([]config.Agent, len(selected))
	for i, label := range selected {
		agents[i] = byLabel[label]
	}
	return agents, nil
}

func labelForAgent(a config.Agent) string {
	return fmt.Sprintf("%s - %s", a, config.LayoutFor(a).Description)
}

// installInitSkills installs the named skills from the registry into every target.
func installInitSkills(cmd *cobra.Command, names []string, flagRegistry string, targets []config.Selection, force bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	src, co, cleanup, err := openSource(out, "", flagRegistry, "")
	if err != nil {
		return err
	}
	defer cleanup()
	skills, err := discoverSkills(src, co)
	if err != nil {
		return err
	}
	for _, name := range names {
		skill, ok := source.Find(skills, name)
		if !ok {
			return fmt.Errorf("skill %q not found in %s", name, src.Display)
		}
		if err := installSkill(out, src, co, skill, targets, force, ""); err != nil {
			return err
		}
	}
	return nil
}

func resolveScaffoldValues(reader *bufio.Reader, path, projectName, owner, technology string, noPrompts bool) (string, string, string, error) {
	if noPrompts {
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
