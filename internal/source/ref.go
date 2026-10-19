package source

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
)

// SkillRef is a source plus the part of the reference that names a branch or a
// folder inside it: owner/repo/skills/pdf, or a GitHub tree or blob URL.
type SkillRef struct {
	Source Source
	Ref    string // branch, tag, or commit the reference names; "" for the default branch
	Path   string // folder (or SKILL.md) inside the source; "" when the whole source is meant
	// Tail is the "<ref>/<path>" text of a GitHub tree or blob URL. A branch name can
	// contain slashes, so it cannot be split without asking the remote: see ResolveTail.
	Tail string
	Blob bool // Tail came from a /blob/ URL, so it ends in a file
}

var segment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// ParseRef accepts everything Parse does, plus a folder inside a GitHub repo:
// owner/repo/path/to/skill and github.com URLs ending in /tree/<ref>/<path> or
// /blob/<ref>/<path>/SKILL.md. Credentials, queries, and fragments are never kept.
func ParseRef(value string) (SkillRef, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return SkillRef{}, fmt.Errorf("source cannot be empty")
	}
	if isLocalPath(raw) {
		s, err := Parse(raw)
		return SkillRef{Source: s}, err
	}

	switch {
	case strings.HasPrefix(raw, "git@github.com:"):
		return githubRef(strings.Split(strings.Trim(strings.TrimPrefix(raw, "git@github.com:"), "/"), "/"), raw)
	case strings.Contains(raw, "://"):
		u, err := url.Parse(raw)
		if err != nil {
			return SkillRef{}, fmt.Errorf("cannot read %q as a URL: %v", StripCredentials(raw), err)
		}
		host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		clean := StripCredentials(raw)
		if host == "github.com" {
			return githubRef(strings.Split(strings.Trim(u.Path, "/"), "/"), clean)
		}
		if strings.Contains(u.Path, "/tree/") || strings.Contains(u.Path, "/blob/") {
			return SkillRef{}, fmt.Errorf("only github.com URLs can name a folder or a branch (%s)\nclone it and import from the local path, or pass its clone URL", clean)
		}
		s, err := Parse(raw)
		s.Display = clean
		return SkillRef{Source: s}, err
	case strings.HasPrefix(raw, "git@"):
		s, err := Parse(raw)
		return SkillRef{Source: s}, err
	}

	segs := strings.Split(strings.Trim(raw, "/"), "/")
	if len(segs) > 2 && segment.MatchString(segs[0]) && segment.MatchString(segs[1]) {
		return githubRef(segs, raw)
	}
	s, err := Parse(raw)
	return SkillRef{Source: s}, err
}

// githubRef builds a reference from the path segments of a github.com address:
// owner, repo, then optionally tree/<tail>, blob/<tail>, or a folder path.
func githubRef(segs []string, shown string) (SkillRef, error) {
	if len(segs) < 2 || !segment.MatchString(segs[0]) || !segment.MatchString(strings.TrimSuffix(segs[1], ".git")) {
		return SkillRef{}, fmt.Errorf("%q is not a GitHub repository: expected owner/repo", shown)
	}
	owner, repo := segs[0], strings.TrimSuffix(segs[1], ".git")
	r := SkillRef{Source: Source{
		Raw: owner + "/" + repo, Kind: KindGitHub, Display: owner + "/" + repo,
		CloneURL: "https://github.com/" + owner + "/" + repo + ".git", Owner: owner, Repo: repo,
	}}
	rest := segs[2:]
	if len(rest) == 0 {
		return r, nil
	}
	if strings.Contains(shown, "://") || strings.HasPrefix(shown, "git@") {
		switch rest[0] {
		case "tree", "blob":
			if len(rest) < 2 {
				return r, nil
			}
			r.Tail, r.Blob = strings.Join(rest[1:], "/"), rest[0] == "blob"
			return r, nil
		default:
			return SkillRef{}, fmt.Errorf("%s does not point at a repository or at a folder in one\nuse owner/repo, or a /tree/<branch>/<folder> link", shown)
		}
	}
	r.Path = trimSkillFile(strings.Join(rest, "/"))
	return r, nil
}

// trimSkillFile turns a path to a SKILL.md into the folder that holds it.
func trimSkillFile(p string) string {
	p = path.Clean(strings.Trim(p, "/"))
	if p == "." {
		return ""
	}
	if path.Base(p) == "SKILL.md" {
		if dir := path.Dir(p); dir != "." {
			return dir
		}
		return ""
	}
	return p
}

// ResolveTail splits Tail into a ref and a folder. refs are the remote's branch
// and tag names; the longest one that starts the tail wins. When none does, the
// first segment is taken as the ref (a branch called "main", or a commit), and
// guessed is true so the caller can say so.
func (r SkillRef) ResolveTail(refs []string) (ref, folder string, guessed bool) {
	segs := strings.Split(r.Tail, "/")
	known := map[string]bool{}
	for _, name := range refs {
		known[name] = true
	}
	ref, rest := segs[0], segs[1:]
	found := len(segs) == 1
	for i := len(segs); i >= 1 && !found; i-- {
		if known[strings.Join(segs[:i], "/")] {
			ref, rest, found = strings.Join(segs[:i], "/"), segs[i:], true
		}
	}
	folder = strings.Join(rest, "/")
	if r.Blob || path.Base(folder) == "SKILL.md" {
		folder = trimSkillFile(folder)
	}
	return ref, folder, !found
}

// ListRefs returns the branch and tag names of a remote source.
func ListRefs(s Source) ([]string, error) {
	if s.Kind == KindLocal {
		return nil, nil
	}
	cmd := exec.Command("git", "ls-remote", "--heads", "--tags", s.CloneURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list refs of %s: %s", s.Display, firstLine(stderr.String(), err))
	}
	set := map[string]bool{}
	for _, line := range strings.Split(out.String(), "\n") {
		_, name, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok {
			continue
		}
		name = strings.TrimSuffix(name, "^{}")
		name = strings.TrimPrefix(strings.TrimPrefix(name, "refs/heads/"), "refs/tags/")
		set[name] = true
	}
	refs := make([]string, 0, len(set))
	for name := range set {
		refs = append(refs, name)
	}
	sort.Strings(refs)
	return refs, nil
}

func firstLine(s string, fallback error) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback.Error()
	}
	return strings.SplitN(s, "\n", 2)[0]
}

// StripCredentials removes any user, password, token, query, and fragment from
// a URL, so it is safe to print or record. Anything that is not a URL is returned as is.
func StripCredentials(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}
