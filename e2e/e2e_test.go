// Package e2e drives the real hikma binary through a pseudo-terminal, the way a
// person at a prompt would. It covers the parts a unit test cannot: that the
// menu draws, answers keys, and exits cleanly, and that color follows the
// terminal. The terminal itself is platform code (a pty on Unix, ConPTY on
// Windows) behind the small term type in term_*_test.go.
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	binary    string
	buildErr  error
)

// hikmaBinary builds the CLI once per test run.
func hikmaBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "hikma-e2e-bin")
		if err != nil {
			buildErr = err
			return
		}
		name := "hikma"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		binary = filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", binary, "github.com/hasankhatib/hikma-ai/cmd/hikma")
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = &buildError{err: err, out: string(out)}
		}
	})
	if buildErr != nil {
		t.Fatalf("build hikma: %v", buildErr)
	}
	return binary
}

type buildError struct {
	err error
	out string
}

func (e *buildError) Error() string { return e.err.Error() + "\n" + e.out }

// sandbox is an isolated home and an empty git project, so a test never reads
// or writes the real user's config.
type sandbox struct {
	home, project string
	env           []string
}

func newSandbox(t *testing.T) sandbox {
	t.Helper()
	home := t.TempDir()
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"APPDATA=" + filepath.Join(home, "AppData", "Roaming"),
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, ".gitconfig"),
		"PATH=" + os.Getenv("PATH"),
	}
	if runtime.GOOS == "windows" {
		env = append(env, "SystemRoot="+os.Getenv("SystemRoot"), "TEMP="+os.Getenv("TEMP"), "TMP="+os.Getenv("TMP"))
	}
	return sandbox{home: home, project: project, env: env}
}

func (s sandbox) with(kv ...string) []string { return append(append([]string{}, s.env...), kv...) }

func (s sandbox) writeSkill(t *testing.T, name string) {
	t.Helper()
	dir := filepath.Join(s.project, ".claude", "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\ndescription: a test skill\n---\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
