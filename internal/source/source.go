// Package source resolves skill sources (GitHub repos, git URLs, local paths)
// into local checkouts and discovers the skills inside them.
package source

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Kind says how a source is read.
type Kind string

const (
	KindGitHub Kind = "github"
	KindGit    Kind = "git"
	KindLocal  Kind = "local"
)

// Source is a place skills can be read from.
type Source struct {
	Raw      string // value as given, or the absolute path for local sources
	Kind     Kind
	Display  string // short human label, e.g. owner/repo
	CloneURL string // remote sources only
	Path     string // local sources only
	Owner    string // GitHub sources only
	Repo     string // GitHub sources only
}

var shorthand = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// LooksLikeSource reports whether arg is a repo, URL, or path rather than a bare skill name.
func LooksLikeSource(arg string) bool {
	return strings.ContainsAny(arg, `/\`) || strings.Contains(arg, "://") ||
		strings.HasPrefix(arg, "git@") || arg == "." || arg == ".." || strings.HasPrefix(arg, "~")
}

// Parse accepts GitHub shorthand (owner/repo), GitHub or other git URLs, and local paths.
func Parse(value string) (Source, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return Source{}, fmt.Errorf("source cannot be empty")
	}

	if isLocalPath(raw) {
		p, err := expandPath(raw)
		if err != nil {
			return Source{}, err
		}
		return Source{Raw: p, Kind: KindLocal, Display: p, Path: p}, nil
	}

	if owner, repo, ok := githubParts(raw); ok {
		return Source{
			Raw: raw, Kind: KindGitHub, Display: owner + "/" + repo,
			CloneURL: "https://github.com/" + owner + "/" + repo + ".git",
			Owner:    owner, Repo: repo,
		}, nil
	}

	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "git@") {
		return Source{Raw: raw, Kind: KindGit, Display: raw, CloneURL: raw}, nil
	}

	return Source{}, fmt.Errorf("source must be GitHub owner/repo, a git URL, or a local path: got %q", value)
}

func isLocalPath(s string) bool {
	return strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~") ||
		s == "." || s == ".." || filepath.IsAbs(s) || strings.HasPrefix(s, `.\`) || strings.HasPrefix(s, `..\`)
}

func expandPath(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", p, err)
	}
	return abs, nil
}

// githubParts extracts owner and repo from shorthand, https, or ssh GitHub forms.
func githubParts(raw string) (owner, repo string, ok bool) {
	s := strings.TrimSuffix(raw, ".git")
	switch {
	case strings.HasPrefix(s, "https://github.com/"):
		s = strings.TrimPrefix(s, "https://github.com/")
	case strings.HasPrefix(s, "http://github.com/"):
		s = strings.TrimPrefix(s, "http://github.com/")
	case strings.HasPrefix(s, "git@github.com:"):
		s = strings.TrimPrefix(s, "git@github.com:")
	case strings.HasPrefix(s, "ssh://git@github.com/"):
		s = strings.TrimPrefix(s, "ssh://git@github.com/")
	case strings.Contains(s, "://") || strings.HasPrefix(s, "git@"):
		return "", "", false
	}
	s = strings.Trim(s, "/")
	if !shorthand.MatchString(s) {
		return "", "", false
	}
	parts := strings.SplitN(s, "/", 2)
	return parts[0], parts[1], true
}
