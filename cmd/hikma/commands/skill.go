package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/preflight"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
)

func newSkillCmd() *cobra.Command {
	var flagRegistry string
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Install, update, diff, verify, remove, adopt, create, and publish skills",
	}
	cmd.PersistentFlags().StringVar(&flagRegistry, "registry", "", "registry to use for this command (owner/repo, git URL, or local path)")

	cmd.AddCommand(
		newSkillListCmd(&flagRegistry),
		newSkillInfoCmd(&flagRegistry),
		newSkillInstallCmd(&flagRegistry),
		newSkillUpdateCmd(),
		newSkillRemoveCmd(),
		newSkillAdoptCmd(),
		newSkillDiffCmd(),
		newSkillVerifyCmd(),
		newSkillCreateCmd(),
		newSkillPushCmd(&flagRegistry),
	)

	return cmd
}

func newSkillCreateCmd() *cobra.Command {
	var flagOwner, flagTeam, flagDescription string
	var flagAgent string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a new skill into the active agent's skill directory",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill create <name> [flags]\n\nExample: hikma skill create my-skill --description \"short description\" --team my-team")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts exactly one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := checkSkillName(name); err != nil {
				return err
			}

			isAgentOverride := flagAgent != ""
			selection, err := config.ResolveSelection(flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w\nrun 'hikma doctor' to check your environment", err)
			}
			printActiveSelection(selection, isAgentOverride)

			targetDir, err := resolveSkillDirForAgent(name, flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			if _, err := os.Stat(targetDir); err == nil {
				ui.UserError("%s already exists at %s/\n  choose a different name or remove the existing directory", name, targetDir)
				return fmt.Errorf("already exists")
			}

			// Resolve author from gh; flags override resolved values.
			owner := resolveGHUser()
			if flagOwner != "" {
				owner = flagOwner
			}

			fmt.Printf("Resolving author from gh... %s\n", owner)
			fmt.Println()

			created, err := scaffold.CreateSkill(scaffold.SkillOptions{
				SkillName:   name,
				Owner:       owner,
				Team:        flagTeam,
				Description: flagDescription,
				TargetDir:   targetDir,
			})
			if err != nil {
				return fmt.Errorf("scaffold skill: %w", err)
			}

			fmt.Printf("Created %s/\n", targetDir)
			for _, p := range created {
				rel := strings.TrimPrefix(p, targetDir+string(filepath.Separator))
				fmt.Printf("  %s\n", rel)
			}

			fmt.Printf("\nOwner: %s\n", owner)
			fmt.Printf("\nEdit %s/SKILL.md, then run 'hikma skill push %s'.\n", targetDir, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagOwner, "owner", "", "override owner (default: resolved from gh)")
	cmd.Flags().StringVar(&flagTeam, "team", "", "team name to write into SKILL.md metadata")
	cmd.Flags().StringVar(&flagDescription, "description", "", "short description to pre-fill in SKILL.md")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// checkSkillName validates a name for create and push against the configured naming convention.
func checkSkillName(name string) error {
	naming, err := config.NamingConvention()
	if err != nil {
		return err
	}
	return config.CheckSkillName(naming, name)
}

// resolveGHUser asks gh for the current user's login. On any error it
// returns "unknown".
func resolveGHUser() string {
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "unknown"
	}
	return strings.Trim(strings.TrimSpace(string(out)), `"`)
}

// normalizeSkillName accepts copied registry paths such as
// @skills/my-skill/ while preserving bare skill names.
func normalizeSkillName(input string) string {
	name := strings.TrimSpace(input)
	name = strings.TrimPrefix(name, "@")
	name = filepath.ToSlash(strings.Trim(name, "/"))

	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "skills" && i+1 < len(parts) && parts[i+1] != "" {
			return parts[i+1]
		}
	}

	return name
}

// printJSON marshals v to indented JSON and prints it.
func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// preflightError formats preflight failures into a single error.
func preflightError(result preflight.Result) error {
	var sb strings.Builder
	for _, f := range result.Failures {
		sb.WriteString(fmt.Sprintf("error: %s\n", f.Err))
		for _, fix := range f.Fix {
			sb.WriteString(fmt.Sprintf("  %s\n", fix))
		}
	}
	sb.WriteString("run 'hikma doctor' for details")
	return fmt.Errorf("%s", sb.String())
}

func resolveSkillDir(name, flagAgent string) (string, error) {
	return resolveSkillDirForAgent(name, flagAgent)
}

// resolveSkillDirForAgent returns the existing directory for name across the
// selected agents, or the first selected agent's directory when it does not exist yet.
func resolveSkillDirForAgent(name, flagAgent string) (string, error) {
	targets, err := config.ResolveTargets(flagAgent)
	if err != nil {
		return "", err
	}
	for _, t := range targets {
		if dir := t.SkillDir(name); dirExists(dir) {
			return dir, nil
		}
	}
	return targets[0].SkillDir(name), nil
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// printActiveSelection writes the active agent and resolved paths to stdout.
func printActiveSelection(selection config.Selection, isOverride bool) {
	fmt.Printf("Active agent: %s%s\n", selection.Agent, overrideSuffix(isOverride))
	fmt.Printf("Skill path: %s\n", selection.Layout.SkillsDir)
}

// confirmPush shows the target registry and asks the user to confirm.
func confirmPush(cmd *cobra.Command, name, skillDir string, r config.Registry, origin string, assumeYes bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\nYou are pushing to registry: %s (%s)\n", r.FullName(), origin)
	fmt.Fprintf(out, "  skill: %s (%s/)\n", name, skillDir)
	if assumeYes {
		return nil
	}
	if !isInteractive() {
		return fmt.Errorf("confirmation required: re-run with --yes to push to %s", r.FullName())
	}
	var ok bool
	if err := survey.AskOne(&survey.Confirm{Message: "Push to " + r.FullName() + "?", Default: false}, &ok); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("push cancelled")
	}
	return nil
}
