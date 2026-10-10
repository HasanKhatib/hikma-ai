package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

func newSkillDiffCmd() *cobra.Command {
	var flagAgent string
	var patch bool
	cmd := &cobra.Command{
		Use:   "diff <name>",
		Short: "Show local edits and upstream changes for an installed skill",
		Long: `Show what differs for a skill recorded in .hikma/lock.json, without changing anything.

Local edits are files that differ from what was installed. Upstream changes are
what 'hikma skill update' would bring in from the recorded source. Files changed
on both sides are marked, since an update would overwrite your edits to them.
Pass --patch to print the file contents that differ.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill diff <name> [--patch]")
			}
			return nil
		},
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
			var keys []string
			for _, t := range targets {
				if k := lockKey(t.SkillDir(name)); hasEntry(lf, k) {
					keys = append(keys, k)
				}
			}
			if len(keys) == 0 {
				for _, t := range targets {
					if _, gh := source.ReadProvenance(t.SkillDir(name)); gh {
						return fmt.Errorf("%q was installed by gh skill and is not tracked by hikma; run 'hikma skill adopt %s' first", name, name)
					}
				}
				return fmt.Errorf("%q is not installed by hikma for the selected agents (no entry in .hikma/lock.json)", name)
			}

			u := &updater{out: out, errOut: cmd.ErrOrStderr(), lf: lf, checkouts: map[string]*openCheckout{}}
			defer u.close()
			for i, key := range keys {
				if i > 0 {
					fmt.Fprintln(out)
				}
				if err := diffOne(u, key, patch); err != nil {
					return err
				}
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
	cmd.Flags().BoolVar(&patch, "patch", false, "print the contents that differ")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func diffOne(u *updater, key string, patch bool) error {
	e := u.lf.Skills[key]
	localDir := filepath.FromSlash(key)
	fmt.Fprintf(u.out, "%s (%s%s)\n", key, e.Source, atCommit(e))

	missing := false
	var localAdded, localMod, localRem []string
	if _, err := os.Stat(localDir); err != nil {
		missing = true
		fmt.Fprintln(u.out, "  the folder is missing; 'hikma sync' restores it")
	} else {
		current, err := source.HashDir(localDir)
		if err != nil {
			return err
		}
		localAdded, localMod, localRem = source.DiffFiles(e.Files, current)
	}
	edited := map[string]bool{}
	for _, g := range [][]string{localAdded, localMod, localRem} {
		for _, f := range g {
			edited[f] = true
		}
	}

	if !missing {
		fmt.Fprintln(u.out, "Local edits:")
		if len(edited) == 0 {
			fmt.Fprintln(u.out, "  none")
		}
		printChanges(u.out, "added", localAdded)
		printChanges(u.out, "modified", localMod)
		printChanges(u.out, "removed", localRem)
	}

	c, err := u.checkout(e)
	if err != nil {
		return fmt.Errorf("check upstream for %s: %w", e.Name, err)
	}
	skills, err := discoverSkills(c.src, c.co)
	if err != nil {
		return err
	}
	skill, ok := source.Find(skills, e.Name)
	if !ok {
		return fmt.Errorf("skill %q no longer exists in %s", e.Name, c.src.Display)
	}
	srcDir := filepath.Join(c.co.Dir, filepath.FromSlash(skill.Rel))
	incoming, err := source.HashDir(srcDir)
	if err != nil {
		return err
	}
	added, modified, removed := source.DiffFiles(e.Files, incoming)

	fmt.Fprintf(u.out, "Upstream changes (%s @ %s):\n", c.src.Display, shortCommit(c.co.Commit))
	if len(added)+len(modified)+len(removed) == 0 {
		fmt.Fprintln(u.out, "  none, up to date")
		return nil
	}
	printMarked(u.out, "added", added, edited)
	printMarked(u.out, "modified", modified, edited)
	printMarked(u.out, "removed", removed, edited)
	if touchesScripts(added, modified, removed) {
		fmt.Fprintln(u.out, "  warning: scripts/ changed - review before updating")
	}
	if patch && !missing {
		return printPatch(u.out, localDir, srcDir, append(append(append([]string{}, added...), modified...), removed...))
	}
	return nil
}

func printMarked(w io.Writer, label string, files []string, edited map[string]bool) {
	for _, f := range files {
		note := ""
		if edited[f] {
			note = "  (also edited locally)"
		}
		fmt.Fprintf(w, "  %-8s %s%s\n", label, f, note)
	}
}

// printPatch prints the differences between the installed files and the
// upstream ones for the given relative paths, using git's diff output.
func printPatch(w io.Writer, localDir, srcDir string, files []string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("--patch needs git on your PATH")
	}
	for _, f := range files {
		a, b := filepath.Join(localDir, filepath.FromSlash(f)), filepath.Join(srcDir, filepath.FromSlash(f))
		if _, err := os.Stat(a); err != nil {
			a = os.DevNull
		}
		if _, err := os.Stat(b); err != nil {
			b = os.DevNull
		}
		// git exits 1 when the files differ.
		out, err := exec.Command("git", "diff", "--no-index", "--no-color", "--src-prefix=installed/", "--dst-prefix=upstream/", "--", a, b).Output()
		if err != nil && len(out) == 0 {
			return fmt.Errorf("diff %s: %w", path.Clean(f), err)
		}
		fmt.Fprintf(w, "\n%s", out)
	}
	return nil
}
