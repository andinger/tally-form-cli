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
		Long:  "Writes workspace, logo, password, primary_color, and domain from global config into the form's YAML frontmatter.",
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
	fields := []struct {
		key   string
		value string
	}{
		{"workspace", cfg.Workspace},
		{"logo", cfg.Logo},
		{"password", cfg.Password},
		{"primary_color", cfg.PrimaryColor},
		{"domain", cfg.Domain},
	}

	for _, f := range fields {
		if f.value == "" {
			continue
		}
		if containsFMKey(fmContent, f.key) {
			// Update existing key
			fmContent = replaceFMKey(fmContent, f.key, f.value)
		} else {
			// Add new key
			fmContent += fmt.Sprintf("\n%s: %q", f.key, f.value)
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

func replaceFMKey(fm, key, value string) string {
	lines := strings.Split(fm, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+":") {
			lines[i] = fmt.Sprintf("%s: %q", key, value)
		}
	}
	return strings.Join(lines, "\n")
}
