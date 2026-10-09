package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Show update instructions",
		Args:  cobra.NoArgs,
		Long: `Show update instructions for Hikma AI.

Hikma does not update itself. Use Homebrew where you installed it that way, or
re-run the install script.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}
}

func runUpdate() error {
	fmt.Println("Homebrew (macOS, Linux):")
	fmt.Println("  brew upgrade HasanKhatib/tap/hikma")
	fmt.Println()
	fmt.Println("Install script (any platform):")
	fmt.Println("  curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash")
	fmt.Println()
	fmt.Println("Releases: https://github.com/HasanKhatib/hikma-ai/releases/latest")
	return nil
}
