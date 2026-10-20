package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func runCompletion(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := Root()
	cmd.SetArgs(append([]string{"completion"}, args...))
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	return out.String(), err
}

func fakeHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestCompletionInstallZshIsIdempotent(t *testing.T) {
	home := fakeHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("export A=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runCompletion(t, "zsh", "--install", "--yes"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(rc)
	if want := "export A=1\n" + completionMarker + "\n" + guardedSource("zsh") + "\n"; string(got) != want {
		t.Fatalf("zshrc = %q, want %q", got, want)
	}
	out, err := runCompletion(t, "zsh", "--install", "--yes")
	if err != nil || !strings.Contains(out, "already") {
		t.Fatalf("second run: %q, %v", out, err)
	}
	if again, _ := os.ReadFile(rc); !bytes.Equal(again, got) {
		t.Fatal("second install changed the file")
	}
}

func TestCompletionInstallFish(t *testing.T) {
	home := fakeHome(t)
	if _, err := runCompletion(t, "fish", "--install", "--yes"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(home, ".config", "fish", "completions", "hikma.fish"))
	if err != nil || !strings.Contains(string(b), "hikma") {
		t.Fatalf("fish file missing or empty: %v", err)
	}
}

func TestCompletionInstallNeedsConfirmationWhenNotInteractive(t *testing.T) {
	home := fakeHome(t)
	_, err := runCompletion(t, "zsh", "--install")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("want confirmation error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".zshrc")); statErr == nil {
		t.Fatal("wrote without confirmation")
	}
}

func TestHasActiveLineIgnoresComments(t *testing.T) {
	line := guardedSource("zsh")
	if hasActiveLine("# "+line+"\n", line) {
		t.Error("commented-out line counted as installed")
	}
	if !hasActiveLine("a\n  "+line+"  \nb", line) {
		t.Error("active line not found")
	}
}

func TestCompletionBashTargetFollowsThePlatform(t *testing.T) {
	home := fakeHome(t)
	tg, err := completionTargetFor("bash")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".bashrc")
	if runtime.GOOS == "darwin" {
		want = filepath.Join(home, ".bash_profile")
	}
	if tg.path != want {
		t.Errorf("bash target = %s, want %s (it must not depend on .bashrc already existing)", tg.path, want)
	}
}

func TestCompletionAlreadySetUpSaysHowToUseItInTheOpenWindow(t *testing.T) {
	home := fakeHome(t)
	rc := filepath.Join(home, ".zshrc")
	body := "export ZSH=x\nsource $ZSH/oh-my-zsh.sh\n" + completionMarker + "\n" + guardedSource("zsh") + "\n"
	if err := os.WriteFile(rc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runCompletion(t, "zsh", "--install", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"already set up in " + rc, "new window", "source " + rc} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "cannot register") {
		t.Errorf("compinit comes first here, so there is nothing to warn about:\n%s", out)
	}
	if after, _ := os.ReadFile(rc); string(after) != body {
		t.Error("an already set up file must not be rewritten")
	}
}

func TestCompletionWarnsWhenCompinitComesAfterTheLine(t *testing.T) {
	home := fakeHome(t)
	rc := filepath.Join(home, ".zshrc")
	body := completionMarker + "\n" + guardedSource("zsh") + "\nautoload -U compinit && compinit\n"
	if err := os.WriteFile(rc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runCompletion(t, "zsh", "--install", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no 'compinit'") || !strings.Contains(out, "cannot register") {
		t.Errorf("the order problem should be named:\n%s", out)
	}
	if strings.Contains(out, "open a new window") {
		t.Errorf("a new window would not fix this, so do not suggest it:\n%s", out)
	}
}

func TestCompinitBeforeIgnoresComments(t *testing.T) {
	line := guardedSource("zsh")
	if compinitBefore("# compinit later\n"+line, line) {
		t.Error("a comment is not compinit")
	}
	if !compinitBefore("autoload -U compinit && compinit\n"+line, line) {
		t.Error("compinit before the line counts")
	}
	if compinitBefore(line+"\ncompinit", line) {
		t.Error("compinit after the line does not count")
	}
}

func TestReloadCommandPerShell(t *testing.T) {
	for shell, want := range map[string]string{"zsh": "source /h/.zshrc", "bash": "source /h/.zshrc", "fish": "exec fish", "powershell": ". $PROFILE"} {
		if got := reloadCommand(shell, "/h/.zshrc"); got != want {
			t.Errorf("%s: got %q, want %q", shell, got, want)
		}
	}
}
