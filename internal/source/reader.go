package source

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// EntryKind says what a tree entry is.
type EntryKind int

const (
	KindFile EntryKind = iota
	KindExec
	KindSymlink
	KindSubmodule
)

// Entry is one file in a source. Path is slash-separated and relative to the
// root of the source.
type Entry struct {
	Path string
	Kind EntryKind
	Sha  string // git object id; empty for a working tree
}

// Reader reads a source without changing a single byte of it.
type Reader interface {
	// Entries lists every entry under dir, recursively. "" means the whole source.
	Entries(dir string) ([]Entry, error)
	// Read returns the bytes of an entry exactly as stored.
	Read(e Entry) ([]byte, error)
	// Identity names the commit and the git tree id of dir. Both are empty, with a
	// reason, when the source is not a clean git tree.
	Identity(dir string) (commit, tree, why string)
	// Describe says where the bytes come from, for the review.
	Describe() string
}

func gitOut(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %s", args[0], firstLine(stderr.String(), err))
	}
	return out.Bytes(), nil
}

// gitReader reads objects from a repository at one commit. Reading objects and
// not a checked-out tree keeps line-ending conversion, .gitattributes filters,
// and Git LFS smudging from changing what is imported.
type gitReader struct {
	dir, commit string
}

func (g *gitReader) Describe() string { return "git objects at " + g.commit[:min(7, len(g.commit))] }

func (g *gitReader) Entries(dir string) ([]Entry, error) {
	args := []string{"ls-tree", "-r", "-z", g.commit}
	if dir != "" && dir != "." {
		args = append(args, "--", dir)
	}
	out, err := gitOut(g.dir, args...)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, rec := range bytes.Split(out, []byte{0}) {
		meta, name, ok := bytes.Cut(rec, []byte{'\t'})
		if !ok {
			continue
		}
		f := strings.Fields(string(meta))
		if len(f) != 3 {
			continue
		}
		e := Entry{Path: string(name), Sha: f[2], Kind: KindFile}
		switch f[0] {
		case "100755":
			e.Kind = KindExec
		case "120000":
			e.Kind = KindSymlink
		case "160000":
			e.Kind = KindSubmodule
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (g *gitReader) Read(e Entry) ([]byte, error) { return gitOut(g.dir, "cat-file", "blob", e.Sha) }

func (g *gitReader) Identity(dir string) (string, string, string) {
	spec := g.commit + "^{tree}"
	if dir != "" && dir != "." {
		spec = g.commit + ":" + dir
	}
	tree, err := gitOut(g.dir, "rev-parse", spec)
	if err != nil {
		return g.commit, "", ""
	}
	return g.commit, strings.TrimSpace(string(tree)), ""
}

// treeReader reads a local folder as it is on disk.
type treeReader struct{ root string }

func (t *treeReader) Describe() string { return "the working tree (not git objects)" }

func (t *treeReader) Entries(dir string) ([]Entry, error) {
	base := t.root
	if dir != "" && dir != "." {
		base = filepath.Join(t.root, filepath.FromSlash(dir))
	}
	var entries []Entry
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(t.root, p)
		if err != nil {
			return err
		}
		e := Entry{Path: filepath.ToSlash(rel), Kind: KindFile}
		if d.Type()&os.ModeSymlink != 0 {
			e.Kind = KindSymlink
		} else if info, err := d.Info(); err == nil && info.Mode()&0o111 != 0 {
			e.Kind = KindExec
		}
		entries = append(entries, e)
		return nil
	})
	return entries, err
}

func (t *treeReader) Read(e Entry) ([]byte, error) {
	return os.ReadFile(filepath.Join(t.root, filepath.FromSlash(e.Path)))
}

func (t *treeReader) Identity(dir string) (string, string, string) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", "git is not installed"
	}
	prefix, err := gitOut(t.root, "rev-parse", "--show-prefix")
	if err != nil {
		return "", "", "not a git repository"
	}
	rel := path.Join(strings.TrimSpace(string(prefix)), dir)
	scope := dir
	if scope == "" {
		scope = "."
	}
	status, err := gitOut(t.root, "status", "--porcelain", "--", scope)
	if err != nil || len(bytes.TrimSpace(status)) > 0 {
		return "", "", "uncommitted changes"
	}
	commit, err := gitOut(t.root, "rev-parse", "HEAD")
	if err != nil {
		return "", "", "no commits yet"
	}
	spec := "HEAD^{tree}"
	if rel != "" && rel != "." {
		spec = "HEAD:" + rel
	}
	tree, err := gitOut(t.root, "rev-parse", spec)
	if err != nil {
		return "", "", "not a git repository"
	}
	return strings.TrimSpace(string(commit)), strings.TrimSpace(string(tree)), ""
}

// OpenReader makes a source readable. A remote source is fetched shallow and
// without file contents (they are read one by one, only for the skill that is
// chosen), and nothing is checked out. The returned func removes the download.
func OpenReader(s Source, ref string) (Reader, func(), error) {
	noop := func() {}
	if s.Kind == KindLocal {
		if ref != "" {
			return nil, noop, fmt.Errorf("--ref is not supported for local paths")
		}
		if info, err := os.Stat(s.Path); err != nil || !info.IsDir() {
			return nil, noop, fmt.Errorf("local source %s is not a directory", s.Path)
		}
		return &treeReader{root: s.Path}, noop, nil
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil, noop, fmt.Errorf("git not found on PATH - install git to read remote sources")
	}
	tmp, err := os.MkdirTemp("", "hikma-src-")
	if err != nil {
		return nil, noop, fmt.Errorf("create temp directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	dir := filepath.Join(tmp, "repo")
	if err := fetchObjects(s, ref, dir); err != nil {
		cleanup()
		return nil, noop, err
	}
	commit, err := gitOut(dir, "rev-parse", "HEAD")
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	return &gitReader{dir: dir, commit: strings.TrimSpace(string(commit))}, cleanup, nil
}

// fetchObjects downloads one commit of a remote, with no working tree.
func fetchObjects(s Source, ref, dir string) error {
	args := []string{"clone", "--quiet", "--depth", "1", "--filter=blob:none", "--no-checkout"}
	branch := append([]string{}, args...)
	if ref != "" {
		branch = append(branch, "--branch", ref)
	}
	err := runGit(append(branch, s.CloneURL, dir)...)
	if err == nil {
		return nil
	}
	// A commit id cannot be given to --branch; fetch it directly.
	if ref != "" {
		_ = os.RemoveAll(dir)
		if runGit("init", "--quiet", dir) == nil && runGit("-C", dir, "remote", "add", "origin", s.CloneURL) == nil &&
			runGit("-C", dir, "fetch", "--quiet", "--depth", "1", "--filter=blob:none", "origin", ref) == nil &&
			runGit("-C", dir, "update-ref", "HEAD", "FETCH_HEAD") == nil {
			return nil
		}
		_ = os.RemoveAll(dir)
	}
	// Private GitHub repositories often work through gh credentials when plain git has none.
	if s.Kind == KindGitHub {
		if _, lookErr := exec.LookPath("gh"); lookErr == nil {
			ghArgs := []string{"repo", "clone", s.Owner + "/" + s.Repo, dir, "--", "--quiet", "--depth", "1", "--filter=blob:none", "--no-checkout"}
			if ref != "" {
				ghArgs = append(ghArgs, "--branch", ref)
			}
			if run("gh", ghArgs...) == nil {
				return nil
			}
		}
	}
	if ref != "" {
		return fmt.Errorf("cannot read %s at %q: %v", s.Display, ref, err)
	}
	return fmt.Errorf("cannot read %s: %w", s.Display, err)
}
