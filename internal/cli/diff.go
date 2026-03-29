package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/markdown"
	"github.com/andinger/tally-form-cli/internal/tally"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <file.md> [form-id]",
		Short: "Compare local Markdown with a Tally form",
		Long:  "Downloads the Tally form and compares it with the local Markdown file. If form-id is omitted, uses form_id from frontmatter.",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runDiff,
	}
}

func runDiff(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	localForm, err := markdown.Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse local: %w", err)
	}

	// Determine form ID
	formID := localForm.FormID
	if len(args) > 1 {
		formID = args[1]
	}
	if formID == "" {
		return fmt.Errorf("no form_id in frontmatter and none provided as argument")
	}

	cfg, err := config.Load(localForm.Settings)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if tokenFlag != "" {
		cfg.Token = tokenFlag
	}
	if cfg.Token == "" {
		return fmt.Errorf("no API token")
	}

	// Fetch remote form
	client := tally.NewClient(cfg.BaseURL, cfg.Token)
	tf, err := client.GetForm(formID)
	if err != nil {
		return fmt.Errorf("get form: %w", err)
	}

	remoteForm, err := tally.Decompile(tf)
	if err != nil {
		return fmt.Errorf("decompile: %w", err)
	}

	// Generate markdown for both
	localMD := markdown.Write(localForm)
	remoteMD := markdown.Write(remoteForm)

	if localMD == remoteMD {
		fmt.Fprintln(os.Stderr, "No differences found.")
		return nil
	}

	// Simple line-by-line diff
	localLines := strings.Split(localMD, "\n")
	remoteLines := strings.Split(remoteMD, "\n")

	fmt.Println("--- local (from file)")
	fmt.Println("+++ remote (from Tally)")
	fmt.Println()

	maxLines := len(localLines)
	if len(remoteLines) > maxLines {
		maxLines = len(remoteLines)
	}

	for i := 0; i < maxLines; i++ {
		local := ""
		remote := ""
		if i < len(localLines) {
			local = localLines[i]
		}
		if i < len(remoteLines) {
			remote = remoteLines[i]
		}
		if local != remote {
			if local != "" {
				fmt.Printf("- %s\n", local)
			}
			if remote != "" {
				fmt.Printf("+ %s\n", remote)
			}
		}
	}

	return nil
}
