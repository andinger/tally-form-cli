package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tokenFlag string

// NewRootCmd creates the root cobra command.
func NewRootCmd(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:   "tally",
		Short: "Markdown-to-Tally form builder",
		Long:  "Bidirectional conversion between Markdown and Tally.so forms.",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	root.PersistentFlags().StringVar(&tokenFlag, "token", "", "Tally API token (overrides config)")

	root.AddCommand(newPushCmd())
	root.AddCommand(newPullCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newSubmissionsCmd())
	root.AddCommand(newPrepareCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newReferenceCmd())

	return root
}
