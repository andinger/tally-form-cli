package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/markdown"
	"github.com/andinger/tally-form-cli/internal/tally"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <form-id>",
		Short: "Download a Tally form as Markdown",
		Args:  cobra.ExactArgs(1),
		RunE:  runPull,
	}
}

func runPull(cmd *cobra.Command, args []string) error {
	formID := args[0]

	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if tokenFlag != "" {
		cfg.Token = tokenFlag
	}
	if cfg.Token == "" {
		return fmt.Errorf("no API token")
	}

	client := tally.NewClient(cfg.BaseURL, cfg.Token)
	tf, err := client.GetForm(formID)
	if err != nil {
		return fmt.Errorf("get form: %w", err)
	}

	form, err := tally.Decompile(tf)
	if err != nil {
		return fmt.Errorf("decompile: %w", err)
	}

	md := markdown.Write(form)
	fmt.Fprint(os.Stdout, md)
	return nil
}
