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

The first public release channel is GitHub Releases. Package-manager-specific
upgrade commands can be added after the release flow is proven.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}
}

func runUpdate() error {
	fmt.Println("Download the latest release from:")
	fmt.Println("  https://github.com/hasankhatib/hikma-ai/releases/latest")
	fmt.Println()
	fmt.Println("Then replace the hikma binary in your PATH.")
	return nil
}
