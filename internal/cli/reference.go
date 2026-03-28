package cli

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed reference.md
var referenceDoc string

func newReferenceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reference",
		Short: "Print CLI reference documentation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print(referenceDoc)
			return nil
		},
	}
}
