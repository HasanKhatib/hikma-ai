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

func newSkillAdoptCmd() *cobra.Command {
	var flagSource, flagRef, flagAgent string
	cmd := &cobra.Command{
		Use:   "adopt <name>",
		Short: "Track a skill installed by another tool in .hikma/lock.json",
		Long: `Record a skill that is already in an agent's skills folder in .hikma/lock.json,
so 'hikma skill update', 'sync', and 'remove' manage it from then on.

Skills installed by 'gh skill' carry their source in SKILL.md, and adopt reads
it from there. For any other skill pass --source (and optionally --ref). The
files on disk are recorded as they are; nothing is downloaded into the folder.
The gh skill metadata in SKILL.md is replaced the next time you update.`,
		Example: `  hikma skill adopt pdf
  hikma skill adopt my-skill --source owner/repo --ref v1.2.0`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill adopt <name> [--source owner/repo]")
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

			var dirs []string
			for _, t := range targets {
				dir := t.SkillDir(name)
				if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
					continue
				}
				if hasEntry(lf, lockKey(dir)) {
					fmt.Fprintf(out, "  already tracked  %s\n", lockKey(dir))
					continue
				}
				dirs = append(dirs, dir)
			}
			if len(dirs) == 0 {
				return fmt.Errorf("no untracked %q skill found in the selected agents' folders", name)
			}

			srcArg, ref := flagSource, flagRef
			if srcArg == "" {
				p, ok := source.ReadProvenance(dirs[0])
				if !ok {
					return fmt.Errorf("%s has no gh skill metadata; pass --source <owner/repo>", dirs[0])
				}
				srcArg, ref = p.SourceRef()
				if flagRef != "" {
					ref = flagRef
				}
			}
			src, err := source.Parse(srcArg)
			if err != nil {
				return err
			}
			co, cleanup, err := source.Resolve(src, ref)
			if err != nil {
				return err
			}
			defer cleanup()
			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			if _, ok := source.Find(skills, name); !ok {
				return fmt.Errorf("skill %q not found in %s; updates would fail", name, src.Display)
			}

			for _, dir := range dirs {
				files, err := source.HashDir(dir)
				if err != nil {
					return err
				}
				lf.Skills[lockKey(dir)] = lock.Entry{Name: name, Source: src.Raw, Ref: ref, Commit: co.Commit, Files: files}
				fmt.Fprintf(out, "  adopted  %s  (%s%s)\n", lockKey(dir), src.Display, atCommit(lock.Entry{Commit: co.Commit, Ref: ref}))
			}
			return lock.Save(".", lf)
		},
	}
	cmd.Flags().StringVar(&flagSource, "source", "", "where the skill came from (owner/repo, git URL, or path); default: read from gh skill metadata")
	cmd.Flags().StringVar(&flagRef, "ref", "", "branch, tag, or commit to update from")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// ghManagedUntracked returns skills that gh skill installed and hikma does not track.
func ghManagedUntracked() []installedSkill {
	rows, err := listInstalledSkills()
	if err != nil {
		return nil
	}
	var out []installedSkill
	for _, r := range rows {
		if r.Status == stateUntracked && r.Origin != "" {
			out = append(out, r)
		}
	}
	return out
}

func adoptHint(rows []installedSkill) string {
	names := map[string]bool{}
	var list []string
	for _, r := range rows {
		if !names[r.Name] {
			names[r.Name] = true
			list = append(list, r.Name)
		}
	}
	return strings.Join(list, ", ")
}
