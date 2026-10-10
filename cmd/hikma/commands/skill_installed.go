package commands

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

// Installed-skill states.
const (
	stateOK        = "ok"
	stateModified  = "modified"
	stateMissing   = "missing"
	stateUntracked = "untracked"
)

type installedSkill struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Source string `json:"source,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Commit string `json:"commit,omitempty"`
	Origin string `json:"origin,omitempty"` // "gh skill" for untracked skills that carry its metadata
}

// listInstalledSkills combines the lockfile with the skill folders on disk.
// Skills in an agent folder without a lockfile entry are reported as untracked.
func listInstalledSkills() ([]installedSkill, error) {
	lf, err := lock.Load(".")
	if err != nil {
		return nil, err
	}
	var rows []installedSkill
	for _, key := range lf.Keys() {
		e := lf.Skills[key]
		row := installedSkill{Name: e.Name, Path: key, Source: e.Source, Ref: e.Ref, Commit: e.Commit, Status: stateMissing}
		dir := filepath.FromSlash(key)
		if dirExists(dir) {
			current, err := source.HashDir(dir)
			if err != nil {
				return nil, err
			}
			if sameFiles(e.Files, current) {
				row.Status = stateOK
			} else {
				row.Status = stateModified
			}
		}
		rows = append(rows, row)
	}

	for base := range layoutSkillDirs() {
		entries, err := os.ReadDir(filepath.FromSlash(base))
		if err != nil {
			continue
		}
		for _, e := range entries {
			key := path.Join(base, e.Name())
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if _, tracked := lf.Skills[key]; tracked {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.FromSlash(key), "SKILL.md")); err != nil {
				continue
			}
			row := installedSkill{Name: e.Name(), Path: key, Status: stateUntracked}
			if prov, ok := source.ReadProvenance(filepath.FromSlash(key)); ok {
				row.Origin = "gh skill"
				row.Source = prov.Repo
			}
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	return rows, nil
}

func listInstalled(cmd *cobra.Command, asJSON bool) error {
	rows, err := listInstalledSkills()
	if err != nil {
		return err
	}
	if asJSON {
		if rows == nil {
			rows = []installedSkill{}
		}
		return printJSON(rows)
	}
	printInstalledTable(cmd.OutOrStdout(), rows)
	return nil
}

func printInstalledTable(w io.Writer, rows []installedSkill) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "No skills installed in this repository.")
		return
	}
	fmt.Fprintf(w, "%-42s  %-10s  %s\n", "PATH", "STATUS", "SOURCE")
	for _, r := range rows {
		src := r.Source
		if r.Status == stateUntracked && r.Origin != "" {
			src = fmt.Sprintf("installed by %s from %s; run: hikma skill adopt %s", r.Origin, r.Source, r.Name)
		} else if src != "" {
			src += atCommit(lock.Entry{Commit: r.Commit, Ref: r.Ref})
		} else {
			src = "(not recorded in .hikma/lock.json)"
		}
		fmt.Fprintf(w, "%-42s  %-10s  %s\n", r.Path, r.Status, src)
	}
}

func newSkillRemoveCmd() *cobra.Command {
	var force bool
	var flagAgent string
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed skill",
		Long: `Remove a skill from the selected agents' skills folders and from .hikma/lock.json.

Only skills recorded in the lockfile are removed. A skill that is not tracked,
or that you edited locally, is left in place unless you pass --force. A removed
skill can be brought back with 'hikma sync' (unchanged skills) or 'hikma skill
install'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			name := strings.TrimPrefix(strings.TrimSpace(args[0]), "@")
			if err := config.CheckSkillName(config.NamingLoose, name); err != nil {
				return err
			}
			targets, err := config.ResolveTargets(flagAgent)
			if err != nil {
				return fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
			}
			lf, err := lock.Load(".")
			if err != nil {
				return err
			}

			removed, blocked := 0, 0
			for _, t := range targets {
				dir := t.SkillDir(name)
				key := lockKey(dir)
				entry, tracked := lf.Skills[key]
				exists := pathExists(dir)
				if !tracked && !exists {
					continue
				}
				if !force {
					if !tracked {
						fmt.Fprintf(cmd.ErrOrStderr(), "  kept     %s (not installed by hikma; use --force to delete it)\n", key)
						blocked++
						continue
					}
					if exists {
						current, err := source.HashDir(dir)
						if err != nil {
							return err
						}
						if !sameFiles(entry.Files, current) {
							fmt.Fprintf(cmd.ErrOrStderr(), "  kept     %s (local changes; use --force to delete them)\n", key)
							blocked++
							continue
						}
					}
				}
				if exists {
					if err := os.RemoveAll(dir); err != nil {
						return fmt.Errorf("remove %s: %w", dir, err)
					}
					pruneEmptyParents(dir)
				}
				delete(lf.Skills, key)
				fmt.Fprintf(out, "  removed  %s\n", key)
				removed++
			}

			if removed > 0 {
				if len(lf.Skills) == 0 {
					if err := os.Remove(lock.Path(".")); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("remove empty lockfile: %w", err)
					}
				} else if err := lock.Save(".", lf); err != nil {
					return err
				}
			}
			switch {
			case blocked > 0:
				return fmt.Errorf("%d install(s) were kept", blocked)
			case removed == 0:
				return fmt.Errorf("%q is not installed for the selected agents", name)
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
	cmd.Flags().BoolVar(&force, "force", false, "also remove untracked skills and skills with local changes")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func pathExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// pruneEmptyParents removes now-empty parent folders of a removed skill
// (for example .claude/skills and .claude), stopping at the first one that
// still has content or at the project root.
func pruneEmptyParents(dir string) {
	for p := filepath.Dir(filepath.Clean(dir)); p != "." && p != ".." && !filepath.IsAbs(p); p = filepath.Dir(p) {
		if os.Remove(p) != nil { // fails unless the directory is empty
			return
		}
	}
}
