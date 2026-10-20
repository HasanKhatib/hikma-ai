package commands

import (
	"context"
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
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
// ctx stops the clone when it is cancelled.
func openSource(ctx context.Context, w io.Writer, repo, flagRegistry, ref string) (source.Source, source.Checkout, func(), error) {
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
	co, cleanup, err := source.ResolveContext(ctx, src, ref)
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

// noInput is the global --no-input flag. It is bound at parse time, so it is
// already set when argument validation runs.
var noInput bool

// isInteractive reports whether a command may prompt: stdin is a terminal and
// --no-input was not passed. Every prompt site gates on it, so --no-input makes
// each one fail with the flag to pass instead of asking.
func isInteractive() bool {
	return !noInput && stdinIsTerminal()
}

// stdinIsTerminal is a variable so tests can pretend to be on a terminal.
var stdinIsTerminal = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// resolveSourceFolder understands a source that names a folder or a branch inside
// a GitHub repository (owner/repo/skills/pdf, or a /tree/<ref>/<folder> URL). It
// returns the plain source to read, the ref to read it at, and the folder to
// narrow to. Anything else comes back unchanged.
func resolveSourceFolder(cmd *cobra.Command, repo, ref string) (string, string, string, error) {
	if repo == "" {
		return repo, ref, "", nil
	}
	sr, err := source.ParseRef(repo)
	if err != nil {
		return "", "", "", err
	}
	if sr.Tail == "" && sr.Path == "" {
		return repo, ref, "", nil
	}
	if sr.Tail != "" {
		if ref != "" {
			return "", "", "", usageError(fmt.Errorf("the URL already names the branch: drop --ref, or use owner/repo with --ref"))
		}
		var refs []string
		if strings.Contains(sr.Tail, "/") {
			refs, _ = source.ListRefs(sr.Source)
		}
		var guessed bool
		ref, sr.Path, guessed = sr.ResolveTail(refs)
		if guessed {
			fmt.Fprintf(cmd.ErrOrStderr(), "note: could not confirm %q is a branch or tag of %s; reading it as the ref\n", ref, sr.Source.Display)
		}
	}
	return sr.Source.Raw, ref, sr.Path, nil
}

// narrowToFolder keeps the skills at or under folder. A folder that holds no skill
// hikma installs from is an error that points at import, which finds more layouts.
func narrowToFolder(skills []source.Skill, folder string, src source.Source) ([]source.Skill, error) {
	if folder == "" {
		return skills, nil
	}
	var out []source.Skill
	for _, s := range skills {
		if s.Rel == folder || strings.HasPrefix(s.Rel, folder+"/") {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no skill to install at %s in %s\ninstall reads skills/<name>/, <name>/, or a SKILL.md at the root; for another layout, bring it into your registry with: hikma skill import %s/%s", folder, src.Display, src.Display, folder)
	}
	return out, nil
}

// pickSkill asks which skill of a source to use. installed marks the ones already
// installed with "[installed]"; it may be nil.
func pickSkill(w io.Writer, skills []source.Skill, installed func(string) bool) (source.Skill, error) {
	if len(skills) == 0 {
		return source.Skill{}, fmt.Errorf("no skills found in this source")
	}
	if !isInteractive() {
		printSkillTableMarked(w, skills, installed)
		return source.Skill{}, fmt.Errorf("choose a skill: hikma skill install <owner/repo> <name>")
	}
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	labels := markLabels(names, installed, "installed")
	for i, s := range skills {
		if s.Description != "" {
			labels[i] += " - " + truncate(s.Description, 70)
		}
	}
	idx, err := prompter.Select("Skill:", labels, 0)
	if err != nil {
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

func printSkillTable(w io.Writer, skills []source.Skill) { printSkillTableMarked(w, skills, nil) }

// printSkillTableMarked prints the skills with "[installed]" in front of the ones
// marked as installed. When none is, the table is the plain one.
func printSkillTableMarked(w io.Writer, skills []source.Skill, installed func(string) bool) {
	if len(skills) == 0 {
		fmt.Fprintln(w, "No skills found.")
		return
	}
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	marked := markLabels(names, installed, "installed")
	lead := ""
	if anyDiffers(marked, names) {
		lead = strings.Repeat(" ", len("[installed] "))
	}
	const nameW, descW = 25, 50
	fmt.Fprintf(w, "%s%-*s  %-*s  %s\n", lead, nameW, "NAME", descW, "DESCRIPTION", "OWNER")
	for i, s := range skills {
		name := s.Name
		if lead != "" {
			name = marked[i]
			fmt.Fprintf(w, "%s  %-*s  %s\n", ui.Pad(name, len(lead)+nameW), descW, truncate(s.Description, descW), s.Owner)
			continue
		}
		fmt.Fprintf(w, "%-*s  %-*s  %s\n", nameW, name, descW, truncate(s.Description, descW), s.Owner)
	}
}

func anyDiffers(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}

func newSkillListCmd(flagRegistry *string) *cobra.Command {
	var asJSON, installed, global bool
	cmd := &cobra.Command{
		Use:   "list [<owner/repo>]",
		Short: "List skills in the configured registry or in any repo, or what is installed",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if global && !installed {
				return usageError(fmt.Errorf("--global only applies with --installed: add --installed, or drop --global"))
			}
			if installed {
				if len(args) > 0 {
					return usageError(fmt.Errorf("--installed lists this repository's skills and takes no source: drop %q, or drop --installed to list a source", args[0]))
				}
				return listInstalled(cmd, asJSON, global)
			}
			status := cmd.OutOrStdout()
			if asJSON {
				status = cmd.ErrOrStderr()
			}
			repo := ""
			if len(args) == 1 {
				repo = args[0]
			}
			src, co, cleanup, err := openSource(cmd.Context(), status, repo, *flagRegistry, "")
			if err != nil {
				return err
			}
			defer cleanup()
			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			if asJSON {
				if skills == nil {
					skills = []source.Skill{}
				}
				return printJSON(cmd.OutOrStdout(), struct {
					Version int            `json:"version"`
					Skills  []source.Skill `json:"skills"`
				}{jsonVersion, skills})
			}
			var installed func(string) bool
			if targets, err := config.ResolveTargets(""); err == nil {
				installed = func(n string) bool { return installedIn(projectScope(), targets, n) }
			}
			printSkillTableMarked(cmd.OutOrStdout(), skills, installed)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&installed, "installed", false, "list the skills installed in this repository and in your home directory instead")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "with --installed, list only your home directory's skills")
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
			src, co, cleanup, err := openSource(cmd.Context(), status, repo, *flagRegistry, ref)
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
				return printJSON(cmd.OutOrStdout(), struct {
					Version int          `json:"version"`
					Skill   source.Skill `json:"skill"`
				}{jsonVersion, skill})
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
	s, err := pickSkill(w, skills, nil)
	return s.Name, err
}

func newSkillInstallCmd(flagRegistry *string) *cobra.Command {
	var force, global, yes bool
	var flagAgent, ref string
	var flagSkills []string

	cmd := &cobra.Command{
		Use:   "install [<owner/repo>] [<name>]",
		Short: "Install a skill from the configured registry or any repo",
		Long: `Install a skill into the active agent's skill directory.

  hikma skill install my-skill               from your configured registry
  hikma skill install owner/repo my-skill    from any GitHub repo
  hikma skill install owner/repo             pick from the skills in that repo
  hikma skill install ./local/path my-skill  from a local checkout
  hikma skill install owner/repo --skill a --skill b
                                             several skills from one checkout

Sources can be GitHub owner/repo, any git URL, or a local path. Use --ref to pin
a branch, tag, or commit. Skill names are not restricted on install.

With --global the skill goes into your home directory (for example
~/.claude/skills) and is tracked in ~/.hikma/lock.json. hikma prints the
absolute paths and asks first; --yes skips the question. It never writes
through a symlinked folder.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(flagSkills) > 0 {
				// With --skill the skills are named by the flag; RunE explains the
				// mix-ups (a name as well) better than a count can.
				return cobra.MaximumNArgs(2)(cmd, args)
			}
			return cobra.RangeArgs(1, 2)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(flagSkills) > 0 && (len(args) == 2 || (len(args) == 1 && !source.LooksLikeSource(args[0]))) {
				return usageError(fmt.Errorf("name the skills with --skill or as an argument, not both: drop %q, or drop --skill", args[len(args)-1]))
			}
			sc, err := scopeFor(global)
			if err != nil {
				return err
			}
			targets, err := config.ResolveTargets(flagAgent)
			if err != nil {
				return fmt.Errorf("%w\nrun 'hikma doctor' to check your environment", err)
			}
			printTargets(out, targets, flagAgent != "")

			repo, name := sourceArgs(args)
			repo, ref, folder, err := resolveSourceFolder(cmd, repo, ref)
			if err != nil {
				return err
			}
			src, co, cleanup, err := openSource(cmd.Context(), out, repo, *flagRegistry, ref)
			if err != nil {
				return err
			}
			defer cleanup()

			skills, err := discoverSkills(src, co)
			if err != nil {
				return err
			}
			if skills, err = narrowToFolder(skills, folder, src); err != nil {
				return err
			}
			if len(flagSkills) > 0 {
				return installMany(cmd, sc, src, co, skills, flagSkills, targets, force, yes, ref)
			}
			var skill source.Skill
			if name == "" && folder != "" && len(skills) == 1 {
				skill = skills[0]
			} else if name == "" {
				if skill, err = pickSkill(out, skills, func(n string) bool { return installedIn(sc, targets, n) }); err != nil {
					return err
				}
			} else {
				var ok bool
				if skill, ok = source.Find(skills, name); !ok {
					printSkillTable(cmd.ErrOrStderr(), skills)
					return fmt.Errorf("skill %q not found in %s", name, src.Display)
				}
			}
			keys := make([]string, len(targets))
			for i, t := range targets {
				keys[i] = sc.key(t, skill.Name)
			}
			if err := sc.guardWrite(keys...); err != nil {
				return err
			}
			for _, w := range sc.shadowWarnings([]string{skill.Name}) {
				ui.PrintWarning(out, w)
			}
			if err := sc.confirmWrite(cmd, yes, keys); err != nil {
				return err
			}
			return installSkill(sc, out, src, co, skill, targets, force, ref)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already installed")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "use your home directory instead of this project (~/.hikma/lock.json)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation before changing your home directory")
	cmd.Flags().StringVar(&ref, "ref", "", "branch, tag, or commit to install from (--pin is an alias, as in gh skill)")
	cmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "pin" {
			name = "ref"
		}
		return pflag.NormalizedName(name)
	})
	cmd.Flags().StringArrayVar(&flagSkills, "skill", nil, "a skill to install; repeat it to install several from one checkout")
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
	warnOverlappingTargets(w, targets)
}

// warnOverlappingTargets notes when one agent loads from two of the folders a
// skill is written to, so it may use either copy. The install still proceeds.
func warnOverlappingTargets(w io.Writer, targets []config.Selection) {
	dirs := map[string]bool{}
	for _, t := range targets {
		dirs[t.Layout.SkillsDir] = true
	}
	warned := map[string]bool{}
	for _, t := range targets {
		for _, other := range t.Reads {
			pair := t.Layout.SkillsDir + "|" + other
			if dirs[other] && other != t.Layout.SkillsDir && !warned[pair] {
				warned[pair] = true
				fmt.Fprintf(w, "%s skills are written to both %s and %s, and an agent you selected reads both, so it may use either copy\n", ui.Warning(w), t.Layout.SkillsDir, other)
			}
		}
	}
}

// installSkill copies one skill into every target's skills directory and records each install.
func installSkill(sc scope, out io.Writer, src source.Source, co source.Checkout, skill source.Skill, targets []config.Selection, force bool, ref string) error {
	// Path safety only: install enforces no naming convention.
	if err := config.CheckSkillName(config.NamingLoose, skill.Name); err != nil {
		return err
	}
	for _, t := range targets {
		dir := sc.skillDir(t, skill.Name)
		if _, err := os.Stat(dir); err == nil && !force {
			return fmt.Errorf("%s is already installed at %s/\n  use --force to reinstall", skill.Name, dir)
		}
		if err := requireSkillFolder(dir); err != nil {
			return err
		}
		if err := sc.guardWrite(sc.key(t, skill.Name)); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "Installing %s from %s...\n", skill.Name, src.Display)
	for _, t := range targets {
		dir := sc.skillDir(t, skill.Name)
		written, err := replaceDir(dir, func(dst string) ([]string, error) {
			return source.CopyDir(filepath.Join(co.Dir, filepath.FromSlash(skill.Rel)), dst)
		})
		if err != nil {
			return fmt.Errorf("install %s: %w", skill.Name, err)
		}
		if err := recordInstall(sc, sc.key(t, skill.Name), skill.Name, src, ref, co.Commit); err != nil {
			return err
		}
		fmt.Fprintf(out, "  -> %s/ (%d files)\n", dir, len(written))
	}
	fmt.Fprintf(out, "\nOpen %s/SKILL.md to review.\n", sc.skillDir(targets[0], skill.Name))
	return nil
}

func lockKey(dir string) string { return filepath.ToSlash(filepath.Clean(dir)) }

func recordInstall(sc scope, key, name string, src source.Source, ref, commit string) error {
	files, err := source.HashDir(sc.path(key))
	if err != nil {
		return err
	}
	lf, err := sc.loadLock()
	if err != nil {
		return err
	}
	lf.Skills[key] = lock.Entry{Name: name, Source: src.Raw, Ref: ref, Commit: commit, Files: files}
	return sc.saveLock(lf)
}

// requireSkillFolder refuses to replace an existing folder that holds no
// SKILL.md. Custom agents and the lockfile are committed, so a definition pointing at a
// source folder must not let --force delete it as if it were a skill.
func requireSkillFolder(dir string) error {
	if !dirExists(dir) {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		return fmt.Errorf("refusing to replace %s/: it is not a skill folder (no SKILL.md); move or delete it first", filepath.ToSlash(dir))
	}
	return nil
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
	var all, force, yes, global bool
	var flagAgent string

	cmd := &cobra.Command{
		Annotations: map[string]string{ownArgErrors: "true"},
		Use:         "update [<name>]",
		Short:       "Update installed skills from the source they were installed from",
		Long: `Update skills recorded in .hikma/lock.json. Each skill is updated from the
source it was installed from, which may differ from your configured registry.
Local edits are detected by file hash and block the update unless --force is used.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if !all && len(args) == 0 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill update <name>\n  hikma skill update --all")
			}
			if all && len(args) > 0 {
				return usageError(fmt.Errorf("--all and a skill name are mutually exclusive: pass one or the other"))
			}
			if all && flagAgent != "" {
				return usageError(fmt.Errorf("--all cannot be combined with --agent: name a skill to limit the agents, or drop --agent"))
			}
			if len(args) > 1 {
				return fmt.Errorf("update takes one skill name, got %d: name one, or use --all", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			sc, err := scopeFor(global)
			if err != nil {
				return err
			}
			lf, err := sc.loadLock()
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
					if k := sc.key(t, name); hasEntry(lf, k) {
						keys = append(keys, k)
					}
				}
				if len(keys) == 0 {
					for _, t := range targets {
						if _, gh := source.ReadProvenance(sc.skillDir(t, name)); gh {
							return fmt.Errorf("%q was installed by gh skill and is not tracked by hikma; run 'hikma skill adopt %s' to track it", name, name)
						}
					}
					return fmt.Errorf("%q is not installed by hikma for the selected agents (no entry in .hikma/lock.json)", name)
				}
			}

			if err := sc.confirmWrite(cmd, yes, keys); err != nil {
				return err
			}
			u := &updater{sc: sc, out: out, errOut: cmd.ErrOrStderr(), force: force, yes: yes, lf: lf, checkouts: map[string]*openCheckout{}}
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
			if err := sc.saveLock(u.lf); err != nil {
				return err
			}
			if all {
				fmt.Fprintf(out, "\nDone: %d processed, %d skipped.\n", updated, skipped)
				if gh := ghManagedUntracked(sc); len(gh) > 0 {
					fmt.Fprintf(out, "Not tracked by hikma (installed by gh skill): %s. Run 'hikma skill adopt <name>' to include them.\n", adoptHint(gh))
				}
			}
			if skipped > 0 && !all {
				return fmt.Errorf("%d skill(s) were skipped; see the messages above", skipped)
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

	cmd.Flags().BoolVar(&all, "all", false, "update every skill recorded in .hikma/lock.json")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite local changes")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply updates that change scripts/, and changes to your home directory, without asking")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "use your home directory instead of this project (~/.hikma/lock.json)")
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
	sc          scope
	out, errOut io.Writer
	force, yes  bool
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
	if ok {
		dirs, err := layoutSkillDirs()
		if err != nil {
			return err
		}
		if err := validateLockEntry(key, e, dirs); err != nil {
			return err
		}
		if err := u.sc.guardWrite(key); err != nil {
			return err
		}
	}
	if !ok {
		if _, gh := source.ReadProvenance(u.sc.path(key)); gh {
			return fmt.Errorf("installed by gh skill and not tracked by hikma; run 'hikma skill adopt %s' to track it", path.Base(key))
		}
		return fmt.Errorf("not installed by hikma (no entry in .hikma/lock.json); run 'hikma skill adopt %s --source <owner/repo>' or 'hikma skill install --force'", path.Base(key))
	}
	localDir := u.sc.path(key)
	if _, err := os.Stat(localDir); err != nil {
		return fmt.Errorf("%s is missing; reinstall it", localDir)
	}
	current, err := source.HashDir(localDir)
	if err != nil {
		return err
	}
	a, m, r := source.DiffFiles(e.Files, current)
	locallyEdited := len(a)+len(m)+len(r) > 0
	if locallyEdited && !u.force {
		return fmt.Errorf("%s has local changes; use --force to overwrite them", localDir)
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
	if len(added)+len(modified)+len(removed) == 0 && !locallyEdited {
		fmt.Fprintf(u.out, "%s is up to date (%s @ %s)\n", e.Name, c.src.Display, shortCommit(c.co.Commit))
		e.Commit = c.co.Commit
		u.lf.Skills[key] = e
		return nil
	}

	if locallyEdited && len(added)+len(modified)+len(removed) == 0 {
		fmt.Fprintf(u.out, "Restoring %s to %s @ %s (local changes overwritten)\n", e.Name, c.src.Display, shortCommit(c.co.Commit))
		printChanges(u.out, "reverted", append(append(append([]string{}, a...), m...), r...))
	} else {
		fmt.Fprintf(u.out, "Updating %s from %s @ %s\n", e.Name, c.src.Display, shortCommit(c.co.Commit))
	}
	printChanges(u.out, "added", added)
	printChanges(u.out, "modified", modified)
	printChanges(u.out, "removed", removed)
	if touchesScripts(added, modified, removed) {
		fmt.Fprintf(u.out, "  %s scripts/ changed - review before use\n", ui.Warning(u.out))
		if !u.yes {
			if !isInteractive() {
				return fmt.Errorf("scripts/ changed; review with 'hikma skill diff %s' and re-run with --yes to apply", e.Name)
			}
			ok, err := prompter.Confirm(fmt.Sprintf("Apply this update to %s, including the scripts/ changes?", e.Name), false)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("update of %s declined", e.Name)
			}
		}
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
func installedSkillCompletions(sc scope) ([]string, cobra.ShellCompDirective) {
	lf, err := sc.loadLock()
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
