package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	brewFormula      = "HasanKhatib/tap/hikma"
	installScriptURL = "https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh"
)

// Seams for tests: the running binary, the script download, and the process
// that does the work.
var (
	upgradeExe   = os.Executable
	upgradeFetch = func(url string) ([]byte, error) {
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s returned %s", url, resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	}
	upgradeExec = func(cmd *exec.Cmd) error { return cmd.Run() }
)

func newUpgradeCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade hikma to the latest release",
		Args:  cobra.NoArgs,
		Long: `Upgrade hikma to the latest release.

hikma works out how it was installed. With Homebrew it runs 'brew upgrade'.
Otherwise it runs the install script, which checks the download against the
release checksums and, when gh is signed in, its build attestation, and replaces
the binary where it is now. The plan is shown first and nothing changes until
you confirm (--yes skips the question).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cmd, yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation before replacing hikma")
	return cmd
}

// isHomebrewPath reports whether the binary lives in a Homebrew prefix.
func isHomebrewPath(p string) bool {
	p = filepath.ToSlash(p)
	for _, marker := range []string{"/Cellar/", "/homebrew/", "/Homebrew/", "/linuxbrew/"} {
		if strings.Contains(p, marker) {
			return true
		}
	}
	return false
}

func runUpgrade(cmd *cobra.Command, yes bool) error {
	out := cmd.OutOrStdout()
	exe, err := upgradeExe()
	if err != nil {
		return fmt.Errorf("find the running hikma: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	brew := isHomebrewPath(exe)
	if brew {
		fmt.Fprintf(out, "hikma at %s was installed with Homebrew.\nThis will run: brew upgrade %s\n", exe, brewFormula)
	} else {
		fmt.Fprintf(out, "This will replace %s with the latest release,\nusing the install script (checksum and attestation checked): %s\n", exe, installScriptURL)
	}

	if !yes {
		if !isInteractive() {
			return fmt.Errorf("not upgrading without a terminal to confirm: hikma upgrade --yes")
		}
		ok, err := prompter.Confirm("Upgrade now?", true)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Not upgrading.")
			return nil
		}
	}

	var run *exec.Cmd
	if brew {
		run = exec.Command("brew", "upgrade", brewFormula)
	} else {
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("bash"); err != nil {
				return fmt.Errorf("upgrading on Windows needs Git Bash; run the install script from it:\n  curl -fsSL %s | bash", installScriptURL)
			}
		}
		script, err := upgradeFetch(installScriptURL)
		if err != nil {
			return fmt.Errorf("download the install script: %w", err)
		}
		tmp, err := os.MkdirTemp("", "hikma-upgrade-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		path := filepath.Join(tmp, "install.sh")
		if err := os.WriteFile(path, script, 0o600); err != nil {
			return err
		}
		run = exec.Command("bash", path)
		run.Env = append(os.Environ(), "HIKMA_INSTALL_DIR="+filepath.Dir(exe))
	}
	run.Stdout = out
	run.Stderr = cmd.ErrOrStderr()
	run.Stdin = nil
	if err := upgradeExec(run); err != nil {
		return fmt.Errorf("the upgrade failed (hikma was not changed by hikma itself): %w", err)
	}
	return nil
}
