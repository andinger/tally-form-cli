package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/markdown"
	"github.com/andinger/tally-form-cli/internal/tally"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <form-id> <file.md>",
		Short: "Update an existing Tally form from Markdown",
		Args:  cobra.ExactArgs(2),
		RunE:  runUpdate,
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print JSON payload without calling API")
	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	formID := args[0]
	content, err := os.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	form, err := markdown.Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	cfg, err := config.Load(configPath, form.Settings)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if tokenFlag != "" {
		cfg.Token = tokenFlag
	}

	compiler := tally.NewCompiler()
	req, err := compiler.Compile(form, cfg)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	if dryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(req)
	}

	if cfg.Token == "" {
		return fmt.Errorf("no API token")
	}

	client := tally.NewClient(cfg.BaseURL, cfg.Token)
	result, err := client.UpdateForm(formID, req)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Updated form %s\n", result.ID)
	return nil
}
