package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/tally"
)

var (
	outputFormat string
	outputDir    string
)

func newSubmissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submissions <form-id>",
		Short: "Download form submissions",
		Long: "Downloads submissions of a Tally form.\n\n" +
			"Default: prints CSV (one row per submission) to stdout.\n" +
			"With --format json: prints the full submissions JSON to stdout.\n" +
			"With --output <dir>: writes one Markdown file per submission into the directory,\n" +
			"named <submission-id>.md, with submission metadata in the YAML frontmatter.",
		Args: cobra.ExactArgs(1),
		RunE: runSubmissions,
	}
	cmd.Flags().StringVar(&outputFormat, "format", "csv", "stdout output format (csv or json) — ignored when --output is set")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "write one Markdown file per submission into this directory")
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

	// Matrix row labels: the list-submissions endpoint does not include them,
	// so fetch the form once and build a uuid → label map from MATRIX_ROW
	// blocks. This single extra call covers any number of submissions.
	rowLabels := make(map[string]string)
	if hasMatrixQuestion(subs.Questions) {
		form, err := client.GetForm(formID)
		if err == nil {
			for uuid, label := range matrixRowLabelsFromBlocks(form.Blocks) {
				rowLabels[uuid] = label
			}
		}
	}

	// --output writes one Markdown file per submission into the directory.
	// This takes precedence over --format.
	if outputDir != "" {
		return writeSubmissionMarkdownFiles(formID, outputDir, subs, rowLabels)
	}

	if outputFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(subs)
	}

	// CSV output
	w := csv.NewWriter(os.Stdout)

	// Build header using best-effort question labels.
	header := []string{"submission_id", "submitted_at"}
	qMap := make(map[string]int) // questionId → column index
	for _, q := range subs.Questions {
		qMap[q.ID] = len(header)
		header = append(header, questionLabel(q))
	}
	_ = w.Write(header)

	for _, sub := range subs.Submissions {
		row := make([]string, len(header))
		row[0] = sub.ID
		row[1] = sub.SubmittedAt
		for _, resp := range sub.Responses {
			idx, ok := qMap[resp.QuestionID]
			if !ok {
				continue
			}
			row[idx] = formatAnswer(resp, rowLabels)
		}
		_ = w.Write(row)
	}

	w.Flush()
	return w.Error()
}

// questionLabel picks a human-readable label for a question column header,
// preferring Name (author-supplied short label) then Title, with a fallback to
// the question ID so every column has something.
func questionLabel(q tally.SubmissionQuestion) string {
	if q.Name != "" {
		return q.Name
	}
	if q.Title != "" {
		return q.Title
	}
	return q.ID
}

// writeSubmissionMarkdownFiles writes one Markdown file per submission into
// dir. Each file is named <submission-id>.md, with submission metadata in the
// YAML frontmatter (submission_id, form_id, submitted_at, is_completed) and
// one H2 + answer block per question.
//
// Files and signatures are rendered as Markdown links (images for image mime
// types). Unanswered questions are skipped so the output stays compact.
func writeSubmissionMarkdownFiles(formID, dir string, subs *tally.SubmissionsResponse, rowLabels map[string]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	for _, sub := range subs.Submissions {
		md := buildSubmissionMarkdown(formID, sub, subs.Questions, rowLabels)
		path := filepath.Join(dir, sub.ID+".md")
		if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	fmt.Fprintf(os.Stderr, "Wrote %d submission(s) to %s\n", len(subs.Submissions), dir)
	return nil
}

// buildSubmissionMarkdown renders a single submission as Markdown. Questions
// keep the order from the form (subs.Questions), not the order responses were
// answered in, so comparing two submissions is straightforward.
func buildSubmissionMarkdown(formID string, sub tally.Submission, order []tally.SubmissionQuestion, rowLabels map[string]string) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("submission_id: %q\n", sub.ID))
	b.WriteString(fmt.Sprintf("form_id: %q\n", formID))
	if sub.SubmittedAt != "" {
		b.WriteString(fmt.Sprintf("submitted_at: %q\n", sub.SubmittedAt))
	}
	b.WriteString(fmt.Sprintf("is_completed: %t\n", sub.IsCompleted))
	b.WriteString("---\n\n")

	b.WriteString(fmt.Sprintf("# Submission %s\n\n", sub.ID))

	// Index responses by questionId for quick lookup, preserving the form's
	// question order in the output.
	respByQID := make(map[string]tally.SubmissionResponse, len(sub.Responses))
	for _, r := range sub.Responses {
		respByQID[r.QuestionID] = r
	}

	for _, q := range order {
		resp, ok := respByQID[q.ID]
		if !ok {
			continue // unanswered question — omit for compactness
		}
		heading := questionLabel(q)
		b.WriteString("## ")
		b.WriteString(heading)
		b.WriteString("\n\n")
		b.WriteString(markdownAnswer(resp, q.Type, rowLabels))
		b.WriteString("\n\n")
	}

	return b.String()
}

// markdownAnswer renders a response's Answer in a Markdown-friendly way:
//   - scalar → plain line
//   - single-select (MULTIPLE_CHOICE, DROPDOWN) one-element list → plain line
//   - multi-select (CHECKBOXES) list of scalars → bulleted list
//   - list of file objects → bulleted list of [name](url) links (images inline)
//   - matrix map → bulleted list of "- Row: Value" with resolved row labels
//
// qType is the Tally questionType (e.g. "MULTIPLE_CHOICE", "CHECKBOXES") and
// may be empty — in that case multi-element lists become bullet lists and
// single-element lists render plain.
func markdownAnswer(resp tally.SubmissionResponse, qType string, rowLabels map[string]string) string {
	if resp.Answer == nil {
		if resp.FormattedAnswer != "" {
			return resp.FormattedAnswer
		}
		return "_(no answer)_"
	}

	switch val := resp.Answer.(type) {
	case string:
		if val == "" {
			return "_(empty)_"
		}
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)

	case []any:
		if len(val) == 0 {
			return "_(empty)_"
		}
		// Files / signatures: list of objects with url.
		if obj, isObj := val[0].(map[string]any); isObj {
			if _, hasURL := obj["url"]; hasURL {
				return fileListMarkdown(val)
			}
		}
		// Single-select question types (or any single-item list) → inline the
		// one value without a bullet, for readability.
		if len(val) == 1 || qType == "MULTIPLE_CHOICE" || qType == "DROPDOWN" {
			return stringifyAnswer(val[0], rowLabels)
		}
		// Multi-select: bulleted list.
		var out strings.Builder
		for _, item := range val {
			out.WriteString("- ")
			out.WriteString(stringifyAnswer(item, rowLabels))
			out.WriteString("\n")
		}
		return strings.TrimRight(out.String(), "\n")

	case map[string]any:
		// Matrix: sorted bulleted list.
		type entry struct{ label, value string }
		entries := make([]entry, 0, len(val))
		for uuid, v := range val {
			label := rowLabels[uuid]
			if label == "" {
				label = uuid
			}
			entries = append(entries, entry{label: label, value: stringifyAnswer(v, rowLabels)})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].label < entries[j].label })
		var out strings.Builder
		for _, e := range entries {
			out.WriteString(fmt.Sprintf("- **%s:** %s\n", e.label, e.value))
		}
		return strings.TrimRight(out.String(), "\n")
	}

	// Unexpected shape — serialize as JSON so nothing is silently lost.
	b, _ := json.Marshal(resp.Answer)
	return "```json\n" + string(b) + "\n```"
}

// fileListMarkdown renders an array of Tally file/signature objects as a
// bulleted Markdown list. Image mime types are embedded with ![], other
// files get a plain [name](url) link.
func fileListMarkdown(items []any) string {
	var out strings.Builder
	for _, it := range items {
		obj, ok := it.(map[string]any)
		if !ok {
			continue
		}
		url, _ := obj["url"].(string)
		name, _ := obj["name"].(string)
		mime, _ := obj["mimeType"].(string)
		if name == "" {
			name = url
		}
		out.WriteString("- ")
		if strings.HasPrefix(mime, "image/") {
			out.WriteString("!")
		}
		out.WriteString(fmt.Sprintf("[%s](%s)\n", name, url))
	}
	return strings.TrimRight(out.String(), "\n")
}

// hasMatrixQuestion returns true when at least one question in the list is of
// type MATRIX. Used to skip the GetForm call when no matrix labels are needed.
func hasMatrixQuestion(qs []tally.SubmissionQuestion) bool {
	for _, q := range qs {
		if q.Type == "MATRIX" {
			return true
		}
	}
	return false
}

// matrixRowLabelsFromBlocks walks a form's blocks and returns a map from each
// MATRIX_ROW block's uuid to the visible row label (extracted from its
// safeHTMLSchema payload). These uuids are the keys Tally uses inside a
// matrix answer object.
func matrixRowLabelsFromBlocks(blocks []tally.TallyBlock) map[string]string {
	labels := make(map[string]string)
	for _, b := range blocks {
		if b.Type != "MATRIX_ROW" {
			continue
		}
		if label := extractSafeHTML(b.Payload["safeHTMLSchema"]); label != "" {
			labels[b.UUID] = label
		}
	}
	return labels
}

// extractSafeHTML flattens a Tally safeHTMLSchema ([[segment, styles?], ...])
// into plain text, dropping styling markers. Returns "" for nil or empty
// schemas.
func extractSafeHTML(v any) string {
	schema, ok := v.([]any)
	if !ok {
		return ""
	}
	var out strings.Builder
	for _, seg := range schema {
		arr, ok := seg.([]any)
		if !ok || len(arr) == 0 {
			continue
		}
		if s, ok := arr[0].(string); ok {
			out.WriteString(s)
		}
	}
	return out.String()
}

// formatAnswer turns the raw Answer value into a single human-readable CSV
// cell. It handles all answer shapes the Tally API returns:
//   - string / number / bool → value as-is
//   - []string (or []any of strings) → comma-joined
//   - []any of file objects → URLs joined with newlines (each file's URL on its own line)
//   - map[string]any (matrix) → "Row: Value | Row: Value", using rowLabels when available
//
// FormattedAnswer is used as a fallback only, since the list endpoint often
// returns it empty.
func formatAnswer(resp tally.SubmissionResponse, rowLabels map[string]string) string {
	if resp.Answer == nil {
		return resp.FormattedAnswer
	}
	return stringifyAnswer(resp.Answer, rowLabels)
}

func stringifyAnswer(v any, rowLabels map[string]string) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		// JSON numbers decode as float64. Render integers without trailing .0.
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case []any:
		return stringifyList(val, rowLabels)
	case map[string]any:
		return stringifyMatrix(val, rowLabels)
	default:
		// Unexpected shape — fall back to JSON so nothing is silently lost.
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func stringifyList(items []any, rowLabels map[string]string) string {
	if len(items) == 0 {
		return ""
	}

	// File / signature: list of objects with url + name.
	if _, isFile := items[0].(map[string]any); isFile {
		var urls []string
		for _, it := range items {
			obj, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if u, ok := obj["url"].(string); ok {
				urls = append(urls, u)
			}
		}
		if len(urls) > 0 {
			return strings.Join(urls, "\n")
		}
	}

	// List of scalars (choice/checkbox/dropdown).
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, stringifyAnswer(it, rowLabels))
	}
	return strings.Join(parts, ", ")
}

func stringifyMatrix(m map[string]any, rowLabels map[string]string) string {
	// Sort row UUIDs so CSV output is stable across runs. Within a single
	// submission we do not know the author-intended row order, so stable
	// alphabetical-by-label is the best we can offer here.
	type entry struct{ label, value string }
	entries := make([]entry, 0, len(m))
	for uuid, v := range m {
		label := rowLabels[uuid]
		if label == "" {
			label = uuid
		}
		entries = append(entries, entry{label: label, value: stringifyAnswer(v, rowLabels)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].label < entries[j].label })

	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		parts = append(parts, fmt.Sprintf("%s: %s", e.label, e.value))
	}
	return strings.Join(parts, " | ")
}
