package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/tally"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <form-id>",
		Short: "Delete a Tally form",
		Long:  "Permanently deletes a Tally form. There is no undo and no confirmation prompt — this is a dev tool used in reset loops.",
		Args:  cobra.ExactArgs(1),
		RunE:  runDelete,
	}
}

func runDelete(cmd *cobra.Command, args []string) error {
	formID := args[0]

	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if tokenFlag != "" {
		cfg.Token = tokenFlag
	}
	if cfg.Token == "" {
		return fmt.Errorf("no API token configured (set in %s or pass --token)", config.ConfigPath())
	}

	client := tally.NewClient(cfg.BaseURL, cfg.Token)
	if err := client.DeleteForm(formID); err != nil {
		return fmt.Errorf("delete form: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Deleted form %s\n", formID)
	return nil
}
