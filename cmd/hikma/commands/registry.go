package commands

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

func newRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Work with a skill registry repository",
	}
	cmd.AddCommand(newRegistryValidateCmd())
	return cmd
}

func newRegistryValidateCmd() *cobra.Command {
	var ref string
	cmd := &cobra.Command{
		Use:   "validate [<source>]",
		Short: "Check every skill in a registry against the skill format",
		Long: `Validate the skills in a registry repository.

For each skill this checks that SKILL.md has YAML frontmatter with a name that
matches the folder and a description, that the folder name follows your naming
setting (hikma config set naming), and that no template placeholders are left.

The source is owner/repo, a git URL, or a path ('.' for the repo you are in).
Without an argument the configured registry is used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			repo := ""
			if len(args) == 1 {
				repo = args[0]
			}
			src, co, cleanup, err := openSource(out, repo, "", ref)
			if err != nil {
				return err
			}
			defer cleanup()
			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			if len(skills) == 0 {
				return fmt.Errorf("no skills found in %s (expected skills/<name>/SKILL.md)", src.Display)
			}

			failed := 0
			for _, s := range skills {
				dir := filepath.Join(co.Dir, filepath.FromSlash(s.Rel))
				issues, err := validateSkillDirAs(dir, s.Name, s.Rel == ".")
				if err != nil {
					return err
				}
				if printIssuesFor(out, s.Name, issues) {
					failed++
				}
			}
			fmt.Fprintf(out, "\n%d skill(s) checked, %d failed.\n", len(skills), failed)
			if failed > 0 {
				return fmt.Errorf("%d skill(s) failed validation", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "", "branch, tag, or commit to validate")
	return cmd
}

// validateSkillDir validates a local skill folder named name.
func validateSkillDir(dir, name string) ([]source.Issue, error) {
	return validateSkillDirAs(dir, name, false)
}

func validateSkillDirAs(dir, name string, isRoot bool) ([]source.Issue, error) {
	issues := source.CheckSkill(dir, name, isRoot)
	if naming, err := config.NamingConvention(); err != nil {
		return nil, err
	} else if err := config.CheckSkillName(naming, name); err != nil {
		issues = append(issues, source.Issue{Severity: source.SeverityError, Message: err.Error()})
	}
	placeholders, err := scaffold.ValidatePlaceholders(dir)
	if err != nil {
		return nil, err
	}
	for _, p := range placeholders {
		issues = append(issues, source.Issue{Severity: source.SeverityError, Message: "unfilled placeholder " + p})
	}
	return issues, nil
}

// printIssuesFor prints the result for one skill and reports whether it has errors.
func printIssuesFor(w io.Writer, name string, issues []source.Issue) bool {
	bad := source.HasErrors(issues)
	if len(issues) == 0 {
		fmt.Fprintf(w, "  ok    %s\n", name)
		return false
	}
	label := "warn"
	if bad {
		label = "FAIL"
	}
	fmt.Fprintf(w, "  %s  %s\n", label, name)
	for _, i := range issues {
		fmt.Fprintf(w, "          %s: %s\n", i.Severity, i.Message)
	}
	return bad
}

// printIssues prints validation problems for the skill in dir and reports whether any is an error.
func printIssues(w io.Writer, dir string, issues []source.Issue) bool {
	return printIssuesFor(w, dir, issues)
}
