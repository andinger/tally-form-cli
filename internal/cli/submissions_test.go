package cli

import (
	"strings"
	"testing"

	"github.com/andinger/tally-form-cli/internal/tally"
)

func TestStringifyAnswer(t *testing.T) {
	rows := map[string]string{
		"row-a": "Teamkultur",
		"row-b": "Prozesse",
	}

	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "Andi", "Andi"},
		{"bool true", true, "true"},
		{"int as float64", float64(15), "15"},
		{"float", 3.14, "3.14"},
		{"choice single", []any{"QA"}, "QA"},
		{"multi choice", []any{"TypeScript", "Go", "php"}, "TypeScript, Go, php"},
		{
			name: "file upload (single)",
			in: []any{map[string]any{
				"id":   "y1Br0p",
				"name": "Test.pdf",
				"url":  "https://storage.tally.so/private/Test.pdf?token=abc",
			}},
			want: "https://storage.tally.so/private/Test.pdf?token=abc",
		},
		{
			name: "file upload (two files, newline separated)",
			in: []any{
				map[string]any{"url": "https://example.com/a.pdf"},
				map[string]any{"url": "https://example.com/b.png"},
			},
			want: "https://example.com/a.pdf\nhttps://example.com/b.png",
		},
		{
			name: "matrix with known row labels, alphabetical by label",
			in: map[string]any{
				"row-b": []any{"OK"},
				"row-a": []any{"Großartig"},
			},
			want: "Prozesse: OK | Teamkultur: Großartig",
		},
		{
			name: "matrix falls back to uuid when label unknown",
			in: map[string]any{
				"unknown-uuid": []any{"Gut"},
			},
			want: "unknown-uuid: Gut",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: matrix expects answer values sorted by label, so reverse-
			// order input for "matrix with known row labels" must still
			// produce alphabetical-by-label output.
			got := stringifyAnswer(tt.in, rows)
			if got != tt.want {
				t.Errorf("\n  got:  %q\n  want: %q", got, tt.want)
			}
		})
	}
}

func TestMatrixRowLabelsFromBlocks(t *testing.T) {
	blocks := []tally.TallyBlock{
		{Type: "TITLE", UUID: "t1"},
		{Type: "MATRIX", UUID: "m1"},
		{
			Type: "MATRIX_ROW", UUID: "r1",
			Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Teamkultur"}},
			},
		},
		{
			Type: "MATRIX_ROW", UUID: "r2",
			Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Work-Life-Balance"}},
			},
		},
		{Type: "MATRIX_COLUMN", UUID: "c1", Payload: map[string]any{
			"safeHTMLSchema": []any{[]any{"ignored"}},
		}},
	}

	got := matrixRowLabelsFromBlocks(blocks)
	want := map[string]string{
		"r1": "Teamkultur",
		"r2": "Work-Life-Balance",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("labels[%q] = %q, want %q", k, got[k], v)
		}
	}
	if _, ok := got["c1"]; ok {
		t.Errorf("MATRIX_COLUMN should not appear in row label map")
	}
}

func TestFormatAnswerFallsBackToFormatted(t *testing.T) {
	// If Answer is nil, FormattedAnswer should be surfaced as-is.
	resp := tally.SubmissionResponse{
		QuestionID:      "Q1",
		Answer:          nil,
		FormattedAnswer: "fallback",
	}
	got := formatAnswer(resp, nil)
	if got != "fallback" {
		t.Errorf("fallback = %q, want fallback", got)
	}
}

func TestBuildSubmissionMarkdown(t *testing.T) {
	questions := []tally.SubmissionQuestion{
		{ID: "Q1", Type: "MULTIPLE_CHOICE", Title: "Role?"},
		{ID: "Q2", Type: "CHECKBOXES", Title: "Languages?"},
		{ID: "Q3", Type: "MATRIX", Title: "Rate areas"},
		{ID: "Q4", Type: "FILE_UPLOAD", Title: "Upload"},
		{ID: "Q5", Type: "INPUT_TEXT", Title: "Unanswered"},
	}
	sub := tally.Submission{
		ID:          "sub-1",
		SubmittedAt: "2026-04-17T08:15:03.000Z",
		IsCompleted: true,
		Responses: []tally.SubmissionResponse{
			{QuestionID: "Q1", Answer: []any{"QA"}},
			{QuestionID: "Q2", Answer: []any{"Go", "TypeScript"}},
			{QuestionID: "Q3", Answer: map[string]any{
				"row-a": []any{"Gut"},
				"row-b": []any{"OK"},
			}},
			{QuestionID: "Q4", Answer: []any{
				map[string]any{"name": "a.pdf", "url": "https://x/a.pdf", "mimeType": "application/pdf"},
				map[string]any{"name": "b.png", "url": "https://x/b.png", "mimeType": "image/png"},
			}},
			// Q5 intentionally unanswered
		},
	}
	rows := map[string]string{"row-a": "Teamkultur", "row-b": "Prozesse"}

	md := buildSubmissionMarkdown("form-42", sub, questions, rows)

	wantContains := []string{
		`submission_id: "sub-1"`,
		`form_id: "form-42"`,
		`submitted_at: "2026-04-17T08:15:03.000Z"`,
		`is_completed: true`,
		"# Submission sub-1",
		"## Role?\n\nQA\n\n",                               // single-select → plain, no bullet
		"## Languages?\n\n- Go\n- TypeScript\n\n",          // multi-select → bullets
		"- **Prozesse:** OK",                               // matrix sorted by label
		"- **Teamkultur:** Gut",                            // matrix sorted by label
		"- [a.pdf](https://x/a.pdf)",                       // file link
		"- ![b.png](https://x/b.png)",                      // image embed
	}
	for _, needle := range wantContains {
		if !strings.Contains(md, needle) {
			t.Errorf("markdown missing %q\n--- full markdown ---\n%s", needle, md)
		}
	}

	// Unanswered questions must be skipped entirely.
	if strings.Contains(md, "## Unanswered") {
		t.Errorf("unanswered question should be skipped — got heading in output:\n%s", md)
	}
}

func TestQuestionLabelPrefersNameThenTitle(t *testing.T) {
	cases := []struct {
		in   tally.SubmissionQuestion
		want string
	}{
		{tally.SubmissionQuestion{ID: "Q1", Name: "Role", Title: "What is your role?"}, "Role"},
		{tally.SubmissionQuestion{ID: "Q2", Title: "Email?"}, "Email?"},
		{tally.SubmissionQuestion{ID: "Q3"}, "Q3"},
	}
	for _, c := range cases {
		if got := questionLabel(c.in); got != c.want {
			t.Errorf("label(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}
