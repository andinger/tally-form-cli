package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configPath string
	tokenFlag  string
)

// NewRootCmd creates the root cobra command.
func NewRootCmd(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:   "tally",
		Short: "Markdown-to-Tally form builder",
		Long:  "Bidirectional conversion between Markdown and Tally.so forms.",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	root.PersistentFlags().StringVar(&configPath, "config", "", "path to project config YAML")
	root.PersistentFlags().StringVar(&tokenFlag, "token", "", "Tally API token (overrides config)")

	root.AddCommand(newPushCmd())
	root.AddCommand(newCreateCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newSubmissionsCmd())
	root.AddCommand(newReferenceCmd())

	return root
}
