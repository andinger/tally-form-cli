package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
)

func newPrepareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prepare <file.md>",
		Short: "Merge global settings into the Markdown frontmatter",
		Long:  "Writes workspace, logo, cover, password, primary_color, and language from global config into the form's YAML frontmatter.",
		Args:  cobra.ExactArgs(1),
		RunE:  runPrepare,
	}
}

func runPrepare(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	text := string(content)

	// Ensure frontmatter exists
	if !strings.HasPrefix(text, "---") {
		text = "---\n---\n" + text
	}

	// Find frontmatter boundaries
	rest := text[3:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return fmt.Errorf("no frontmatter closing found")
	}

	fmContent := rest[:idx]
	afterFM := rest[idx:]

	// Merge settings into frontmatter
	// String fields: only add if not already present (idempotent)
	stringFields := []struct {
		key   string
		value string
	}{
		{"workspace", cfg.Workspace},
		{"logo", cfg.Logo},
		{"cover", cfg.Cover},
		{"password", cfg.Password},
		{"primary_color", cfg.PrimaryColor},
		{"language", cfg.Language},
	}

	for _, f := range stringFields {
		if f.value == "" {
			continue
		}
		if !containsFMKey(fmContent, f.key) {
			fmContent += fmt.Sprintf("\n%s: %q", f.key, f.value)
		}
	}

	// Boolean fields: only add if not already present (idempotent)
	boolFields := []struct {
		key   string
		value string
	}{
		{"has_progress_bar", fmt.Sprintf("%v", cfg.Settings["hasProgressBar"])},
		{"has_partial_submissions", fmt.Sprintf("%v", cfg.Settings["hasPartialSubmissions"])},
		{"save_for_later", fmt.Sprintf("%v", cfg.Settings["saveForLater"])},
	}

	for _, f := range boolFields {
		if f.value == "<nil>" {
			continue
		}
		if !containsFMKey(fmContent, f.key) {
			fmContent += fmt.Sprintf("\n%s: %s", f.key, f.value)
		}
	}

	newContent := "---" + fmContent + afterFM
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Prepared %s with global settings\n", filePath)
	return nil
}

func containsFMKey(fm, key string) bool {
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+":") {
			return true
		}
	}
	return false
}
