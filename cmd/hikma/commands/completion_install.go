package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const completionMarker = "# hikma shell completion"

// completionTarget is where a shell's completion gets installed. Either the
// whole file is ours (fish) or we append a marked line to a file the user owns.
type completionTarget struct {
	path   string
	line   string // appended to path; empty when the file is ours entirely
	create bool   // the file is written whole from the generated script
}

func detectShell() string {
	switch filepath.Base(os.Getenv("SHELL")) {
	case "zsh":
		return "zsh"
	case "bash":
		return "bash"
	case "fish":
		return "fish"
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return ""
}

func completionTargetFor(shell string) (completionTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return completionTarget{}, fmt.Errorf("find your home folder: %w", err)
	}
	switch shell {
	case "zsh":
		return completionTarget{path: filepath.Join(home, ".zshrc"), line: guardedSource("zsh")}, nil
	case "bash":
		// macOS terminals start login shells, which read .bash_profile. Elsewhere
		// interactive shells read .bashrc, even when it does not exist yet.
		rc := filepath.Join(home, ".bashrc")
		if runtime.GOOS == "darwin" {
			rc = filepath.Join(home, ".bash_profile")
		}
		return completionTarget{path: rc, line: guardedSource("bash")}, nil
	case "fish":
		return completionTarget{path: filepath.Join(home, ".config", "fish", "completions", "hikma.fish"), create: true}, nil
	case "powershell":
		profile, err := powershellProfile()
		if err != nil {
			return completionTarget{}, err
		}
		return completionTarget{path: profile, line: "if (Get-Command hikma -ErrorAction SilentlyContinue) { hikma completion powershell | Out-String | Invoke-Expression }"}, nil
	}
	return completionTarget{}, fmt.Errorf("unknown shell %q", shell)
}

// guardedSource skips the line quietly when hikma is no longer on PATH, so
// uninstalling hikma never leaves an error on every new shell.
func guardedSource(shell string) string {
	return "command -v hikma >/dev/null 2>&1 && source <(hikma completion " + shell + ")"
}

// hasActiveLine reports whether a line of the file is exactly line, ignoring
// comments, so a commented-out copy does not count as installed.
func hasActiveLine(content, line string) bool {
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

func powershellProfile() (string, error) {
	for _, bin := range []string{"pwsh", "powershell"} {
		if p, err := exec.LookPath(bin); err == nil {
			out, err := exec.Command(p, "-NoProfile", "-Command", "$PROFILE").Output()
			if err == nil && strings.TrimSpace(string(out)) != "" {
				return strings.TrimSpace(string(out)), nil
			}
		}
	}
	return "", fmt.Errorf("could not find your PowerShell profile; add this line to it yourself:\n  hikma completion powershell | Out-String | Invoke-Expression")
}

func generateCompletion(cmd *cobra.Command, shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(w)
	case "zsh":
		return cmd.Root().GenZshCompletion(w)
	case "fish":
		return cmd.Root().GenFishCompletion(w, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(w)
	}
	return fmt.Errorf("unknown shell %q", shell)
}

// reloadCommand is how to load the completion into the shell that is already open.
func reloadCommand(shell, path string) string {
	switch shell {
	case "powershell":
		return ". $PROFILE"
	case "fish":
		return "exec fish"
	}
	return "source " + path
}

// compinitBefore reports whether zsh's completion system is started before the
// hikma line. The hikma script registers itself with compdef, which does not
// exist until compinit has run (oh-my-zsh runs it when it is sourced), so a hikma
// line above it fails with "command not found: compdef" when the shell starts.
func compinitBefore(content, line string) bool {
	before, _, found := strings.Cut(content, line)
	if !found {
		return false
	}
	for _, l := range strings.Split(before, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") {
			continue
		}
		if strings.Contains(l, "compinit") || strings.Contains(l, "oh-my-zsh.sh") {
			return true
		}
	}
	return false
}

// printCompletionAdvice follows "already set up". A line in the shell's startup
// file only counts in windows opened after it was added, so an open window needs
// a reload, and a zsh file that starts compinit after the hikma line needs
// reordering. Saying "already set up" and stopping left people with a completion
// that did nothing and no hint why.
func printCompletionAdvice(out io.Writer, shell string, t completionTarget, content string) {
	if shell == "zsh" && !t.create && !compinitBefore(content, t.line) {
		fmt.Fprintf(out, "But no 'compinit' (or oh-my-zsh) comes before it in %s, so zsh cannot register the completion; new shells may print \"command not found: compdef\".\n", t.path)
		fmt.Fprintln(out, "Move the lines below the one that starts completion, for example 'autoload -U compinit && compinit'.")
		return
	}
	fmt.Fprintf(out, "It only works in terminal windows opened after that. If Tab completes nothing in this one, open a new window or run:\n  %s\n", reloadCommand(shell, t.path))
}

// installCompletion shows exactly what it will change, asks, then writes.
// It never touches anything without a yes (or --yes), and never twice.
func installCompletion(cmd *cobra.Command, shell string, assumeYes bool) error {
	out := cmd.OutOrStdout()
	t, err := completionTargetFor(shell)
	if err != nil {
		return err
	}

	var script bytes.Buffer
	if t.create {
		if err := generateCompletion(cmd, shell, &script); err != nil {
			return err
		}
	}
	existing, readErr := os.ReadFile(t.path)
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read %s: %w", t.path, readErr)
	}
	if t.create && bytes.Equal(existing, script.Bytes()) {
		fmt.Fprintf(out, "Completion for %s is already installed (%s).\n", shell, t.path)
		printCompletionAdvice(out, shell, t, string(existing))
		return nil
	}
	if !t.create && hasActiveLine(string(existing), t.line) {
		fmt.Fprintf(out, "Completion for %s is already set up in %s.\n", shell, t.path)
		printCompletionAdvice(out, shell, t, string(existing))
		return nil
	}

	if t.create {
		verb := "create"
		if len(existing) > 0 {
			verb = "replace"
		}
		fmt.Fprintf(out, "This will %s %s with the %s completion script.\n", verb, t.path, shell)
	} else {
		fmt.Fprintf(out, "This will add these lines to %s:\n  %s\n  %s\n", t.path, completionMarker, t.line)
		if shell == "zsh" && !strings.Contains(string(existing), "compinit") && !strings.Contains(string(existing), "oh-my-zsh") {
			fmt.Fprintln(out, "Note: no compinit found in this file. zsh completion needs 'autoload -U compinit && compinit' before this line.")
		}
		if shell == "bash" {
			fmt.Fprintln(out, "Note: bash completion needs bash 4.1+ and the bash-completion package (macOS ships bash 3.2).")
		}
	}

	if !assumeYes {
		if !isInteractive() {
			return fmt.Errorf("confirmation required: re-run with --yes to install completion for %s", shell)
		}
		ok, err := prompter.Confirm("Install completion for "+shell+"?", !(t.create && len(existing) > 0))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Skipped. Print the script instead with: hikma completion "+shell)
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}
	if t.create {
		if err := os.WriteFile(t.path, script.Bytes(), 0o644); err != nil {
			return err
		}
	} else {
		f, err := os.OpenFile(t.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		prefix := ""
		if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
			prefix = "\n"
		}
		_, werr := fmt.Fprintf(f, "%s%s\n%s\n", prefix, completionMarker, t.line)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return werr
		}
	}
	fmt.Fprintf(out, "Installed. Open a new %s session to use it, or run:\n  %s\n", shell, reloadCommand(shell, t.path))
	return nil
}

// chooseShell uses $SHELL when it is a supported shell, otherwise asks.
func chooseShell(shells []string) (string, error) {
	if s := detectShell(); s != "" {
		return s, nil
	}
	if !isInteractive() {
		return "", fmt.Errorf("could not detect your shell; run: hikma completion --install <%s>", strings.Join(shells, "|"))
	}
	idx, err := prompter.Select("Which shell?", shells, 0)
	if err != nil {
		return "", err
	}
	return shells[idx], nil
}
