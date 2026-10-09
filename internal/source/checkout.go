package source

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Checkout is a readable local directory holding a source's files.
type Checkout struct {
	Dir    string
	Commit string // empty when the directory is not a git repository
	Ref    string
}

// Resolve makes the source readable locally. Remote sources are shallow-cloned
// into a temporary directory; the returned cleanup func removes it.
func Resolve(s Source, ref string) (Checkout, func(), error) {
	noop := func() {}
	if s.Kind == KindLocal {
		if ref != "" {
			return Checkout{}, noop, fmt.Errorf("--ref is not supported for local paths")
		}
		info, err := os.Stat(s.Path)
		if err != nil || !info.IsDir() {
			return Checkout{}, noop, fmt.Errorf("local source %s is not a directory", s.Path)
		}
		return Checkout{Dir: s.Path, Commit: headCommit(s.Path)}, noop, nil
	}

	if _, err := exec.LookPath("git"); err != nil {
		return Checkout{}, noop, fmt.Errorf("git not found on PATH - install git to read remote sources")
	}
	tmp, err := os.MkdirTemp("", "hikma-src-")
	if err != nil {
		return Checkout{}, noop, fmt.Errorf("create temp directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	dir := tmp + string(os.PathSeparator) + "repo"

	if err := clone(s, ref, dir); err != nil {
		cleanup()
		return Checkout{}, noop, err
	}
	return Checkout{Dir: dir, Commit: headCommit(dir), Ref: ref}, cleanup, nil
}

func clone(s Source, ref, dir string) error {
	args := []string{"clone", "--quiet", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	err := runGit(append(args, s.CloneURL, dir)...)
	if err == nil {
		return nil
	}

	// A ref that is a commit SHA cannot be used with --branch; clone fully and check it out.
	if ref != "" {
		_ = os.RemoveAll(dir)
		if full := runGit("clone", "--quiet", s.CloneURL, dir); full == nil {
			if co := runGit("-C", dir, "checkout", "--quiet", ref); co == nil {
				return nil
			}
			return fmt.Errorf("ref %q not found in %s", ref, s.Display)
		}
	}

	// Private GitHub repos often work through gh credentials even when plain git cannot authenticate.
	if s.Kind == KindGitHub {
		if _, lookErr := exec.LookPath("gh"); lookErr == nil {
			_ = os.RemoveAll(dir)
			ghArgs := []string{"repo", "clone", s.Owner + "/" + s.Repo, dir, "--", "--quiet", "--depth", "1"}
			if ref != "" {
				ghArgs = append(ghArgs, "--branch", ref)
			}
			if ghErr := run("gh", ghArgs...); ghErr == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("cannot read %s: %w", s.Display, err)
}

func headCommit(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runGit(args ...string) error { return run("git", args...) }

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
