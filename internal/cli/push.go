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

var (
	dryRun      bool
	forceCreate bool
)

func newPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <file.md>",
		Short: "Push a Markdown form to Tally (upsert)",
		Long:  "Creates a new form or updates an existing one based on form_id in frontmatter. Use --create to force creating a new form.",
		Args:  cobra.ExactArgs(1),
		RunE:  runPush,
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print JSON payload without calling API")
	cmd.Flags().BoolVar(&forceCreate, "create", false, "force creating a new form (ignores form_id)")
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

	cfg, err := config.Load(form.Settings)
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
		return fmt.Errorf("no API token configured (set in %s or TALLY_API_TOKEN)", config.ConfigPath())
	}

	client := tally.NewClient(cfg.BaseURL, cfg.Token)

	formURL := func(id string) string {
		host := "tally.so"
		if cfg.Domain != "" {
			host = cfg.Domain
		}
		return fmt.Sprintf("https://%s/r/%s", host, id)
	}

	if form.FormID != "" && !forceCreate {
		// Update existing form
		result, err := client.UpdateForm(form.FormID, req)
		if err != nil {
			return fmt.Errorf("update form: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Updated form %s → %s\n", result.ID, formURL(result.ID))
	} else {
		// Create new form
		result, err := client.CreateForm(req)
		if err != nil {
			return fmt.Errorf("create form: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Created form %s → %s\n", result.ID, formURL(result.ID))

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

	// Replace existing empty form_id or add new one
	if strings.Contains(frontmatter, "form_id:") {
		lines := strings.Split(frontmatter, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "form_id:") {
				lines[i] = fmt.Sprintf("form_id: %q", formID)
			}
		}
		frontmatter = strings.Join(lines, "\n")
		newContent := "---" + frontmatter + afterFrontmatter
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	newContent := "---" + frontmatter + fmt.Sprintf("\nform_id: %q", formID) + afterFrontmatter
	return os.WriteFile(filePath, []byte(newContent), 0644)
}
