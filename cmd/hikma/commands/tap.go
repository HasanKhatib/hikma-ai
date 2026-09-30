package commands

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/registry"
	"github.com/spf13/cobra"
)

func newTapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tap",
		Short: "Show registry connection and cache status",
		Args:  cobra.NoArgs,
		Long: `Show the registry URL, authenticated GitHub user, and local cache status.

Use this command to verify your registry connection and confirm your identity
before publishing skills.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTap()
		},
	}
}

func runTap() error {
	activeRegistry, registryErr := config.ResolveRegistry("")
	user, err := ghUser()
	if err != nil {
		user = "(not authenticated)"
	}

	cachePath, cacheAge, err := cacheInfo()
	var cacheDisplay string
	if err != nil {
		cacheDisplay = "(no cache)"
	} else {
		cacheDisplay = fmt.Sprintf("%s (updated %s ago)", tildeHome(cachePath), formatAge(cacheAge))
	}

	if registryErr != nil {
		fmt.Printf("Registry:   (not configured)\n")
		fmt.Printf("            set with: hikma config registry hasankhatib/ai\n")
	} else {
		fmt.Printf("Registry:   %s\n", activeRegistry.CloneURL())
	}
	fmt.Printf("User:       %s\n", user)
	fmt.Printf("Cache:      %s\n", cacheDisplay)
	return nil
}

// ghUser returns the login name of the authenticated GitHub user.
func ghUser() (string, error) {
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// cacheInfo returns the cache file path and how long ago it was last modified.
func cacheInfo() (string, time.Duration, error) {
	path, err := registry.CacheFilePath()
	if err != nil {
		return "", 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return path, 0, err
	}
	return path, time.Since(info.ModTime()), nil
}

// tildeHome replaces the user's home directory prefix with ~.
func tildeHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// formatAge returns a human-readable duration (e.g. "14 minutes", "2 hours").
func formatAge(d time.Duration) string {
	minutes := int(math.Round(d.Minutes()))
	if minutes < 1 {
		return "less than a minute"
	}
	if minutes == 1 {
		return "1 minute"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := int(math.Round(d.Hours()))
	if hours == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}
