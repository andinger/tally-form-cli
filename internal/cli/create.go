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

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <file.md>",
		Short: "Create a new Tally form from Markdown (always creates, ignores form_id)",
		Args:  cobra.ExactArgs(1),
		RunE:  runCreate,
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print JSON payload without calling API")
	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	content, err := os.ReadFile(args[0])
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
	result, err := client.CreateForm(req)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created form %s\n", result.ID)
	fmt.Println(result.ID)
	return nil
}
