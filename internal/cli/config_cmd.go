package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show configuration file location",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(config.ConfigPath())
			return nil
		},
	}
}
