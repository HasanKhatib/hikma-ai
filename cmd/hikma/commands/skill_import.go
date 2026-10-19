package commands

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
)

// importDoc is the --json document of `skill import`: the plan, and the result
// when the import ran.
type importDoc struct {
	Version int          `json:"version"`
	DryRun  bool         `json:"dryRun"`
	Source  importSource `json:"source"`
	Skill   importSkill  `json:"skill"`
	Files   []importFile `json:"files"`
	Skipped []skipDoc    `json:"skipped"`
	Changes []changeDoc  `json:"changes"`
	License struct {
		Value *string `json:"value"`
		Note  string  `json:"note"`
	} `json:"license"`
	Review struct {
		Scripts      []string `json:"scripts"`
		Binaries     []string `json:"binaries"`
		OutsideLinks []string `json:"outsideLinks"`
		RunLines     []string `json:"runLines"`
	} `json:"review"`
	Warnings []string      `json:"warnings"`
	Target   *importTarget `json:"target"`
	Result   *importResult `json:"result"`
}

type importSource struct {
	Source string  `json:"source"`
	Ref    *string `json:"ref"`
	Commit *string `json:"commit"`
	Tree   *string `json:"tree"`
	Path   string  `json:"path"`
	ReadAs string  `json:"readFrom"`
}

type importSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type importFile struct {
	Path       string `json:"path"`
	Size       int    `json:"size"`
	Executable bool   `json:"executable"`
}

type skipDoc struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type changeDoc struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type importTarget struct {
	Registry string `json:"registry"`
	Origin   string `json:"origin"`
	How      string `json:"how"`
}

type importResult struct {
	URL  string `json:"url"`
	Mode string `json:"mode"` // "pull-request" or "commit"
}

var licenseFile = regexp.MustCompile(`(?i)^(licen[cs]e|copying)(\.[a-z]+)?$`)

func newSkillImportCmd(flagRegistry *string) *cobra.Command {
	var ref, as string
	var force, pr, yes, dryRun, asJSON bool

	cmd := &cobra.Command{
		Use:   "import <source> [<name>]",
		Short: "Bring a skill from anywhere into your registry",
		Long: `Copy a skill you found into your own registry, after a review of what is in it.

  hikma skill import owner/repo code-review
  hikma skill import https://github.com/owner/repo/tree/main/skills/code-review
  hikma skill import owner/repo/skills/code-review   an exact path skips discovery
  hikma skill import owner/repo                      pick from the skills in that repo
  hikma skill import ./local/checkout my-skill

Hikma shows the files, the licence, and anything worth reading first (scripts,
executables, binaries, links outside the folder), then asks before it writes.

The skill is published like 'hikma skill push': committed to the registry's default
branch when you can write, as a pull request when the branch requires one or you
cannot write (through your fork), and always as a pull request with --pr.

Files are copied byte for byte, except that metadata.github-* is removed from
SKILL.md (so 'gh skill update' cannot bypass your registry) and --as rewrites the
name line. Where the skill came from is recorded in skills/<name>/.hikma-origin.json.
An existing skill is never overwritten without --force.

Sources are github.com repositories and local folders. --dry-run shows the plan and
touches nothing, not even the registry.`,
		Args: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				return fmt.Errorf("missing argument: say where the skill is\n\nUsage:\n  hikma skill import <source> [<name>]\n\nExample: hikma skill import owner/repo code-review")
			case len(args) > 2:
				return fmt.Errorf("unexpected arguments %q\n\nUsage:\n  hikma skill import <source> [<name>]", strings.Join(args[2:], " "))
			case !source.LooksLikeSource(args[0]):
				return fmt.Errorf("%q is a skill name, not a source: import needs to know where to read it\n\nTo import it from a repository: hikma skill import <owner/repo> %s\nTo install it from your registry: hikma skill install %s", args[0], args[0], args[0])
			}
			return nil
		},
		Annotations: map[string]string{ownArgErrors: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON && !dryRun && !yes {
				return usageError(fmt.Errorf("--json never asks a question: add --yes to import, or --dry-run to see the plan"))
			}
			name := ""
			if len(args) == 2 {
				name = strings.TrimPrefix(strings.TrimSpace(args[1]), "@")
			}
			return runImport(cmd, importArgs{
				arg: args[0], name: name, ref: ref, as: as, force: force, pr: pr,
				yes: yes, dryRun: dryRun, asJSON: asJSON, registry: *flagRegistry,
			})
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "branch, tag, or commit to read from (for a URL, put it in the URL)")
	cmd.Flags().StringVar(&as, "as", "", "import under a different name (rewrites the name line in SKILL.md)")
	cmd.Flags().BoolVar(&force, "force", false, "replace a skill of the same name that the registry already has")
	cmd.Flags().BoolVar(&pr, "pr", false, "open a pull request even if you can write to the registry")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation (required without a terminal)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be imported and write nothing")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the plan (and the result) as JSON on stdout (needs --yes or --dry-run)")
	return cmd
}

type importArgs struct {
	arg, name, ref, as, registry string
	force, pr, yes, dryRun       bool
	asJSON                       bool
}

func runImport(cmd *cobra.Command, a importArgs) error {
	out, errw := cmd.OutOrStdout(), cmd.ErrOrStderr()
	human := out // the prose; with --json it moves to stderr so stdout is one document
	if a.asJSON {
		human = errw
	}

	sr, err := source.ParseRef(a.arg)
	if err != nil {
		return err
	}
	switch sr.Source.Kind {
	case source.KindGit:
		return fmt.Errorf("only github.com repositories and local folders can be imported (%s)\nclone it, then import from the folder: git clone <url> <dir> && hikma skill import <dir> <name>", sr.Source.Display)
	case source.KindLocal:
		if a.ref != "" {
			return usageError(fmt.Errorf("--ref only applies to a GitHub source, not a local folder"))
		}
	}
	if sr.Tail != "" && a.ref != "" {
		return usageError(fmt.Errorf("the URL already names the branch: drop --ref, or use owner/repo with --ref"))
	}

	// The registry is looked up first so a missing one fails before anything is downloaded.
	registry, regOrigin, regErr := resolvePushRegistry(a.registry)
	if regErr != nil && !a.dryRun {
		return regErr
	}

	ref := a.ref
	if sr.Tail != "" {
		var refs []string
		if strings.Contains(sr.Tail, "/") {
			refs, _ = source.ListRefs(sr.Source)
		}
		var guessed bool
		ref, sr.Path, guessed = sr.ResolveTail(refs)
		if guessed {
			fmt.Fprintf(errw, "note: could not confirm %q is a branch or tag of %s; reading it as the ref\n", ref, sr.Source.Display)
		}
	}

	fmt.Fprintf(errw, "Reading %s%s...\n", sr.Source.Display, suffixIf(ref != "", " at "+ref))
	reader, cleanup, err := source.OpenReader(sr.Source, ref)
	if err != nil {
		return err
	}
	defer cleanup()

	if sr.Source.Kind == source.KindLocal {
		if folder, inside := agentFolderOf(sr.Source.Path); inside {
			return fmt.Errorf("%s is inside your agent folder (%s): it is already a local skill, not something to import\npublish it to your registry with: hikma skill push %s", sr.Source.Path, folder, filepath.Base(sr.Source.Path))
		}
	}

	fallback := sr.Source.Repo
	if fallback == "" {
		fallback = filepath.Base(sr.Source.Path)
	}
	cands, err := source.FindCandidates(reader, sr.Path, fallback)
	if err != nil {
		return err
	}
	cand, err := chooseCandidate(cmd, human, cands, a, sr)
	if err != nil {
		return err
	}

	finalName := cand.Name
	if a.as != "" {
		finalName = a.as
	}
	if err := checkSkillName(finalName); err != nil {
		return err
	}

	prep, err := prepareImport(reader, sr, ref, cand, finalName, a.as != "")
	if err != nil {
		return err
	}

	issues, err := validateImport(prep, finalName)
	if err != nil {
		return err
	}
	doc := prep.doc(a.dryRun)
	for _, i := range issues {
		if i.Severity == source.SeverityWarning {
			doc.Warnings = append(doc.Warnings, i.Message)
		}
	}
	printImportReview(human, prep, doc)
	if source.HasErrors(issues) {
		printIssuesFor(errw, finalName, issues)
		msg := "validation failed: the skill does not pass the registry's checks (above). Hikma changes only the name line and metadata.github-*, so the problem is in the upstream skill"
		if a.as == "" {
			msg += "\nif only the name is wrong, import it under another one: hikma skill import " + a.arg + " --as <name>"
		}
		return fmt.Errorf("%s", msg)
	}
	if len(issues) > 0 {
		printIssuesFor(human, finalName, issues)
	}

	how := "committed to the default branch, or as a pull request if you cannot write or the branch requires one (--pr always opens one)"
	if regErr == nil {
		doc.Target = &importTarget{Registry: registry.FullName(), Origin: regOrigin, How: how}
	}
	if a.dryRun {
		if regErr == nil {
			fmt.Fprintf(human, "\nWould import into registry %s (%s) as %s.\n", registry.FullName(), regOrigin, how)
		} else {
			fmt.Fprintf(human, "\nNo registry is configured, so this only reviews the skill: hikma config set registry <owner/repo>\n")
		}
		fmt.Fprintln(human, "Dry run: nothing was written, and the registry was not contacted.")
		if a.asJSON {
			return printJSON(out, doc)
		}
		return nil
	}

	if result := preflightCheck(); !result.OK() {
		return preflightError(result)
	}
	if err := checkCollision(human, registry, finalName, prep, a); err != nil {
		return err
	}
	how = describePush(registry, a.pr)
	doc.Target.How = how
	if err := confirmImport(cmd, human, prep, registry, regOrigin, how, a.yes); err != nil {
		return err
	}

	plan := pushPlan{
		upstream: registry, name: finalName, files: prep.pushFiles(finalName), viaPR: a.pr,
		message:    fmt.Sprintf("feat(skills): import %s from %s", finalName, prep.origin.Source),
		prBody:     importPRBody(prep, finalName),
		branch:     "import/" + finalName,
		replace:    a.force,
		directHint: fmt.Sprintf("import it through a pull request instead: hikma skill import %s --pr", a.arg),
	}
	url, err := pushSkill(human, plan)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}
	mode := "commit"
	if strings.Contains(url, "/pull/") {
		mode = "pull-request"
		fmt.Fprintf(human, "\nImported %s into a pull request. Review and merge it in %s, then install it anywhere with: hikma skill install %s\n", finalName, registry.FullName(), finalName)
	} else if url != "" {
		fmt.Fprintf(human, "\nImported %s into %s. Install it anywhere with: hikma skill install %s\n", finalName, registry.FullName(), finalName)
	}
	doc.Result = &importResult{URL: url, Mode: mode}
	if a.asJSON {
		return printJSON(out, doc)
	}
	return nil
}

func suffixIf(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}

// chooseCandidate picks one skill: by name or path, the only one there is, a
// choice in a terminal, or else a tab-separated list and an error.
func chooseCandidate(cmd *cobra.Command, w io.Writer, cands []source.Candidate, a importArgs, sr source.SkillRef) (source.Candidate, error) {
	if len(cands) == 0 {
		where := sr.Source.Display
		if sr.Path != "" {
			where += "/" + sr.Path
		}
		return source.Candidate{}, fmt.Errorf("no skills found in %s\nlooked for SKILL.md at the root, in <name>/, skills/<name>/, skills/<scope>/<name>/ and plugins/<scope>/skills/<name>/ (folders that start with a dot are skipped unless you name them)\nif the skill is somewhere else, give its exact path: hikma skill import owner/repo/path/to/skill", where)
	}
	if a.name != "" {
		c, err := source.SelectCandidate(cands, a.name)
		if err != nil {
			printCandidates(cmd.ErrOrStderr(), cands)
			return source.Candidate{}, fmt.Errorf("%v in %s", err, sr.Source.Display)
		}
		return c, nil
	}
	if len(cands) == 1 {
		return cands[0], nil
	}
	if !isInteractive() {
		printCandidates(cmd.OutOrStdout(), cands)
		return source.Candidate{}, usageError(fmt.Errorf("%d skills found: pass a name or an exact path, for example: hikma skill import %s %s", len(cands), a.arg, cands[0].Name))
	}
	labels := make([]string, len(cands))
	for i, c := range cands {
		labels[i] = c.Name
		if c.Description != "" {
			labels[i] += " - " + truncate(c.Description, 60)
		}
		labels[i] += "  (" + c.Dir + ")"
	}
	idx, err := prompter.Select("Skill to import:", labels, 0)
	if err != nil {
		return source.Candidate{}, err
	}
	return cands[idx], nil
}

// printCandidates writes name, path, and description, one skill per line and
// tab-separated, so a script can read it.
func printCandidates(w io.Writer, cands []source.Candidate) {
	for _, c := range cands {
		fmt.Fprintf(w, "%s\t%s\t%s\n", c.Name, c.Dir, strings.ReplaceAll(c.Description, "\n", " "))
	}
}

// agentFolderOf reports whether abs is inside one of this project's agent skill folders.
func agentFolderOf(abs string) (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	folders := map[string]bool{}
	for _, ag := range config.ValidAgents {
		l := config.LayoutFor(ag)
		folders[l.SkillsDir] = true
		for _, f := range l.AlsoReads {
			folders[f] = true
		}
	}
	if targets, err := config.ResolveTargets(""); err == nil {
		for _, t := range targets {
			folders[t.Layout.SkillsDir] = true
		}
	}
	for f := range folders {
		if rel == f || strings.HasPrefix(rel, f+"/") {
			return f, true
		}
	}
	return "", false
}

// importPrep is everything known about an import before anything is written.
type importPrep struct {
	cand       source.Candidate
	sr         source.SkillRef
	files      []source.File // as they will be stored
	origin     source.Origin
	skipped    []source.Skipped
	review     source.Review
	licenseRaw string
	licenseMsg string
	notes      []string // warnings that are not validation issues
	readFrom   string
}

func prepareImport(r source.Reader, sr source.SkillRef, ref string, cand source.Candidate, name string, rename bool) (importPrep, error) {
	loaded, err := source.LoadSkill(r, cand.Dir)
	if err != nil {
		return importPrep{}, err
	}
	p := importPrep{cand: cand, sr: sr, skipped: loaded.Skipped, readFrom: r.Describe()}
	commit, tree, why := r.Identity(cand.Dir)
	p.origin = source.NewOrigin(source.OriginSource(sr.Source), ref, commit, tree, cand.Dir, "")
	if why != "" {
		p.notes = append(p.notes, "the commit and tree are not recorded: "+why)
	}

	var changes []source.Change
	hasSkillMD := false
	for _, f := range loaded.Files {
		if f.Path != "SKILL.md" {
			continue
		}
		hasSkillMD = true
		original := source.Hash(f.Data)
		p.licenseRaw = source.FrontmatterString(f.Data, "license")
		data, removed, err := source.StripGitHubMetadata(f.Data)
		if err != nil {
			return importPrep{}, err
		}
		if len(removed) > 0 {
			changes = append(changes, source.Change{File: "SKILL.md", Reason: "strip-github-metadata", Detail: "removed " + strings.Join(removed, ", "), Upstream: original})
		}
		if rename {
			if data, err = source.SetSkillName(data, name); err != nil {
				return importPrep{}, err
			}
			changes = append(changes, source.Change{File: "SKILL.md", Reason: "rename", Detail: fmt.Sprintf("name: %s -> %s", cand.Name, name), Upstream: original})
		}
		for i := range loaded.Files {
			if loaded.Files[i].Path == "SKILL.md" {
				loaded.Files[i].Data = data
			}
		}
	}
	if !hasSkillMD {
		return importPrep{}, fmt.Errorf("%s has no SKILL.md", cand.Dir)
	}
	p.files = loaded.Files
	p.origin.Changes = append(p.origin.Changes, changes...)
	for _, f := range p.files {
		p.origin.Files[f.Path] = source.Hash(f.Data)
	}
	if p.licenseRaw != "" {
		s := p.licenseRaw
		p.origin.License = &s
	}
	p.review = source.ReviewFiles(p.files)
	p.licenseMsg = p.findLicense(r)
	return p, nil
}

// findLicense says where the licence is, or warns that there is none.
func (p *importPrep) findLicense(r source.Reader) string {
	if p.licenseRaw != "" {
		return fmt.Sprintf("%s (license in SKILL.md)", p.licenseRaw)
	}
	for _, f := range p.files {
		if !strings.Contains(f.Path, "/") && licenseFile.MatchString(f.Path) {
			p.review.LicenseSrc = f.Path
			return "a " + f.Path + " file is in the skill folder"
		}
	}
	if entries, err := r.Entries(""); err == nil {
		for _, e := range entries {
			if !strings.Contains(e.Path, "/") && licenseFile.MatchString(e.Path) {
				p.review.LicenseSrc = e.Path
				return "the repository has a " + e.Path + " at its root; it is not copied into the skill"
			}
		}
	}
	p.notes = append(p.notes, "no licence found: you may not have the right to copy and share this skill")
	return "none found"
}

// validateImport runs the registry's own checks on the skill as it will be stored.
func validateImport(p importPrep, name string) ([]source.Issue, error) {
	tmp, err := os.MkdirTemp("", "hikma-import-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tmp)
	dir := filepath.Join(tmp, name)
	for _, f := range p.files {
		dst := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dst, f.Data, 0o644); err != nil {
			return nil, err
		}
	}
	return validateSkillDirSeverity(dir, name, false, source.SeverityWarning)
}

func (p importPrep) pushFiles(name string) []pushFile {
	var files []pushFile
	for _, f := range p.files {
		files = append(files, pushFile{path: "skills/" + name + "/" + f.Path, content: f.Data, exec: f.Exec})
	}
	if data, err := p.origin.Marshal(); err == nil {
		files = append(files, pushFile{path: "skills/" + name + "/" + source.OriginFile, content: data})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files
}

func (p importPrep) doc(dry bool) importDoc {
	d := importDoc{Version: jsonVersion, DryRun: dry, Files: []importFile{}, Skipped: []skipDoc{}, Changes: []changeDoc{}, Warnings: append([]string{}, p.notes...)}
	d.Source = importSource{Source: p.origin.Source, Ref: p.origin.Ref, Commit: p.origin.Commit, Tree: p.origin.Tree, Path: p.origin.Path, ReadAs: p.readFrom}
	d.Skill = importSkill{Name: p.cand.Name, Description: p.cand.Description}
	for _, f := range p.files {
		d.Files = append(d.Files, importFile{Path: f.Path, Size: len(f.Data), Executable: f.Exec})
	}
	for _, s := range p.skipped {
		d.Skipped = append(d.Skipped, skipDoc{Path: s.Path, Reason: s.Reason})
	}
	for _, c := range p.origin.Changes {
		d.Changes = append(d.Changes, changeDoc{File: c.File, Reason: c.Reason, Detail: c.Detail})
	}
	d.License.Value, d.License.Note = p.origin.License, p.licenseMsg
	d.Review.Scripts, d.Review.Binaries = strs(p.review.Scripts), strs(p.review.Binaries)
	d.Review.OutsideLinks, d.Review.RunLines = strs(p.review.OutLinks), strs(p.review.RunLines)
	return d
}

func kib(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f KiB", float64(n)/1024)
}

// printImportReview shows what is about to be brought in. Everything here is
// decided before the first call to GitHub.
func printImportReview(w io.Writer, p importPrep, d importDoc) {
	total := 0
	for _, f := range p.files {
		total += len(f.Data)
	}
	from := p.origin.Source
	if p.origin.Commit != nil {
		from += " @ " + shortCommit(*p.origin.Commit)
	}
	fmt.Fprintf(w, "\nImporting %s\n", p.cand.Name)
	fmt.Fprintf(w, "  from:     %s (%s)\n", from, p.origin.Path)
	fmt.Fprintf(w, "  read as:  %s\n", p.readFrom)
	fmt.Fprintf(w, "  files:    %d, %s\n", len(p.files), kib(total))
	for _, f := range p.files {
		mark := ""
		if f.Exec {
			mark = "  (executable)"
		}
		fmt.Fprintf(w, "    %s  %s%s\n", path.Clean(f.Path), kib(len(f.Data)), mark)
	}
	fmt.Fprintf(w, "  licence:  %s\n", p.licenseMsg)
	for _, c := range p.origin.Changes {
		fmt.Fprintf(w, "  changed:  %s: %s\n", c.File, c.Detail)
	}
	for _, s := range p.skipped {
		fmt.Fprintf(w, "  skipped:  %s (%s)\n", s.Path, s.Reason)
	}
	r := p.review
	flag := func(label string, items []string) {
		if len(items) > 0 {
			fmt.Fprintf(w, "  %s %s\n", ui.Yellow(w, label), strings.Join(items, ", "))
		}
	}
	if len(r.Scripts)+len(r.Binaries)+len(r.OutLinks)+len(r.RunLines) > 0 {
		fmt.Fprintln(w, "  Read these before you continue. Skills run with your agent's permissions:")
	}
	flag("scripts and executables:", r.Scripts)
	flag("binary files:", r.Binaries)
	flag("links outside the skill folder:", r.OutLinks)
	flag("lines that run a download in a shell:", r.RunLines)
	for _, n := range p.notes {
		ui.PrintWarning(w, n)
	}
	fmt.Fprintln(w, "  Read SKILL.md before you import it; this review points at things to read, it does not scan for harm.")
}

// registrySkillNames lists the skill folders in the registry. A registry with no skills folder has none.
func registrySkillNames(r config.Registry) ([]string, error) {
	out, err := ghGet("repos/"+r.FullName()+"/contents/skills", ".[].name")
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read the skills in %s: %w", r.FullName(), err)
	}
	return strings.Fields(out), nil
}

// checkCollision refuses to overwrite a skill the registry already has.
func checkCollision(w io.Writer, r config.Registry, name string, p importPrep, a importArgs) error {
	names, err := registrySkillNames(r)
	if err != nil {
		return err
	}
	for _, existing := range names {
		if !strings.EqualFold(existing, name) {
			continue
		}
		if existing != name {
			return fmt.Errorf("%s already has a skill called %q, which differs from %q only in letter case; both would collide on macOS and Windows\nimport it under another name: hikma skill import %s --as <name>", r.FullName(), existing, name, a.arg)
		}
		if a.force {
			fmt.Fprintf(w, "\n%s already has %s; --force replaces it.\n", r.FullName(), name)
			return nil
		}
		have, _ := ghGet(fmt.Sprintf("repos/%s/contents/skills/%s", r.FullName(), existing), `[.[] | if .type == "dir" then .name + "/" else .name end] | join(", ")`)
		var mine []string
		for _, f := range p.pushFiles(name) {
			rel := strings.TrimPrefix(f.path, "skills/"+name+"/")
			mine = append(mine, strings.SplitN(rel, "/", 2)[0])
		}
		mine = uniq(mine)
		return fmt.Errorf("%s already has a skill called %q, so nothing was imported\n  registry has: %s\n  import has:   %s\nimport it beside the existing one: hikma skill import %s --as <name>\nor replace the existing one: add --force", r.FullName(), name, have, strings.Join(mine, ", "), a.arg)
	}
	return nil
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func confirmImport(cmd *cobra.Command, w io.Writer, p importPrep, r config.Registry, regOrigin, how string, yes bool) error {
	from := p.origin.Source
	if p.origin.Commit != nil {
		from += " @ " + shortCommit(*p.origin.Commit)
	}
	fmt.Fprintf(w, "\nYou are importing a skill into registry: %s (%s)\n", r.FullName(), regOrigin)
	fmt.Fprintf(w, "  skill: %s, %d files\n", p.cand.Name, len(p.files))
	fmt.Fprintf(w, "  from:  %s\n", from)
	fmt.Fprintf(w, "  how:   %s\n", how)
	for _, n := range p.notes {
		if strings.HasPrefix(n, "no licence") {
			fmt.Fprintf(w, "  note:  %s\n", n)
		}
	}
	if yes {
		return nil
	}
	if !isInteractive() {
		return fmt.Errorf("confirmation required: re-run with --yes to import into %s (add --dry-run to review without writing)", r.FullName())
	}
	ok, err := prompter.Confirm("Import into "+r.FullName()+"?", false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("import cancelled")
	}
	return nil
}

func importPRBody(p importPrep, name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Imports the `%s` skill from %s.\n\n", name, p.origin.Source)
	fmt.Fprintf(&b, "- Path: `%s`\n", p.origin.Path)
	if p.origin.Commit != nil {
		fmt.Fprintf(&b, "- Commit: `%s`\n", *p.origin.Commit)
	}
	if p.origin.Tree != nil {
		fmt.Fprintf(&b, "- Tree: `%s`\n", *p.origin.Tree)
	}
	fmt.Fprintf(&b, "- Licence: %s\n", p.licenseMsg)
	fmt.Fprintf(&b, "- Files: %d\n", len(p.files))
	for _, c := range p.origin.Changes {
		fmt.Fprintf(&b, "- Changed from upstream: %s (%s)\n", c.File, c.Detail)
	}
	for _, grp := range []struct {
		label string
		items []string
	}{{"Scripts and executables", p.review.Scripts}, {"Binary files", p.review.Binaries}, {"Links outside the skill folder", p.review.OutLinks}, {"Lines that run a download in a shell", p.review.RunLines}} {
		if len(grp.items) > 0 {
			fmt.Fprintf(&b, "- **%s to read:** `%s`\n", grp.label, strings.Join(grp.items, "`, `"))
		}
	}
	b.WriteString("\nSkills run with the agent's permissions: read `SKILL.md` and anything listed above before merging. Where it came from is recorded in `.hikma-origin.json`, which is advisory and does not show the skill is safe.\n\nCreated via `hikma skill import`.")
	return b.String()
}
