package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/markdown"
	"github.com/andinger/tally-form-cli/internal/tally"
)

var dryRun bool

func newPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <file.md>",
		Short: "Create or update a Tally form from Markdown (upsert)",
		Args:  cobra.ExactArgs(1),
		RunE:  runPush,
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print JSON payload without calling API")
	return cmd
}

func runPush(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	form, err := markdown.Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse markdown: %w", err)
	}

	cfg, err := config.Load(configPath, form.Settings)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
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
		return fmt.Errorf("no API token configured (set in ~/.config/tally-form-cli/config.yaml or TALLY_API_TOKEN)")
	}

	client := tally.NewClient(cfg.BaseURL, cfg.Token)

	if form.FormID != "" {
		// Update existing form
		result, err := client.UpdateForm(form.FormID, req)
		if err != nil {
			return fmt.Errorf("update form: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Updated form %s\n", result.ID)
	} else {
		// Create new form
		result, err := client.CreateForm(req)
		if err != nil {
			return fmt.Errorf("create form: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Created form %s\n", result.ID)

		// Write back form_id to the markdown file
		err = writeBackFormID(filePath, string(content), result.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not write form_id back: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Wrote form_id: %q to %s\n", result.ID, filePath)
		}
	}

	return nil
}

func writeBackFormID(filePath, content, formID string) error {
	// Find the frontmatter closing --- and insert form_id before it
	if !strings.HasPrefix(content, "---") {
		return fmt.Errorf("no frontmatter found")
	}

	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return fmt.Errorf("no frontmatter closing found")
	}

	frontmatter := rest[:idx]
	afterFrontmatter := rest[idx:]

	// Check if form_id already exists
	if strings.Contains(frontmatter, "form_id:") {
		return nil // already has form_id
	}

	// Insert form_id
	newContent := "---" + frontmatter + fmt.Sprintf("\nform_id: %q", formID) + afterFrontmatter

	return os.WriteFile(filePath, []byte(newContent), 0644)
}
