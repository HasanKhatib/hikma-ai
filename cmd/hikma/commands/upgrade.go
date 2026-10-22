package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	brewFormula      = "HasanKhatib/tap/hikma"
	installScriptURL = "https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh"
	latestReleaseURL = "https://api.github.com/repos/HasanKhatib/hikma-ai/releases/latest"
)

// Seams for tests: the running binary, downloads (the install script and the
// latest-release document), the process that does the work, and the check of what
// ended up installed.
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

	// upgradeBrewStable is the version the tap offers right now, read from
	// Homebrew after 'brew update'. Unlike 'brew outdated' it does not depend on
	// whether the formula is pinned.
	upgradeBrewStable = func() (string, error) {
		out, err := exec.Command("brew", "info", "--json=v2", brewFormula).Output()
		if err != nil {
			return "", err
		}
		var info struct {
			Formulae []struct {
				Versions struct {
					Stable string `json:"stable"`
				} `json:"versions"`
			} `json:"formulae"`
		}
		if err := json.Unmarshal(out, &info); err != nil || len(info.Formulae) == 0 || info.Formulae[0].Versions.Stable == "" {
			return "", errors.New("no version in the answer from brew info")
		}
		return info.Formulae[0].Versions.Stable, nil
	}

	// upgradeInstalled is the version that is installed after an upgrade: what
	// Homebrew reports, or what the replaced binary says about itself.
	upgradeInstalled = func(brew bool, exe string) (string, error) {
		if brew {
			out, err := exec.Command("brew", "list", "--versions", brewFormula).Output()
			if err != nil {
				return "", err
			}
			return newestVersion(strings.Fields(string(out))), nil
		}
		out, err := exec.Command(exe, "--version").Output()
		if err != nil {
			return "", err
		}
		f := strings.Fields(string(out))
		if len(f) == 0 {
			return "", errors.New("no version printed")
		}
		return f[len(f)-1], nil
	}
)

func newUpgradeCmd() *cobra.Command {
	var dryRun, yes, verbose bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade hikma to the latest release",
		Args:  cobra.NoArgs,
		Long: `Upgrade hikma to the latest release.

It shows the version you have and the latest release, and when they differ it
upgrades at once, then checks that the new version is what got installed.

hikma works out how it was installed. With Homebrew it runs 'brew update' and
'brew upgrade'. Otherwise it runs the install script, which checks the download
against the release checksums and, when gh is signed in, its build attestation,
and replaces the binary where it is now. What brew or the script print is held
back and shown only if a step fails; --verbose shows it as it happens. --dry-run
shows what would run and changes nothing.

This upgrades the hikma binary only. To update installed skills, run
'hikma skill update'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cmd, dryRun, verbose)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the versions and what would run, and change nothing")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show the output of brew or the install script, not only when it fails")
	// Upgrading no longer asks; the flag stays so scripts that pass it keep working.
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "accepted and ignored: upgrade no longer asks")
	_ = cmd.Flags().MarkHidden("yes")
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

// parseVersion reads "1.2.3" or "v1.2.3" (a "-rc.1" style suffix is allowed and
// ignored for ordering). ok is false for anything else, such as "dev".
func parseVersion(s string) (v [3]int, ok bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	s, _, _ = strings.Cut(s, "-")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

// compareVersions is -1, 0, or 1; ok is false when either side is not a version.
func compareVersions(a, b string) (c int, ok bool) {
	va, okA := parseVersion(a)
	vb, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for i := range va {
		switch {
		case va[i] < vb[i]:
			return -1, true
		case va[i] > vb[i]:
			return 1, true
		}
	}
	return 0, true
}

// newestVersion picks the highest version among fields (Homebrew lists every
// installed version after the name), or the last field when none parse.
func newestVersion(fields []string) string {
	best := ""
	for _, f := range fields {
		if _, ok := parseVersion(f); !ok {
			continue
		}
		if best == "" {
			best = f
		} else if c, _ := compareVersions(f, best); c > 0 {
			best = f
		}
	}
	if best == "" && len(fields) > 0 {
		return fields[len(fields)-1]
	}
	return best
}

// latestRelease asks GitHub for the newest published release.
func latestRelease() (string, error) {
	body, err := upgradeFetch(latestReleaseURL)
	if err != nil {
		return "", err
	}
	var rel struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &rel); err != nil || rel.Tag == "" {
		return "", errors.New("the answer had no release tag")
	}
	return strings.TrimPrefix(rel.Tag, "v"), nil
}

func runUpgrade(cmd *cobra.Command, dryRun, verbose bool) error {
	out := cmd.OutOrStdout()
	exe, err := upgradeExe()
	if err != nil {
		return fmt.Errorf("find the running hikma: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	brew := isHomebrewPath(exe)
	manual := "brew upgrade " + brewFormula
	if !brew {
		manual = "curl -fsSL " + installScriptURL + " | bash"
	}

	latest, err := latestRelease()
	if err != nil {
		return fmt.Errorf("check the latest release: %w\nto upgrade anyway, run: %s", err, manual)
	}
	current := version
	fmt.Fprintf(out, "Current version: %s\nLatest release:  %s\n", current, latest)

	if c, ok := compareVersions(current, latest); ok && c >= 0 {
		if c == 0 {
			fmt.Fprintf(out, "hikma %s is up to date.\n", current)
		} else {
			fmt.Fprintf(out, "hikma %s is newer than the latest release, so there is nothing to upgrade to.\n", current)
		}
		return nil
	}

	how := fmt.Sprintf("the install script (checksum and attestation checked), replacing %s", exe)
	if brew {
		how = "Homebrew (brew update, then brew upgrade " + brewFormula + ")"
	}
	if dryRun {
		fmt.Fprintf(out, "Would upgrade %s -> %s with %s. Nothing was changed.\n", current, latest, how)
		return nil
	}
	fmt.Fprintf(out, "Upgrading with %s...\n", how)

	if brew {
		updated := true
		if err := upgradeRun(cmd, verbose, exec.Command("brew", "update", "--quiet")); err != nil {
			updated = false
			fmt.Fprintf(out, "warning: brew update failed (%v); the tap may be out of date.\n", err)
		}
		// Only a tap that was just refreshed can say the formula is behind. After a
		// failed update it says nothing, so go on and let the check after the upgrade
		// speak.
		if updated {
			if stable, err := upgradeBrewStable(); err == nil {
				if c, ok := compareVersions(stable, latest); ok && c < 0 {
					return fmt.Errorf("hikma %s is out, but Homebrew still offers %s: the tap may not have the new formula yet, so try again in a few minutes", latest, stable)
				}
			}
		}
		if err := upgradeRun(cmd, verbose, exec.Command("brew", "upgrade", brewFormula)); err != nil {
			return fmt.Errorf("the upgrade failed (hikma was not changed by hikma itself): %w", err)
		}
	} else {
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("bash"); err != nil {
				return fmt.Errorf("upgrading on Windows needs Git Bash; run the install script from it:\n  %s", manual)
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
		run := exec.Command("bash", path)
		run.Env = append(os.Environ(), "HIKMA_INSTALL_DIR="+filepath.Dir(exe))
		if err := upgradeRun(cmd, verbose, run); err != nil {
			return fmt.Errorf("the upgrade failed (hikma was not changed by hikma itself): %w", err)
		}
	}

	// A release is published before the Homebrew formula catches up, so check that
	// what is installed now is what was announced.
	got, err := upgradeInstalled(brew, exe)
	if err != nil {
		return fmt.Errorf("the upgrade ran, but checking the new version failed: %w", err)
	}
	if c, ok := compareVersions(got, latest); !ok || c < 0 {
		if brew {
			return fmt.Errorf("hikma %s is out, but Homebrew still provides %s: the tap may not have the new formula yet, so try again in a few minutes", latest, got)
		}
		return fmt.Errorf("hikma %s is out, but %s is what is installed now", latest, got)
	}
	fmt.Fprintf(out, "  %s -> %s\nDone.\n", current, got)
	return nil
}

// upgradeRun runs one step of the upgrade. What the step prints is held back so
// the upgrade reads as a few lines, and shown only when the step fails, because
// then it says why. With verbose it goes straight through.
func upgradeRun(cmd *cobra.Command, verbose bool, c *exec.Cmd) error {
	if verbose {
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()
		return upgradeExec(c)
	}
	var held bytes.Buffer
	c.Stdout, c.Stderr = &held, &held
	err := upgradeExec(c)
	if err != nil && held.Len() > 0 {
		fmt.Fprint(cmd.ErrOrStderr(), held.String())
	}
	return err
}
