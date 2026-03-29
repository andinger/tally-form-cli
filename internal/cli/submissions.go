package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/tally"
)

var outputFormat string

func newSubmissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submissions <form-id>",
		Short: "Download form submissions",
		Args:  cobra.ExactArgs(1),
		RunE:  runSubmissions,
	}
	cmd.Flags().StringVar(&outputFormat, "format", "csv", "output format (csv or json)")
	return cmd
}

func runSubmissions(cmd *cobra.Command, args []string) error {
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
	subs, err := client.GetSubmissions(formID)
	if err != nil {
		return fmt.Errorf("get submissions: %w", err)
	}

	if outputFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(subs)
	}

	// CSV output
	w := csv.NewWriter(os.Stdout)

	// Header
	header := []string{"submission_id", "submitted_at"}
	qMap := make(map[string]int) // questionId → column index
	for i, q := range subs.Questions {
		header = append(header, q.Name)
		qMap[q.ID] = i + 2 // offset by 2 for id and timestamp
	}
	w.Write(header)

	// Rows
	for _, sub := range subs.Submissions {
		row := make([]string, len(header))
		row[0] = sub.ID
		row[1] = sub.SubmittedAt
		for _, resp := range sub.Responses {
			if idx, ok := qMap[resp.QuestionID]; ok {
				row[idx] = resp.FormattedAnswer
			}
		}
		w.Write(row)
	}

	w.Flush()
	return w.Error()
}
