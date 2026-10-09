package source

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provenance is the source tracking that `gh skill install` writes into a
// skill's SKILL.md frontmatter, under metadata.
type Provenance struct {
	Repo    string // github-repo, a repository URL
	Path    string // github-path, the skill's folder inside the repository
	Ref     string // github-ref, such as refs/heads/main or refs/tags/v1.0.0
	TreeSHA string // github-tree-sha
	Pinned  string // github-pinned, a tag or commit when installed with --pin
}

// ReadProvenance returns the gh skill tracking metadata of the skill in dir.
// ok is false when SKILL.md has none.
func ReadProvenance(dir string) (p Provenance, ok bool) {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return p, false
	}
	var fm struct {
		Metadata map[string]any `yaml:"metadata"`
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return p, false
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 || yaml.Unmarshal([]byte(content[4:4+end]), &fm) != nil {
		return p, false
	}
	get := func(k string) string {
		s, _ := fm.Metadata[k].(string)
		return strings.TrimSpace(s)
	}
	p = Provenance{Repo: get("github-repo"), Path: get("github-path"), Ref: get("github-ref"), TreeSHA: get("github-tree-sha"), Pinned: get("github-pinned")}
	return p, p.Repo != ""
}

// SourceRef returns the source and ref to update this skill from: the pinned
// version when there is one, otherwise the recorded branch or tag.
func (p Provenance) SourceRef() (src, ref string) {
	ref = p.Pinned
	if ref == "" {
		ref = strings.TrimPrefix(strings.TrimPrefix(p.Ref, "refs/heads/"), "refs/tags/")
	}
	return p.Repo, ref
}
