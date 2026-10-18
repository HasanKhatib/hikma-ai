package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
)

func newSkillVerifyCmd() *cobra.Command {
	var flagAgent string
	var global, asJSON bool
	cmd := &cobra.Command{
		Use:   "verify [<name>]",
		Short: "Check installed skills against the hashes in .hikma/lock.json",
		Long: `Re-hash the installed files and compare them with .hikma/lock.json, offline.

Reports files that were added, modified, or removed since the install, and
skills whose folder is missing. It changes nothing and exits non-zero when any
skill differs, so it works as a CI check or a pre-commit hook. Without a name
every skill in the lockfile is checked.

Exit codes: 0 when everything matches, 3 when a skill differs, 1 when the check
could not run. --json prints one document (see the docs) on stdout.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if asJSON {
				out = io.Discard
			}
			sc, err := scopeFor(global)
			if err != nil {
				return err
			}
			lf, err := sc.loadLock()
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
					if k := sc.key(t, name); hasEntry(lf, k) {
						keys = append(keys, k)
					}
				}
				if len(keys) == 0 {
					return fmt.Errorf("%q is not installed by hikma for the selected agents (no entry in %s)\nrun 'hikma skill list --installed' to see what is installed", name, sc.lockName())
				}
			}
			if len(keys) == 0 {
				if asJSON {
					return printJSON(cmd.OutOrStdout(), verifyDoc{Version: jsonVersion, OK: true, Skills: []verifyRow{}})
				}
				fmt.Fprintln(out, "No skills recorded in .hikma/lock.json.")
				return nil
			}

			failed := 0
			rows := []verifyRow{}
			for _, key := range keys {
				e := lf.Skills[key]
				dir := sc.path(key)
				row := verifyRow{Path: key, Name: e.Name, Status: stateOK, Added: []string{}, Modified: []string{}, Removed: []string{}}
				if _, err := os.Stat(dir); err != nil {
					row.Status = stateMissing
					rows = append(rows, row)
					fmt.Fprintf(out, "  %s  %s\n          %s\n", ui.Red(out, "FAIL"), key, ui.Dim(out, "the folder is missing ('hikma sync' restores it)"))
					failed++
					continue
				}
				current, err := source.HashDir(dir)
				if err != nil {
					return err
				}
				added, modified, removed := source.DiffFiles(e.Files, current)
				row.Added, row.Modified, row.Removed = strs(added), strs(modified), strs(removed)
				if len(added)+len(modified)+len(removed) == 0 {
					rows = append(rows, row)
					fmt.Fprintf(out, "  %s    %s\n", ui.Green(out, "ok"), key)
					continue
				}
				row.Status = stateModified
				rows = append(rows, row)
				failed++
				fmt.Fprintf(out, "  %s  %s\n", ui.Red(out, "FAIL"), key)
				for _, g := range []struct {
					label string
					files []string
				}{{"modified", modified}, {"added", added}, {"removed", removed}} {
					for _, f := range g.files {
						fmt.Fprintf(out, "          %-8s %s\n", g.label, f)
					}
				}
				if touchesScripts(added, modified, removed) {
					fmt.Fprintf(out, "          %s scripts/ differ from what was installed\n", ui.Warning(out))
				}
			}
			fmt.Fprintf(out, "\n%d skill(s) checked, %d differ.\n", len(keys), failed)
			if asJSON {
				if err := printJSON(cmd.OutOrStdout(), verifyDoc{Version: jsonVersion, OK: failed == 0, Checked: len(keys), Differ: failed, Skills: rows}); err != nil {
					return err
				}
			}
			if failed > 0 {
				return mismatchError(fmt.Errorf("%d skill(s) differ from .hikma/lock.json", failed))
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return installedSkillCompletions(projectScope())
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON on stdout")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "use your home directory instead of this project (~/.hikma/lock.json)")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// verifyDoc is the --json document of 'hikma skill verify'.
type verifyDoc struct {
	Version int         `json:"version"`
	OK      bool        `json:"ok"`
	Checked int         `json:"checked"`
	Differ  int         `json:"differ"`
	Skills  []verifyRow `json:"skills"`
}

type verifyRow struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Status   string   `json:"status"` // ok, modified, or missing
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Removed  []string `json:"removed"`
}
