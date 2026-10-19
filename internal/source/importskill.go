package source

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Limits on what one import may bring in. They are documented and have no
// override in the first release.
const (
	MaxImportFiles = 100
	MaxImportBytes = 5 << 20
)

// OriginFile is the name of the file that records where an imported skill came from.
const OriginFile = ".hikma-origin.json"

// Candidate is a skill found in a source.
type Candidate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Dir         string `json:"path"` // folder relative to the source, "." for the root
}

// FindCandidates lists the skills in a source, in the layouts gh skill uses:
// SKILL.md at the root, <name>/SKILL.md, skills/<name>/SKILL.md,
// skills/<scope>/<name>/SKILL.md, and plugins/<scope>/skills/<name>/SKILL.md.
// Hidden folders such as .claude/skills are skipped unless scope points into one.
// scope narrows the search to a folder; a scope that is itself a skill folder
// returns just that skill. fallback names a skill at the root.
func FindCandidates(r Reader, scope, fallback string) ([]Candidate, error) {
	entries, err := r.Entries("")
	if err != nil {
		return nil, err
	}
	allowHidden := hasHidden(scope)
	var out []Candidate
	for _, e := range entries {
		if e.Kind != KindFile && e.Kind != KindExec {
			continue
		}
		if e.Path != "SKILL.md" && !strings.HasSuffix(e.Path, "/SKILL.md") {
			continue
		}
		dir := path.Dir(e.Path)
		if scope != "" && dir != scope && !strings.HasPrefix(dir, scope+"/") {
			continue
		}
		if scope != dir && !layoutOK(dir) {
			continue
		}
		if !allowHidden && hasHidden(dir) {
			continue
		}
		c := Candidate{Name: path.Base(dir), Dir: dir}
		if dir == "." {
			c.Name = fallback
		}
		if data, err := r.Read(e); err == nil {
			if fm, ok := parseFrontmatter(string(data)); ok {
				c.Description = strings.TrimSpace(fm.Description)
				if dir == "." && fm.Name != "" {
					c.Name = fm.Name
				}
			}
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out, nil
}

// layoutOK reports whether a folder holding a SKILL.md sits where gh skill looks.
func layoutOK(dir string) bool {
	parts := strings.Split(dir, "/")
	switch {
	case dir == ".":
		return true
	case len(parts) == 1:
		return true
	case parts[0] == "skills" && (len(parts) == 2 || len(parts) == 3):
		return true
	case parts[0] == "plugins" && len(parts) == 4 && parts[2] == "skills":
		return true
	}
	return false
}

func hasHidden(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if strings.HasPrefix(part, ".") && part != "." && part != "" {
			return true
		}
	}
	return false
}

// SelectCandidate picks a skill by name or by path. Two skills with the same
// name are an error that lists their paths: the caller must pick by path.
func SelectCandidate(cands []Candidate, name string) (Candidate, error) {
	var hits []Candidate
	for _, c := range cands {
		if c.Dir == name || c.Name == name {
			hits = append(hits, c)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return Candidate{}, fmt.Errorf("skill %q not found", name)
	}
	var paths []string
	for _, c := range hits {
		paths = append(paths, c.Dir)
	}
	return Candidate{}, fmt.Errorf("%d skills are named %q (%s): pass the path of the one you want", len(hits), name, strings.Join(paths, ", "))
}

// File is one file of a skill, relative to the skill folder.
type File struct {
	Path string
	Data []byte
	Exec bool
}

// Skipped is an entry that is not imported, and why.
type Skipped struct{ Path, Reason string }

// Loaded is the content of one skill, read from a source.
type Loaded struct {
	Files   []File
	Skipped []Skipped
}

var lfsPointer = []byte("version https://git-lfs.github.com/spec/v1")

// LoadSkill reads a skill folder, with the caps and containment an import needs.
// Symlinks and submodules are skipped and reported; names that could leave the
// folder are refused.
func LoadSkill(r Reader, dir string) (Loaded, error) {
	entries, err := r.Entries(dir)
	if err != nil {
		return Loaded{}, err
	}
	prefix := ""
	if dir != "" && dir != "." {
		prefix = dir + "/"
	}
	var l Loaded
	var take []Entry
	seen := map[string]string{}
	for _, e := range entries {
		rel := strings.TrimPrefix(e.Path, prefix)
		switch {
		case e.Kind == KindSymlink:
			l.Skipped = append(l.Skipped, Skipped{rel, "symlink"})
			continue
		case e.Kind == KindSubmodule:
			l.Skipped = append(l.Skipped, Skipped{rel, "submodule"})
			continue
		case rel == OriginFile:
			l.Skipped = append(l.Skipped, Skipped{rel, "name is reserved for hikma"})
			continue
		}
		if err := safeRel(rel); err != nil {
			return Loaded{}, fmt.Errorf("%s: %w", rel, err)
		}
		if other, dup := seen[strings.ToLower(rel)]; dup {
			return Loaded{}, fmt.Errorf("%s and %s differ only in letter case, so they would collide on macOS and Windows", other, rel)
		}
		seen[strings.ToLower(rel)] = rel
		take = append(take, e)
	}
	if len(take) > MaxImportFiles {
		return Loaded{}, fmt.Errorf("the skill folder has %d files; an import takes at most %d (there is no override in this release)", len(take), MaxImportFiles)
	}
	var total int
	for _, e := range take {
		data, err := r.Read(e)
		if err != nil {
			return Loaded{}, fmt.Errorf("read %s: %w", e.Path, err)
		}
		rel := strings.TrimPrefix(e.Path, prefix)
		if len(data) < 1024 && bytes.HasPrefix(data, lfsPointer) {
			return Loaded{}, fmt.Errorf("%s is a Git LFS pointer, not the file itself, so it would be unusable once imported", rel)
		}
		total += len(data)
		if total > MaxImportBytes {
			return Loaded{}, fmt.Errorf("the skill is larger than %d MiB; an import takes at most that (there is no override in this release)", MaxImportBytes>>20)
		}
		l.Files = append(l.Files, File{Path: rel, Data: data, Exec: e.Kind == KindExec})
	}
	sort.Slice(l.Files, func(i, j int) bool { return l.Files[i].Path < l.Files[j].Path })
	return l, nil
}

// safeRel refuses a path that is empty, absolute, or could climb out of the skill folder.
func safeRel(rel string) error {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, `\`) || strings.Contains(rel, "\x00") {
		return fmt.Errorf("unsafe file name")
	}
	if len(rel) >= 2 && rel[1] == ':' {
		return fmt.Errorf("unsafe file name")
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return fmt.Errorf("unsafe file name")
		}
	}
	return nil
}

// Hash is the sha256 of data, written the way the origin file records it.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// splitFrontmatter returns the lines of SKILL.md and the index of the closing
// "---" line, keeping line endings so edits change nothing else.
func splitFrontmatter(data []byte) (lines [][]byte, end int, ok bool) {
	lines = bytes.SplitAfter(data, []byte("\n"))
	if len(lines) == 0 || string(trimEOL(lines[0])) != "---" {
		return lines, 0, false
	}
	for i := 1; i < len(lines); i++ {
		if string(trimEOL(lines[i])) == "---" {
			return lines, i, true
		}
	}
	return lines, 0, false
}

func trimEOL(b []byte) []byte { return bytes.TrimRight(b, "\r\n ") }

var (
	metadataKey = regexp.MustCompile(`^metadata:\s*(#.*)?$`)
	ghKeyLine   = regexp.MustCompile(`^[ \t]+(github-[A-Za-z0-9_-]+)\s*:`)
	nameLine    = regexp.MustCompile(`^name:\s*.*$`)
)

// StripGitHubMetadata removes metadata.github-* from SKILL.md, the keys `gh skill
// install` writes to remember where a skill came from. If they stayed, `gh skill
// update` on a machine that installed from your registry could fetch the upstream
// skill and skip your review. Every other byte is kept. removed names the keys.
func StripGitHubMetadata(data []byte) (out []byte, removed []string, err error) {
	lines, end, ok := splitFrontmatter(data)
	if !ok {
		return data, nil, nil
	}
	start := -1
	for i := 1; i < end; i++ {
		if metadataKey.Match(trimEOL(lines[i])) {
			start = i
			break
		}
	}
	if start < 0 {
		return data, nil, ghKeysIn(data)
	}
	stop := start + 1
	for stop < end {
		l := lines[stop]
		if len(trimEOL(l)) > 0 && l[0] != ' ' && l[0] != '\t' {
			break
		}
		stop++
	}
	var keep [][]byte
	keep = append(keep, lines[:start+1]...)
	remaining := 0
	for _, l := range lines[start+1 : stop] {
		if m := ghKeyLine.FindSubmatch(trimEOL(l)); m != nil {
			removed = append(removed, "metadata."+string(m[1]))
			continue
		}
		if t := bytes.TrimSpace(l); len(t) > 0 && t[0] != '#' {
			remaining++
		}
		keep = append(keep, l)
	}
	if len(removed) == 0 {
		return data, nil, ghKeysIn(data)
	}
	if remaining == 0 {
		keep = keep[:start] // no other metadata left: drop the key too
		for _, l := range lines[start+1 : stop] {
			if t := bytes.TrimSpace(l); len(t) > 0 && t[0] == '#' {
				keep = append(keep, l)
			}
		}
	}
	keep = append(keep, lines[stop:]...)
	out = bytes.Join(keep, nil)
	if err := ghKeysIn(out); err != nil {
		return nil, nil, err
	}
	return out, removed, nil
}

// ghKeysIn fails when SKILL.md still has metadata.github-* in a form that could not be removed line by line.
func ghKeysIn(data []byte) error {
	lines, end, ok := splitFrontmatter(data)
	if !ok {
		return nil
	}
	var doc struct {
		Metadata map[string]any `yaml:"metadata"`
	}
	if yaml.Unmarshal(bytes.Join(lines[1:end], nil), &doc) != nil {
		return nil
	}
	for k := range doc.Metadata {
		if strings.HasPrefix(k, "github-") {
			return fmt.Errorf("SKILL.md has metadata.%s in a form hikma cannot remove safely: remove the metadata.github-* keys and run the import again", k)
		}
	}
	return nil
}

// SetSkillName rewrites the name line of SKILL.md and nothing else. The spec
// requires the name to match the folder, so --as has to change it.
func SetSkillName(data []byte, name string) ([]byte, error) {
	lines, end, ok := splitFrontmatter(data)
	if !ok {
		return nil, fmt.Errorf("SKILL.md must start with YAML frontmatter between --- lines")
	}
	for i := 1; i < end; i++ {
		if !nameLine.Match(trimEOL(lines[i])) {
			continue
		}
		eol := lines[i][len(trimEOL(lines[i])):]
		lines[i] = append([]byte("name: "+name), eol...)
		out := bytes.Join(lines, nil)
		if fm, ok := parseFrontmatter(string(out)); !ok || fm.Name != name {
			return nil, fmt.Errorf("could not set the skill name to %q in SKILL.md", name)
		}
		return out, nil
	}
	return nil, fmt.Errorf("SKILL.md has no top-level name line to rewrite")
}

// FrontmatterString returns a top-level string field of SKILL.md, or "".
func FrontmatterString(data []byte, key string) string {
	lines, end, ok := splitFrontmatter(data)
	if !ok {
		return ""
	}
	var doc map[string]any
	if yaml.Unmarshal(bytes.Join(lines[1:end], nil), &doc) != nil {
		return ""
	}
	s, _ := doc[key].(string)
	return strings.TrimSpace(s)
}

// Change records one difference between an imported file and its upstream.
type Change struct {
	File     string `json:"file"`
	Reason   string `json:"reason"`
	Detail   string `json:"detail,omitempty"`
	Upstream string `json:"upstream"`
}

// Origin is the content of .hikma-origin.json: where an imported skill came from.
// It is advisory. It records a source; it does not prove the copy is safe.
type Origin struct {
	Version int               `json:"version"`
	Source  string            `json:"source"`
	Ref     *string           `json:"ref"`
	Commit  *string           `json:"commit"`
	Tree    *string           `json:"tree"`
	Path    string            `json:"path"`
	License *string           `json:"license"`
	Files   map[string]string `json:"files"`
	Changes []Change          `json:"changes"`
}

// OriginVersion is the schema version of the origin file.
const OriginVersion = 1

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NewOrigin fills in the fields that do not depend on the files.
func NewOrigin(src, ref, commit, tree, dir, license string) Origin {
	if dir == "" {
		dir = "."
	}
	return Origin{
		Version: OriginVersion, Source: src, Ref: ptr(ref), Commit: ptr(commit), Tree: ptr(tree),
		Path: dir, License: ptr(license), Files: map[string]string{}, Changes: []Change{},
	}
}

// OriginSource is the credential-free text recorded as the source: owner/repo
// for GitHub, and "local" for a folder, never an absolute path.
func OriginSource(s Source) string {
	switch s.Kind {
	case KindLocal:
		return "local"
	case KindGitHub:
		return s.Owner + "/" + s.Repo
	}
	return StripCredentials(s.CloneURL)
}

// Marshal renders the origin file.
func (o Origin) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ParseOrigin reads an origin file. Unknown fields are ignored, so a file
// written by a newer hikma still reads.
func ParseOrigin(data []byte) (Origin, error) {
	var o Origin
	if err := json.Unmarshal(data, &o); err != nil {
		return Origin{}, fmt.Errorf("read %s: %w", OriginFile, err)
	}
	return o, nil
}

// Review is what a person should look at before an import is confirmed.
type Review struct {
	Scripts    []string // files under scripts/, executables, and files that start with a shebang
	Binaries   []string
	OutLinks   []string // links in SKILL.md to paths outside the skill folder
	RunLines   []string // lines of SKILL.md that pipe a download into a shell
	LicenseSrc string   // where the licence was found, "" when it was not
}

var (
	mdLink  = regexp.MustCompile(`\]\(\s*<?([^)\s>]+)`)
	runLine = regexp.MustCompile(`(?i)(curl|wget)\b[^|\n]*\|\s*(sudo\s+)?(ba|z|da)?sh\b|\b(iwr|irm|Invoke-WebRequest|Invoke-RestMethod)\b[^|\n]*\|\s*(iex|Invoke-Expression)\b`)
)

// ReviewFiles inspects loaded files for the things the review flags.
func ReviewFiles(files []File) Review {
	var r Review
	for _, f := range files {
		if strings.HasPrefix(f.Path, "scripts/") || f.Exec || bytes.HasPrefix(f.Data, []byte("#!")) {
			r.Scripts = append(r.Scripts, f.Path)
		}
		head := f.Data
		if len(head) > 8000 {
			head = head[:8000]
		}
		if bytes.IndexByte(head, 0) >= 0 {
			r.Binaries = append(r.Binaries, f.Path)
		}
		if f.Path != "SKILL.md" {
			continue
		}
		for _, m := range mdLink.FindAllSubmatch(f.Data, -1) {
			target := string(m[1])
			if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			target = strings.SplitN(strings.SplitN(target, "#", 2)[0], "?", 2)[0]
			if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "~") || strings.HasPrefix(path.Clean(target), "..") {
				r.OutLinks = append(r.OutLinks, target)
			}
		}
		for _, line := range strings.Split(string(f.Data), "\n") {
			if runLine.MatchString(line) {
				r.RunLines = append(r.RunLines, strings.TrimSpace(line))
			}
		}
	}
	return r
}
