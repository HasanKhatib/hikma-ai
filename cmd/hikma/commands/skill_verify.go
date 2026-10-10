package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

func newSkillVerifyCmd() *cobra.Command {
	var flagAgent string
	cmd := &cobra.Command{
		Use:   "verify [<name>]",
		Short: "Check installed skills against the hashes in .hikma/lock.json",
		Long: `Re-hash the installed files and compare them with .hikma/lock.json, offline.

Reports files that were added, modified, or removed since the install, and
skills whose folder is missing. It changes nothing and exits non-zero when any
skill differs, so it works as a CI check or a pre-commit hook. Without a name
every skill in the lockfile is checked.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			lf, err := lock.Load(".")
			if err != nil {
				return err
			}
			keys := lf.Keys()
			if len(args) == 1 {
				name := strings.TrimPrefix(strings.TrimSpace(args[0]), "@")
				if err := config.CheckSkillName(config.NamingLoose, name); err != nil {
					return err
				}
				targets, err := config.ResolveTargets(flagAgent)
				if err != nil {
					return fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
				}
				keys = nil
				for _, t := range targets {
					if k := lockKey(t.SkillDir(name)); hasEntry(lf, k) {
						keys = append(keys, k)
					}
				}
				if len(keys) == 0 {
					return fmt.Errorf("%q is not installed by hikma for the selected agents (no entry in .hikma/lock.json)", name)
				}
			}
			if len(keys) == 0 {
				fmt.Fprintln(out, "No skills recorded in .hikma/lock.json.")
				return nil
			}

			failed := 0
			for _, key := range keys {
				e := lf.Skills[key]
				dir := filepath.FromSlash(key)
				if _, err := os.Stat(dir); err != nil {
					fmt.Fprintf(out, "  FAIL  %s\n          the folder is missing ('hikma sync' restores it)\n", key)
					failed++
					continue
				}
				current, err := source.HashDir(dir)
				if err != nil {
					return err
				}
				added, modified, removed := source.DiffFiles(e.Files, current)
				if len(added)+len(modified)+len(removed) == 0 {
					fmt.Fprintf(out, "  ok    %s\n", key)
					continue
				}
				failed++
				fmt.Fprintf(out, "  FAIL  %s\n", key)
				for _, g := range []struct {
					label string
					files []string
				}{{"modified", modified}, {"added", added}, {"removed", removed}} {
					for _, f := range g.files {
						fmt.Fprintf(out, "          %-8s %s\n", g.label, f)
					}
				}
				if touchesScripts(added, modified, removed) {
					fmt.Fprintln(out, "          warning: scripts/ differ from what was installed")
				}
			}
			fmt.Fprintf(out, "\n%d skill(s) checked, %d differ.\n", len(keys), failed)
			if failed > 0 {
				return fmt.Errorf("%d skill(s) differ from .hikma/lock.json", failed)
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return installedSkillCompletions()
		},
	}
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
