package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/lock"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// resolveSource picks the source for a command: --registry flag, then config.
// The second return value says where the choice came from.
func resolveSource(flagRegistry string) (source.Source, string, error) {
	if flagRegistry != "" {
		s, err := source.Parse(flagRegistry)
		return s, "--registry flag", err
	}
	e, err := config.Lookup(config.KeyRegistry)
	if err != nil {
		return source.Source{}, "", err
	}
	if e.Value == "" {
		return source.Source{}, "", fmt.Errorf("no registry configured - run 'hikma config set registry <owner/repo>', pass --registry, or name a repo: hikma skill install <owner/repo> <name>")
	}
	s, err := source.Parse(e.Value)
	if err != nil {
		return source.Source{}, "", err
	}
	return s, originLabel(e), nil
}

func originLabel(e config.Entry) string {
	switch e.Source {
	case config.SourceEnv:
		return "env " + config.EnvVar(e.Key)
	case config.SourceProject:
		return "project config"
	case config.SourceUser:
		return "user config"
	}
	return string(e.Source)
}

// sourceArgs splits "[<owner/repo>] [<name>]" arguments. A single argument that
// looks like a repo, URL, or path is a source; otherwise it is a skill name.
func sourceArgs(args []string) (repo, name string) {
	switch len(args) {
	case 2:
		return args[0], args[1]
	case 1:
		if source.LooksLikeSource(args[0]) {
			return args[0], ""
		}
		return "", strings.TrimPrefix(strings.TrimSpace(args[0]), "@")
	}
	return "", ""
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// openSource resolves the source for a command and returns its checkout.
func openSource(w io.Writer, repo, flagRegistry, ref string) (source.Source, source.Checkout, func(), error) {
	var (
		src    source.Source
		origin string
		err    error
	)
	if repo != "" {
		src, err = source.Parse(repo)
		origin = "argument"
	} else {
		src, origin, err = resolveSource(flagRegistry)
	}
	if err != nil {
		return src, source.Checkout{}, func() {}, err
	}
	fmt.Fprintf(w, "Source: %s (%s)\n", src.Display, origin)
	co, cleanup, err := source.Resolve(src, ref)
	if err != nil {
		return src, source.Checkout{}, func() {}, err
	}
	if co.Commit != "" {
		fmt.Fprintf(w, "Commit: %s\n", shortCommit(co.Commit))
	}
	return src, co, cleanup, nil
}

func discoverSkills(src source.Source, co source.Checkout) ([]source.Skill, error) {
	fallback := src.Repo
	if fallback == "" {
		fallback = filepath.Base(strings.TrimSuffix(strings.TrimSuffix(src.Display, "/"), ".git"))
	}
	return source.Discover(co.Dir, fallback)
}

func isInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func pickSkill(w io.Writer, skills []source.Skill) (source.Skill, error) {
	if len(skills) == 0 {
		return source.Skill{}, fmt.Errorf("no skills found in this source")
	}
	if !isInteractive() {
		printSkillTable(w, skills)
		return source.Skill{}, fmt.Errorf("choose a skill: hikma skill install <owner/repo> <name>")
	}
	labels := make([]string, len(skills))
	for i, s := range skills {
		labels[i] = s.Name
		if s.Description != "" {
			labels[i] += " - " + truncate(s.Description, 70)
		}
	}
	var idx int
	if err := survey.AskOne(&survey.Select{Message: "Skill:", Options: labels}, &idx); err != nil {
		return source.Skill{}, err
	}
	return skills[idx], nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

func printSkillTable(w io.Writer, skills []source.Skill) {
	if len(skills) == 0 {
		fmt.Fprintln(w, "No skills found.")
		return
	}
	const nameW, descW = 25, 50
	fmt.Fprintf(w, "%-*s  %-*s  %s\n", nameW, "NAME", descW, "DESCRIPTION", "OWNER")
	for _, s := range skills {
		fmt.Fprintf(w, "%-*s  %-*s  %s\n", nameW, s.Name, descW, truncate(s.Description, descW), s.Owner)
	}
}

func newSkillListCmd(flagRegistry *string) *cobra.Command {
	var asJSON, installed bool
	cmd := &cobra.Command{
		Use:   "list [<owner/repo>]",
		Short: "List skills in the configured registry or in any repo, or what is installed",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if installed {
				if len(args) > 0 {
					return fmt.Errorf("--installed lists this repository's skills and takes no source")
				}
				return listInstalled(cmd, asJSON)
			}
			status := cmd.OutOrStdout()
			if asJSON {
				status = cmd.ErrOrStderr()
			}
			repo := ""
			if len(args) == 1 {
				repo = args[0]
			}
			src, co, cleanup, err := openSource(status, repo, *flagRegistry, "")
			if err != nil {
				return err
			}
			defer cleanup()
			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(skills)
			}
			printSkillTable(cmd.OutOrStdout(), skills)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&installed, "installed", false, "list the skills installed in this repository instead")
	return cmd
}

func newSkillInfoCmd(flagRegistry *string) *cobra.Command {
	var asJSON bool
	var ref string
	cmd := &cobra.Command{
		Use:   "info [<owner/repo>] <name>",
		Short: "Show details for a skill",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			status := cmd.OutOrStdout()
			if asJSON {
				status = cmd.ErrOrStderr()
			}
			repo, name := sourceArgs(args)
			src, co, cleanup, err := openSource(status, repo, *flagRegistry, ref)
			if err != nil {
				return err
			}
			defer cleanup()
			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			if name == "" {
				if name, err = nameFromPicker(status, skills); err != nil {
					return err
				}
			}
			skill, ok := source.Find(skills, name)
			if !ok {
				return fmt.Errorf("skill %q not found in %s", name, src.Display)
			}
			if asJSON {
				return printJSON(skill)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Name:        %s\n", skill.Name)
			fmt.Fprintf(w, "Description: %s\n", skill.Description)
			if skill.Owner != "" {
				fmt.Fprintf(w, "Owner:       %s\n", skill.Owner)
			}
			fmt.Fprintf(w, "Source:      %s\n", src.Display)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	cmd.Flags().StringVar(&ref, "ref", "", "branch, tag, or commit to read")
	return cmd
}

func nameFromPicker(w io.Writer, skills []source.Skill) (string, error) {
	s, err := pickSkill(w, skills)
	return s.Name, err
}

func newSkillInstallCmd(flagRegistry *string) *cobra.Command {
	var force bool
	var flagAgent, ref string

	cmd := &cobra.Command{
		Use:   "install [<owner/repo>] [<name>]",
		Short: "Install a skill from the configured registry or any repo",
		Long: `Install a skill into the active agent's skill directory.

  hikma skill install my-skill               from your configured registry
  hikma skill install owner/repo my-skill    from any GitHub repo
  hikma skill install owner/repo             pick from the skills in that repo
  hikma skill install ./local/path my-skill  from a local checkout

Sources can be GitHub owner/repo, any git URL, or a local path. Use --ref to pin
a branch, tag, or commit. Skill names are not restricted on install.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			targets, err := config.ResolveTargets(flagAgent)
			if err != nil {
				return fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
			}
			printTargets(out, targets, flagAgent != "")

			repo, name := sourceArgs(args)
			src, co, cleanup, err := openSource(out, repo, *flagRegistry, ref)
			if err != nil {
				return err
			}
			defer cleanup()

			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			var skill source.Skill
			if name == "" {
				if skill, err = pickSkill(out, skills); err != nil {
					return err
				}
			} else {
				var ok bool
				if skill, ok = source.Find(skills, name); !ok {
					printSkillTable(cmd.ErrOrStderr(), skills)
					return fmt.Errorf("skill %q not found in %s", name, src.Display)
				}
			}
			return installSkill(out, src, co, skill, targets, force, ref)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already installed")
	cmd.Flags().StringVar(&ref, "ref", "", "branch, tag, or commit to install from")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func overrideSuffix(override bool) string {
	if override {
		return " (override)"
	}
	return ""
}

// printTargets names the agents and skill directories a command will use.
func printTargets(w io.Writer, targets []config.Selection, override bool) {
	agents := make([]string, len(targets))
	dirs := make([]string, len(targets))
	for i, t := range targets {
		agents[i] = string(t.Agent)
		dirs[i] = t.Layout.SkillsDir
	}
	fmt.Fprintf(w, "Agents: %s%s\n", strings.Join(agents, ", "), overrideSuffix(override))
	fmt.Fprintf(w, "Skill paths: %s\n", strings.Join(dirs, ", "))
}

// installSkill copies one skill into every target's skills directory and records each install.
func installSkill(out io.Writer, src source.Source, co source.Checkout, skill source.Skill, targets []config.Selection, force bool, ref string) error {
	// Path safety only: install enforces no naming convention.
	if err := config.CheckSkillName(config.NamingLoose, skill.Name); err != nil {
		return err
	}
	for _, t := range targets {
		dir := t.SkillDir(skill.Name)
		if _, err := os.Stat(dir); err == nil && !force {
			ui.UserError("%s is already installed at %s/\n  use --force to reinstall", skill.Name, dir)
			return fmt.Errorf("already installed")
		}
	}
	fmt.Fprintf(out, "Installing %s from %s...\n", skill.Name, src.Display)
	for _, t := range targets {
		dir := t.SkillDir(skill.Name)
		written, err := replaceDir(dir, func(dst string) ([]string, error) {
			return source.CopyDir(filepath.Join(co.Dir, filepath.FromSlash(skill.Rel)), dst)
		})
		if err != nil {
			return fmt.Errorf("install %s: %w", skill.Name, err)
		}
		if err := recordInstall(dir, skill.Name, src, ref, co.Commit); err != nil {
			return err
		}
		for _, p := range written {
			fmt.Fprintf(out, "  -> %s/%s\n", dir, p)
		}
	}
	fmt.Fprintf(out, "\nOpen %s/SKILL.md to review.\n", targets[0].SkillDir(skill.Name))
	return nil
}

func lockKey(dir string) string { return filepath.ToSlash(filepath.Clean(dir)) }

func recordInstall(dir, name string, src source.Source, ref, commit string) error {
	files, err := source.HashDir(dir)
	if err != nil {
		return err
	}
	lf, err := lock.Load(".")
	if err != nil {
		return err
	}
	lf.Skills[lockKey(dir)] = lock.Entry{Name: name, Source: src.Raw, Ref: ref, Commit: commit, Files: files}
	return lock.Save(".", lf)
}

// replaceDir fills a staging directory next to targetDir, then swaps it in.
func replaceDir(targetDir string, fill func(dst string) ([]string, error)) ([]string, error) {
	parent := filepath.Dir(targetDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, fmt.Errorf("create skill parent directory: %w", err)
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(targetDir)+"-staging-")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	written, err := fill(staging)
	if err != nil {
		return nil, err
	}

	backup := staging + "-backup"
	existed := false
	if _, err := os.Stat(targetDir); err == nil {
		existed = true
		if err := os.Rename(targetDir, backup); err != nil {
			return nil, fmt.Errorf("stage existing skill directory: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect existing skill directory: %w", err)
	}
	if err := os.Rename(staging, targetDir); err != nil {
		if existed {
			if restoreErr := os.Rename(backup, targetDir); restoreErr != nil {
				return nil, fmt.Errorf("replace skill directory: %w (restore failed: %v)", err, restoreErr)
			}
		}
		return nil, fmt.Errorf("replace skill directory: %w", err)
	}
	if existed {
		_ = os.RemoveAll(backup)
	}
	return written, nil
}

func newSkillUpdateCmd() *cobra.Command {
	var all, force bool
	var flagAgent string

	cmd := &cobra.Command{
		Use:   "update [<name>]",
		Short: "Update installed skills from the source they were installed from",
		Long: `Update skills recorded in .hikma/lock.json. Each skill is updated from the
source it was installed from, which may differ from your configured registry.
Local edits are detected by file hash and block the update unless --force is used.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if !all && len(args) == 0 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill update <name>\n  hikma skill update --all")
			}
			if all && len(args) > 0 {
				return fmt.Errorf("--all and a skill name are mutually exclusive")
			}
			if all && flagAgent != "" {
				return fmt.Errorf("--all cannot be combined with --agent")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts at most one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			lf, err := lock.Load(".")
			if err != nil {
				return err
			}

			var keys []string
			if all {
				keys = lf.Keys()
				if len(keys) == 0 {
					fmt.Fprintln(out, "No installed skills recorded in .hikma/lock.json.")
					return nil
				}
			} else {
				targets, err := config.ResolveTargets(flagAgent)
				if err != nil {
					return fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
				}
				name := strings.TrimPrefix(strings.TrimSpace(args[0]), "@")
				if err := config.CheckSkillName(config.NamingLoose, name); err != nil {
					return err
				}
				for _, t := range targets {
					if k := lockKey(t.SkillDir(name)); hasEntry(lf, k) {
						keys = append(keys, k)
					}
				}
				if len(keys) == 0 {
					return fmt.Errorf("%q is not installed by hikma for the selected agents (no entry in .hikma/lock.json)", name)
				}
			}

			u := &updater{out: out, errOut: cmd.ErrOrStderr(), force: force, lf: lf, checkouts: map[string]*openCheckout{}}
			defer u.close()
			var updated, skipped int
			for _, key := range keys {
				if err := u.update(key); err != nil {
					ui.UserError("skipping %s: %s", key, err)
					skipped++
				} else {
					updated++
				}
			}
			if err := lock.Save(".", u.lf); err != nil {
				return err
			}
			if all {
				fmt.Fprintf(out, "\nDone: %d processed, %d skipped.\n", updated, skipped)
			}
			if skipped > 0 && !all {
				return fmt.Errorf("update failed")
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

	cmd.Flags().BoolVar(&all, "all", false, "update every skill recorded in .hikma/lock.json")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite local changes")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "agents, comma-separated (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

type openCheckout struct {
	src     source.Source
	co      source.Checkout
	cleanup func()
}

// updater shares one checkout per source and ref across a whole update run.
type updater struct {
	out, errOut io.Writer
	force       bool
	lf          lock.File
	checkouts   map[string]*openCheckout
}

func (u *updater) close() {
	for _, c := range u.checkouts {
		c.cleanup()
	}
}

func (u *updater) checkout(e lock.Entry) (*openCheckout, error) {
	k := e.Source + "@" + e.Ref
	if c, ok := u.checkouts[k]; ok {
		return c, nil
	}
	src, err := source.Parse(e.Source)
	if err != nil {
		return nil, err
	}
	co, cleanup, err := source.Resolve(src, e.Ref)
	if err != nil {
		return nil, err
	}
	c := &openCheckout{src: src, co: co, cleanup: cleanup}
	u.checkouts[k] = c
	return c, nil
}

func (u *updater) update(key string) error {
	e, ok := u.lf.Skills[key]
	if !ok {
		return fmt.Errorf("not installed by hikma (no entry in .hikma/lock.json); run 'hikma skill install --force' to track it")
	}
	localDir := filepath.FromSlash(key)
	if _, err := os.Stat(localDir); err != nil {
		return fmt.Errorf("%s is missing; reinstall it", localDir)
	}
	if !u.force {
		current, err := source.HashDir(localDir)
		if err != nil {
			return err
		}
		if a, m, r := source.DiffFiles(e.Files, current); len(a)+len(m)+len(r) > 0 {
			return fmt.Errorf("%s has local changes; use --force to overwrite them", localDir)
		}
	}

	c, err := u.checkout(e)
	if err != nil {
		return err
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
	if len(added)+len(modified)+len(removed) == 0 {
		fmt.Fprintf(u.out, "%s is up to date (%s @ %s)\n", e.Name, c.src.Display, shortCommit(c.co.Commit))
		e.Commit = c.co.Commit
		u.lf.Skills[key] = e
		return nil
	}

	fmt.Fprintf(u.out, "Updating %s from %s @ %s\n", e.Name, c.src.Display, shortCommit(c.co.Commit))
	printChanges(u.out, "added", added)
	printChanges(u.out, "modified", modified)
	printChanges(u.out, "removed", removed)
	if touchesScripts(added, modified, removed) {
		fmt.Fprintln(u.out, "  warning: scripts/ changed - review before use")
	}

	if _, err := replaceDir(localDir, func(dst string) ([]string, error) { return source.CopyDir(srcDir, dst) }); err != nil {
		return err
	}
	files, err := source.HashDir(localDir)
	if err != nil {
		return err
	}
	e.Files, e.Commit = files, c.co.Commit
	u.lf.Skills[key] = e
	return nil
}

func printChanges(w io.Writer, label string, files []string) {
	for _, f := range files {
		fmt.Fprintf(w, "  %-8s %s\n", label, f)
	}
}

func touchesScripts(groups ...[]string) bool {
	for _, g := range groups {
		for _, f := range g {
			if strings.HasPrefix(f, "scripts/") {
				return true
			}
		}
	}
	return false
}

func hasEntry(lf lock.File, key string) bool {
	_, ok := lf.Skills[key]
	return ok
}

// installedSkillCompletions returns skill names recorded in the lockfile.
func installedSkillCompletions() ([]string, cobra.ShellCompDirective) {
	lf, err := lock.Load(".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	seen := map[string]bool{}
	for _, e := range lf.Skills {
		seen[e.Name] = true
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}
