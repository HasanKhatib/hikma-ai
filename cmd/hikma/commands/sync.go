package commands

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var force, yes, dryRun, frozen bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Restore every skill recorded in .hikma/lock.json",
		Long: `Install the skills recorded in .hikma/lock.json, like npm ci: use it on a fresh
clone to get exactly the skills the repository was committed with.

Each skill is fetched from its recorded source at its recorded commit, and its
files must match the recorded hashes; anything else is refused. Skills that
are already installed and unchanged are left alone. Skills you edited locally
are skipped unless you pass --force.

Because the lockfile comes from the repository you cloned, sync lists what it
will fetch and asks you to confirm (--yes skips the prompt, and is required
without a terminal). Lockfile entries that point outside the agent skill
folders are rejected.

With --frozen, sync must reproduce the lockfile exactly and fails, changing
nothing, if it cannot: a skill with local changes, an entry not pinned to a
commit, or a source that cannot be fetched at its recorded commit. Use it in CI.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(cmd, force, yes, dryRun, frozen)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite skills that have local changes")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be restored without changing anything")
	cmd.Flags().BoolVar(&frozen, "frozen", false, "fail on any drift instead of skipping: for CI (needs --yes)")
	return cmd
}

type syncState string

const (
	syncRestore  syncState = "restore"  // not installed yet
	syncOK       syncState = "ok"       // installed and unchanged
	syncModified syncState = "modified" // installed with local changes
)

type syncItem struct {
	key   string
	entry lock.Entry
	state syncState
}

// layoutSkillDirs returns every skills folder a built-in agent layout uses.
func layoutSkillDirs() map[string]bool {
	dirs := map[string]bool{}
	for _, a := range config.ValidAgents {
		dirs[config.LayoutFor(a).SkillsDir] = true
	}
	return dirs
}

// validateLockEntry rejects lockfile entries that would write outside the
// skills folders. The lockfile is committed to the repository, so it is untrusted input.
func validateLockEntry(key string, e lock.Entry) error {
	if e.Name == "" {
		return fmt.Errorf("%s: entry has no name", key)
	}
	if err := config.CheckSkillName(config.NamingLoose, e.Name); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	clean := path.Clean(key)
	if clean != key || path.IsAbs(key) || strings.Contains(key, `\`) || strings.HasPrefix(key, "../") || key == ".." {
		return fmt.Errorf("%s: install path must be a clean relative path", key)
	}
	if path.Base(key) != e.Name || !layoutSkillDirs()[path.Dir(key)] {
		return fmt.Errorf("%s: install path must be <agent skills folder>/%s", key, e.Name)
	}
	if e.Source == "" {
		return fmt.Errorf("%s: entry has no source", key)
	}
	if _, err := source.Parse(e.Source); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}

func sameFiles(a, b map[string]string) bool {
	added, modified, removed := source.DiffFiles(a, b)
	return len(added)+len(modified)+len(removed) == 0
}

func runSync(cmd *cobra.Command, force, yes, dryRun, frozen bool) error {
	out := cmd.OutOrStdout()
	lf, err := lock.Load(".")
	if err != nil {
		return err
	}
	keys := lf.Keys()
	if len(keys) == 0 {
		fmt.Fprintln(out, "No skills recorded in .hikma/lock.json.")
		return nil
	}

	var items []syncItem
	var invalid []string
	for _, key := range keys {
		e := lf.Skills[key]
		if err := validateLockEntry(key, e); err != nil {
			invalid = append(invalid, err.Error())
			continue
		}
		it := syncItem{key: key, entry: e, state: syncRestore}
		if dirExists(filepath.FromSlash(key)) {
			current, err := source.HashDir(filepath.FromSlash(key))
			if err != nil {
				return err
			}
			if sameFiles(e.Files, current) {
				it.state = syncOK
			} else {
				it.state = syncModified
			}
		}
		items = append(items, it)
	}
	if len(invalid) > 0 {
		for _, msg := range invalid {
			fmt.Fprintf(cmd.ErrOrStderr(), "  refused  %s\n", msg)
		}
		return fmt.Errorf("%d lockfile entr(ies) refused; nothing was changed", len(invalid))
	}

	if frozen {
		if problems := frozenProblems(items, force); len(problems) > 0 {
			for _, msg := range problems {
				fmt.Fprintf(cmd.ErrOrStderr(), "  drift    %s\n", msg)
			}
			return fmt.Errorf("frozen: %d problem(s) with .hikma/lock.json; nothing was changed", len(problems))
		}
	}

	var work []syncItem
	for _, it := range items {
		if it.state == syncRestore || (it.state == syncModified && force) {
			work = append(work, it)
		}
	}
	upToDate := 0
	for _, it := range items {
		if it.state == syncOK {
			upToDate++
		}
	}
	if len(work) == 0 && upToDate == len(items) {
		fmt.Fprintf(out, "All %d skill(s) are up to date.\n", len(items))
		return nil
	}

	fmt.Fprintf(out, "Skills recorded in .hikma/lock.json: %d\n", len(items))
	for _, it := range items {
		fmt.Fprintf(out, "  %-9s %s  (%s%s)\n", planLabel(it, force), it.key, it.entry.Source, atCommit(it.entry))
	}

	if dryRun {
		fmt.Fprintln(out, "\nDry run - nothing was changed.")
		return nil
	}
	if len(work) > 0 {
		if err := confirmSync(out, work, yes); err != nil {
			return err
		}
	}

	restored, failures := 0, 0
	checkouts := map[string]*openCheckout{}
	defer func() {
		for _, c := range checkouts {
			c.cleanup()
		}
	}()
	for _, it := range work {
		if err := restoreItem(out, it, checkouts); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  failed   %s: %v\n", it.key, err)
			failures++
			continue
		}
		fmt.Fprintf(out, "  restored %s\n", it.key)
		restored++
	}

	skipped := 0
	for _, it := range items {
		if it.state == syncModified && !force {
			skipped++
			fmt.Fprintf(out, "  skipped  %s (local changes; use --force to overwrite)\n", it.key)
		}
	}
	fmt.Fprintf(out, "\nDone: %d restored, %d up to date, %d skipped, %d failed.\n", restored, upToDate, skipped, failures)
	if failures > 0 || skipped > 0 {
		return fmt.Errorf("sync incomplete")
	}
	return nil
}

func atCommit(e lock.Entry) string {
	if e.Commit != "" {
		return " @ " + shortCommit(e.Commit)
	}
	if e.Ref != "" {
		return " @ " + e.Ref
	}
	return ""
}

func planLabel(it syncItem, force bool) string {
	switch it.state {
	case syncRestore:
		return "restore"
	case syncModified:
		if force {
			return "overwrite"
		}
		return "modified"
	}
	return "ok"
}

func confirmSync(out io.Writer, work []syncItem, yes bool) error {
	sources := map[string]bool{}
	for _, it := range work {
		sources[it.entry.Source] = true
	}
	if yes {
		return nil
	}
	if !isInteractive() {
		return fmt.Errorf("confirmation required: re-run with --yes to fetch %d skill(s) from %d source(s)", len(work), len(sources))
	}
	var ok bool
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Fetch %d skill(s) from %d source(s) listed above?", len(work), len(sources)),
		Default: false,
	}, &ok); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("sync cancelled")
	}
	return nil
}

// restoreItem installs one lockfile entry from its recorded source and commit,
// refusing content that does not match the recorded hashes.
func restoreItem(out io.Writer, it syncItem, checkouts map[string]*openCheckout) error {
	e := it.entry
	src, err := source.Parse(e.Source)
	if err != nil {
		return err
	}
	ref := e.Commit
	if ref == "" {
		ref = e.Ref
	}
	if src.Kind == source.KindLocal {
		ref = "" // a local path cannot be checked out at a commit; the hash check below still applies
	}
	cacheKey := e.Source + "@" + ref
	c, ok := checkouts[cacheKey]
	if !ok {
		fmt.Fprintf(out, "Fetching %s%s...\n", src.Display, atCommit(e))
		co, cleanup, err := source.Resolve(src, ref)
		if err != nil {
			return err
		}
		c = &openCheckout{src: src, co: co, cleanup: cleanup}
		checkouts[cacheKey] = c
	}
	skills, err := discoverSkills(c.src, c.co)
	if err != nil {
		return err
	}
	skill, ok := source.Find(skills, e.Name)
	if !ok {
		return fmt.Errorf("skill %q not found in %s%s", e.Name, src.Display, atCommit(e))
	}
	srcDir := filepath.Join(c.co.Dir, filepath.FromSlash(skill.Rel))
	incoming, err := source.HashDir(srcDir)
	if err != nil {
		return err
	}
	if added, modified, removed := source.DiffFiles(e.Files, incoming); len(added)+len(modified)+len(removed) > 0 {
		return fmt.Errorf("files do not match .hikma/lock.json (%s); refusing to install", describeDrift(added, modified, removed))
	}
	_, err = replaceDir(filepath.FromSlash(it.key), func(dst string) ([]string, error) {
		return source.CopyDir(srcDir, dst)
	})
	if err != nil {
		return err
	}
	// Make sure the installed files really are what the lockfile says.
	installed, err := source.HashDir(filepath.FromSlash(it.key))
	if err != nil {
		return err
	}
	if !sameFiles(e.Files, installed) {
		_ = os.RemoveAll(filepath.FromSlash(it.key))
		return fmt.Errorf("installed files do not match .hikma/lock.json")
	}
	return nil
}

func describeDrift(added, modified, removed []string) string {
	var parts []string
	if len(modified) > 0 {
		parts = append(parts, "changed: "+strings.Join(modified, ", "))
	}
	if len(added) > 0 {
		parts = append(parts, "unexpected: "+strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		parts = append(parts, "missing: "+strings.Join(removed, ", "))
	}
	return strings.Join(parts, "; ")
}

// frozenProblems lists why a frozen sync cannot reproduce the lockfile exactly.
func frozenProblems(items []syncItem, force bool) []string {
	var problems []string
	for _, it := range items {
		if it.state == syncModified && !force {
			problems = append(problems, fmt.Sprintf("%s has local changes (use --force to overwrite them)", it.key))
		}
		if it.state != syncOK && it.entry.Commit == "" {
			if src, err := source.Parse(it.entry.Source); err != nil || src.Kind != source.KindLocal {
				problems = append(problems, fmt.Sprintf("%s is not pinned to a commit in the lockfile", it.key))
			}
		}
	}
	return problems
}
