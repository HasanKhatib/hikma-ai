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
	Scope  string `json:"scope"`            // "project" or "global"
	AlsoAt string `json:"alsoAt,omitempty"` // the other scope, when the same name is installed there too
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
func listInstalledSkills(sc scope) ([]installedSkill, error) {
	lf, err := sc.loadLock()
	if err != nil {
		return nil, err
	}
	scopeName := "project"
	if sc.global {
		scopeName = "global"
	}
	var rows []installedSkill
	for _, key := range lf.Keys() {
		e := lf.Skills[key]
		row := installedSkill{Scope: scopeName, Name: e.Name, Path: key, Source: e.Source, Ref: e.Ref, Commit: e.Commit, Status: stateMissing}
		dir := sc.path(key)
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

	dirs, err := layoutSkillDirs()
	if err != nil {
		return nil, err
	}
	for base := range dirs {
		entries, err := os.ReadDir(sc.path(base))
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
			if _, err := os.Stat(filepath.Join(sc.path(key), "SKILL.md")); err != nil {
				continue
			}
			row := installedSkill{Scope: scopeName, Name: e.Name(), Path: key, Status: stateUntracked}
			if prov, ok := source.ReadProvenance(sc.path(key)); ok {
				row.Origin = "gh skill"
				row.Source = prov.Repo
			}
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	return rows, nil
}

func listInstalled(cmd *cobra.Command, asJSON, global bool) error {
	var rows []installedSkill
	if global {
		gsc, err := globalScope()
		if err != nil {
			return err
		}
		if rows, err = listInstalledSkills(gsc); err != nil {
			return err
		}
	} else {
		var err error
		if rows, err = listInstalledSkills(projectScope()); err != nil {
			return err
		}
		// Global skills are shown beside the project's so a duplicate is visible.
		// Reading them is harmless, so a missing or unreadable home is skipped.
		if gsc, err := globalScope(); err == nil && !sameDir(".", gsc.root) {
			if g, err := listInstalledSkills(gsc); err == nil {
				rows = markDuplicates(append(rows, g...))
			}
		}
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

// sameDir reports whether two paths are the same directory.
func sameDir(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	return err == nil && os.SameFile(ai, bi)
}

// markDuplicates sets AlsoAt on every row whose name is installed at both scopes.
func markDuplicates(rows []installedSkill) []installedSkill {
	scopes := map[string]map[string]bool{}
	for _, r := range rows {
		if scopes[r.Name] == nil {
			scopes[r.Name] = map[string]bool{}
		}
		scopes[r.Name][r.Scope] = true
	}
	for i, r := range rows {
		if len(scopes[r.Name]) < 2 {
			continue
		}
		if r.Scope == "global" {
			rows[i].AlsoAt = "project"
		} else {
			rows[i].AlsoAt = "global"
		}
	}
	return rows
}

func printInstalledTable(w io.Writer, rows []installedSkill) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "No skills installed.")
		return
	}
	fmt.Fprintf(w, "%-8s  %-42s  %-10s  %s\n", "SCOPE", "PATH", "STATUS", "SOURCE")
	for _, r := range rows {
		src := r.Source
		lockName := ".hikma/lock.json"
		if r.Scope == "global" {
			lockName = "~/.hikma/lock.json"
		}
		if r.Status == stateUntracked && r.Origin != "" {
			src = fmt.Sprintf("installed by %s from %s; run: hikma skill adopt %s", r.Origin, r.Source, r.Name)
			if r.Scope == "global" {
				src += " --global"
			}
		} else if src != "" {
			src += atCommit(lock.Entry{Commit: r.Commit, Ref: r.Ref})
		} else {
			src = "(not recorded in " + lockName + ")"
		}
		if r.AlsoAt != "" {
			src += "  [also installed at " + r.AlsoAt + " scope]"
		}
		fmt.Fprintf(w, "%-8s  %-42s  %-10s  %s\n", r.Scope, r.Path, r.Status, src)
	}
}

func newSkillRemoveCmd() *cobra.Command {
	var force, global, yes bool
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
			sc, err := scopeFor(global)
			if err != nil {
				return err
			}
			lf, err := sc.loadLock()
			if err != nil {
				return err
			}

			var cands []string
			for _, t := range targets {
				if k := sc.key(t, name); hasEntry(lf, k) || pathExists(sc.skillDir(t, name)) {
					cands = append(cands, k)
				}
			}
			if len(cands) > 0 {
				if err := sc.guardWrite(cands...); err != nil {
					return err
				}
				if err := sc.confirmWrite(cmd, yes, cands); err != nil {
					return err
				}
			}

			removed, blocked := 0, 0
			for _, t := range targets {
				dir := sc.skillDir(t, name)
				key := sc.key(t, name)
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
					pruneEmptyParents(sc.root, dir)
				}
				delete(lf.Skills, key)
				fmt.Fprintf(out, "  removed  %s\n", key)
				removed++
			}

			if removed > 0 {
				if len(lf.Skills) == 0 {
					if err := os.Remove(sc.lockPath()); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("remove empty lockfile: %w", err)
					}
				} else if err := sc.saveLock(lf); err != nil {
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
			return installedSkillCompletions(projectScope())
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "also remove untracked skills and skills with local changes")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "use your home directory instead of this project (~/.hikma/lock.json)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation before changing your home directory")
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
func pruneEmptyParents(root, dir string) {
	root = filepath.Clean(root)
	for p := filepath.Dir(filepath.Clean(dir)); ; p = filepath.Dir(p) {
		rel, err := filepath.Rel(root, p)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return
		}
		if os.Remove(p) != nil { // fails unless the directory is empty
			return
		}
	}
}
