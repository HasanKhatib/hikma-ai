package commands

import (
	"fmt"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/spf13/cobra"
)

func newConfigAddAgentCmd() *cobra.Command {
	var skillsDir, instructionFile string
	var project, user, use, force, yes bool
	cmd := &cobra.Command{
		Use:   "add-agent <name>",
		Short: "Define a custom agent: where it reads skills from",
		Args:  cobra.ExactArgs(1),
		Long: `Define a custom agent for a tool hikma does not support out of the box.

Without flags it asks for the skills folder, shows the config file it will
write, and asks before writing. The folder is not created now: the first
'hikma skill install --agent <name>' creates it.

The definition is saved under customAgents in the project config
(.hikma/config.json, shared with your team) or your user config. Using the
agent is a separate step: add it to 'agents', or answer yes to the last question.`,
		Example: `  hikma config add-agent acme
  hikma config add-agent acme --skills-dir .acme/skills --instruction-file ACME.md --project --use`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigAddAgent(cmd, args[0], skillsDir, instructionFile, project, user, use, force, yes)
		},
	}
	f := cmd.Flags()
	f.StringVar(&skillsDir, "skills-dir", "", "folder the agent reads skills from, relative to the project (for example .acme/skills)")
	f.StringVar(&instructionFile, "instruction-file", "", "optional instruction file hikma init writes for this agent (for example ACME.md)")
	f.BoolVar(&project, "project", false, "save to .hikma/config.json in this repository")
	f.BoolVar(&user, "user", false, "save to your user config")
	f.BoolVar(&use, "use", false, "also add the agent to the agents list in the same file")
	f.BoolVar(&force, "force", false, "replace an existing definition")
	f.BoolVarP(&yes, "yes", "y", false, "accept the default answer to every question")
	return cmd
}

func runConfigAddAgent(cmd *cobra.Command, name, skillsDir, instructionFile string, project, user, use, force, yes bool) error {
	out := cmd.OutOrStdout()
	if project && user {
		return fmt.Errorf("use either --project or --user, not both")
	}
	if err := config.ValidateCustomAgent(name, config.CustomAgent{SkillsDir: "skills"}); err != nil {
		return err
	}
	interactive := isInteractive() && !yes

	if skillsDir == "" {
		if !interactive {
			return fmt.Errorf("--skills-dir is required when there is no terminal to ask")
		}
		validate := func(ans interface{}) error {
			return config.ValidateCustomAgent(name, config.CustomAgent{SkillsDir: ans.(string)})
		}
		if err := survey.AskOne(&survey.Input{Message: "Skills folder (relative to the project):", Help: "For example .acme/skills"}, &skillsDir, survey.WithValidator(survey.Required), survey.WithValidator(validate)); err != nil {
			return err
		}
	}
	def := config.CustomAgent{SkillsDir: skillsDir, InstructionFile: instructionFile}
	if err := config.ValidateCustomAgent(name, def); err != nil {
		return err
	}

	scope, err := chooseAgentScope(cmd, project, user, interactive, yes)
	if err != nil {
		return err
	}
	if scope == "" {
		fmt.Fprintln(out, "Cancelled; nothing was written.")
		return nil
	}
	path, err := config.ScopePath(scope)
	if err != nil {
		return err
	}

	if !use && interactive {
		// A user-level agents list affects every repository, so it defaults to no there.
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Use %q in this config now (add it to agents)?", name),
			Default: scope == config.ScopeProject,
		}, &use); err != nil {
			return err
		}
	}

	path, err = config.AddCustomAgent(scope, name, def, force, use)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Added agent %q to %s\n  skills: %s\n", name, path, def.SkillsDir)
	if def.InstructionFile != "" {
		fmt.Fprintf(out, "  instructions: %s\n", def.InstructionFile)
	}
	if use {
		fmt.Fprintf(out, "  agents: now includes %s\n", name)
	} else {
		fmt.Fprintf(out, "Use it with: hikma skill install <skill> --agent %s\n  or: hikma config set agents <list including %s>%s\n", name, name, projectFlag(scope))
	}
	return nil
}

func projectFlag(scope config.Scope) string {
	if scope == config.ScopeProject {
		return " --project"
	}
	return ""
}

// chooseAgentScope resolves which config file to write: from the flags, from
// y/n questions that show each file, or the project default under --yes.
// It returns "" when the user declines both files.
func chooseAgentScope(cmd *cobra.Command, project, user, interactive, yes bool) (config.Scope, error) {
	switch {
	case project:
		return config.ScopeProject, nil
	case user:
		return config.ScopeUser, nil
	case yes:
		return config.ScopeProject, nil
	case !interactive:
		return "", fmt.Errorf("pass --project or --user to choose the config file")
	}
	for _, s := range []struct {
		scope config.Scope
		label string
	}{{config.ScopeProject, "the project config"}, {config.ScopeUser, "your user config"}} {
		path, err := config.ScopePath(s.scope)
		if err != nil {
			return "", err
		}
		ok := true
		if err := survey.AskOne(&survey.Confirm{Message: fmt.Sprintf("Save to %s (%s)?", s.label, path), Default: true}, &ok); err != nil {
			return "", err
		}
		if ok {
			return s.scope, nil
		}
	}
	return "", nil
}

func newConfigRemoveAgentCmd() *cobra.Command {
	var project, user, yes bool
	cmd := &cobra.Command{
		Use:   "remove-agent <name>",
		Short: "Remove a custom agent definition",
		Args:  cobra.ExactArgs(1),
		Long: `Remove a custom agent from the project or user config.

It also removes the name from the agents list and from agent in that same file,
so the config stays valid. Skills already installed in the agent's folder and
entries in .hikma/lock.json are left alone.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigRemoveAgent(cmd, args[0], project, user, yes)
		},
	}
	cmd.Flags().BoolVar(&project, "project", false, "remove from .hikma/config.json in this repository")
	cmd.Flags().BoolVar(&user, "user", false, "remove from your user config")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	return cmd
}

func runConfigRemoveAgent(cmd *cobra.Command, name string, project, user, yes bool) error {
	out := cmd.OutOrStdout()
	if project && user {
		return fmt.Errorf("use either --project or --user, not both")
	}
	var scope config.Scope
	switch {
	case project:
		scope = config.ScopeProject
	case user:
		scope = config.ScopeUser
	default:
		var found []config.Scope
		for _, s := range []config.Scope{config.ScopeProject, config.ScopeUser} {
			if _, ok, err := config.CustomAgentIn(s, name); err != nil {
				return err
			} else if ok {
				found = append(found, s)
			}
		}
		switch len(found) {
		case 0:
			return fmt.Errorf("custom agent %q is not defined in the project or user config", name)
		case 1:
			scope = found[0]
		default:
			return fmt.Errorf("custom agent %q is defined in both the project and user config; pass --project or --user", name)
		}
	}
	path, err := config.ScopePath(scope)
	if err != nil {
		return err
	}
	if _, ok, err := config.CustomAgentIn(scope, name); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("custom agent %q is not defined in %s", name, path)
	}
	if !yes {
		if !isInteractive() {
			return fmt.Errorf("confirmation required: re-run with --yes to remove %q from %s", name, path)
		}
		var ok bool
		if err := survey.AskOne(&survey.Confirm{Message: fmt.Sprintf("Remove %q from %s?", name, path), Default: false}, &ok); err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Cancelled; nothing was changed.")
			return nil
		}
	}
	path, removed, err := config.RemoveCustomAgent(scope, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed from %s: %s\n", path, strings.Join(removed, ", "))
	return nil
}
