package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andinger/tally-form-cli/internal/tally"
)

func TestDigitWidth(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 2}, // minimum is 2
		{1, 2},
		{9, 2},
		{10, 2},
		{99, 2},
		{100, 3},
		{999, 3},
		{1000, 4},
	}
	for _, tt := range tests {
		if got := digitWidth(tt.in); got != tt.want {
			t.Errorf("digitWidth(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestQuestionFilename(t *testing.T) {
	tests := []struct {
		name  string
		index int
		width int
		q     tally.SubmissionQuestion
		want  string
	}{
		{
			name:  "two-digit pad with title slug",
			index: 1, width: 2,
			q:    tally.SubmissionQuestion{ID: "qABC", Title: "Wie zufrieden sind Sie?"},
			want: "01-wie-zufrieden-sind-sie.md",
		},
		{
			name:  "three-digit pad",
			index: 7, width: 3,
			q:    tally.SubmissionQuestion{ID: "qXYZ", Title: "Empfehlung"},
			want: "007-empfehlung.md",
		},
		{
			name:  "name preferred over title",
			index: 2, width: 2,
			q:    tally.SubmissionQuestion{ID: "qDEF", Name: "role", Title: "What is your role?"},
			want: "02-role.md",
		},
		{
			name:  "falls back to ID when slug empty",
			index: 5, width: 2,
			q:    tally.SubmissionQuestion{ID: "qEMOJI", Title: "🎉🎉🎉"},
			want: "05-qEMOJI.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := questionFilename(tt.index, tt.width, tt.q)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountResponses(t *testing.T) {
	subs := []tally.Submission{
		{ID: "s1", Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: "yes"},
			{QuestionID: "q2", Answer: nil},
		}},
		{ID: "s2", Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: "no"},
		}},
		{ID: "s3", Responses: []tally.SubmissionResponse{}}, // no answers at all
	}
	if got := countResponses("q1", subs); got != 2 {
		t.Errorf("q1 responses = %d, want 2", got)
	}
	if got := countResponses("q2", subs); got != 0 {
		t.Errorf("q2 responses = %d, want 0 (Answer nil counted as no response)", got)
	}
	if got := countResponses("q3", subs); got != 0 {
		t.Errorf("q3 (absent) = %d, want 0", got)
	}
}

func TestAnswerForSubmission_FreeTextWrappedInCodeFence(t *testing.T) {
	sub := tally.Submission{
		ID: "s1",
		Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: "## not a heading\n---\nplain answer"},
		},
	}
	q := tally.SubmissionQuestion{ID: "q1", Type: "TEXTAREA"}
	got := answerForSubmission(sub, q, nil)
	want := "```\n## not a heading\n---\nplain answer\n```"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnswerForSubmission_InputTextWrapped(t *testing.T) {
	sub := tally.Submission{
		ID: "s1",
		Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: "short"},
		},
	}
	q := tally.SubmissionQuestion{ID: "q1", Type: "INPUT_TEXT"}
	got := answerForSubmission(sub, q, nil)
	if !strings.HasPrefix(got, "```") {
		t.Errorf("INPUT_TEXT should be code-fenced, got: %q", got)
	}
}

func TestAnswerForSubmission_StructuredAnswerNotFenced(t *testing.T) {
	sub := tally.Submission{
		ID: "s1",
		Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: []any{"Option A"}},
		},
	}
	q := tally.SubmissionQuestion{ID: "q1", Type: "MULTIPLE_CHOICE"}
	got := answerForSubmission(sub, q, nil)
	if strings.HasPrefix(got, "```") {
		t.Errorf("MULTIPLE_CHOICE answer must not be code-fenced, got: %q", got)
	}
	if got != "Option A" {
		t.Errorf("got %q, want %q", got, "Option A")
	}
}

func TestAnswerForSubmission_EmailNotFenced(t *testing.T) {
	sub := tally.Submission{
		ID: "s1",
		Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: "user@example.com"},
		},
	}
	q := tally.SubmissionQuestion{ID: "q1", Type: "INPUT_EMAIL"}
	got := answerForSubmission(sub, q, nil)
	if strings.HasPrefix(got, "```") {
		t.Errorf("INPUT_EMAIL should not be fenced, got: %q", got)
	}
}

func TestAnswerForSubmission_MissingResponseReturnsPlaceholder(t *testing.T) {
	sub := tally.Submission{ID: "s1", Responses: []tally.SubmissionResponse{}}
	q := tally.SubmissionQuestion{ID: "q1", Type: "INPUT_TEXT"}
	got := answerForSubmission(sub, q, nil)
	if got != "_(keine Antwort)_" {
		t.Errorf("got %q, want %q", got, "_(keine Antwort)_")
	}
}

func TestAnswerForSubmission_EmptyStringReturnsPlaceholder(t *testing.T) {
	sub := tally.Submission{
		ID: "s1",
		Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: ""},
		},
	}
	q := tally.SubmissionQuestion{ID: "q1", Type: "TEXTAREA"}
	got := answerForSubmission(sub, q, nil)
	if got != "_(keine Antwort)_" {
		t.Errorf("got %q, want %q", got, "_(keine Antwort)_")
	}
}

func TestAnswerForSubmission_AdaptiveCodeFence(t *testing.T) {
	sub := tally.Submission{
		ID: "s1",
		Responses: []tally.SubmissionResponse{
			{QuestionID: "q1", Answer: "see ```bash\nls\n``` example"},
		},
	}
	q := tally.SubmissionQuestion{ID: "q1", Type: "TEXTAREA"}
	got := answerForSubmission(sub, q, nil)
	if !strings.HasPrefix(got, "````\n") || !strings.HasSuffix(got, "\n````") {
		t.Errorf("expected 4-backtick fence, got:\n%s", got)
	}
}

func TestBuildQuestionMarkdown_FullStructure(t *testing.T) {
	q := tally.SubmissionQuestion{
		ID:    "qROLE",
		Type:  "MULTIPLE_CHOICE",
		Title: "Was ist Ihre Rolle?",
		Name:  "role",
		Fields: []tally.SubmissionField{
			{UUID: "opt-a", BlockGroupUUID: "cg1"},
		},
	}
	cfg := questionConfig{
		IsRequired: true,
		Options: []optionEntry{
			{UUID: "opt-a", Label: "QA"},
			{UUID: "opt-b", Label: "DevOps"},
		},
	}
	subs := []tally.Submission{
		{ID: "s1", Responses: []tally.SubmissionResponse{
			{QuestionID: "qROLE", Answer: []any{"QA"}},
		}},
		{ID: "s2", Responses: []tally.SubmissionResponse{
			// did not answer qROLE
		}},
		{ID: "s3", Responses: []tally.SubmissionResponse{
			{QuestionID: "qROLE", Answer: []any{"DevOps"}},
		}},
	}
	md := buildQuestionMarkdown("form-1", q, cfg, subs, nil, len(subs))

	wantContains := []string{
		"form_id: form-1",
		"question_id: qROLE",
		"type: MULTIPLE_CHOICE",
		"title: Was ist Ihre Rolle?",
		"name: role",
		"is_required: true",
		"options:",
		"- uuid: opt-a",
		"label: QA",
		"num_responses: 2",
		"num_submissions: 3",
		"# Was ist Ihre Rolle?",
		"## s1\n\nQA",
		"## s2\n\n_(keine Antwort)_",
		"## s3\n\nDevOps",
	}
	for _, needle := range wantContains {
		if !strings.Contains(md, needle) {
			t.Errorf("markdown missing %q\n--- markdown ---\n%s", needle, md)
		}
	}
}

func TestBuildQuestionMarkdown_NoFormDataOmitsIsRequired(t *testing.T) {
	// When extractQuestionConfig had no blocks to work with, is_required must
	// not appear (asserting it was "false" would be a lie).
	q := tally.SubmissionQuestion{ID: "qX", Type: "INPUT_TEXT", Title: "Anmerkungen"}
	cfg := questionConfig{} // empty: no form data
	md := buildQuestionMarkdown("form-1", q, cfg, nil, nil, 0)
	if strings.Contains(md, "is_required") {
		t.Errorf("is_required should be omitted when no form data:\n%s", md)
	}
}

func TestWriteQuestionMarkdownFiles_WritesAllFiles(t *testing.T) {
	dir := t.TempDir()
	subs := &tally.SubmissionsResponse{
		Questions: []tally.SubmissionQuestion{
			{ID: "qA", Type: "INPUT_TEXT", Title: "Vorname"},
			{ID: "qB", Type: "TEXTAREA", Title: "Anmerkungen"},
		},
		Submissions: []tally.Submission{
			{ID: "s1", SubmittedAt: "2026-04-01T10:00:00Z", Responses: []tally.SubmissionResponse{
				{QuestionID: "qA", Answer: "Andi"},
				{QuestionID: "qB", Answer: "Alles gut"},
			}},
		},
	}
	if err := writeQuestionMarkdownFiles("form-7", dir, subs, nil, nil); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	wantFiles := []string{"01-vorname.md", "02-anmerkungen.md"}
	gotNames := make([]string, len(got))
	for i, e := range got {
		gotNames[i] = e.Name()
	}
	for _, want := range wantFiles {
		found := false
		for _, name := range gotNames {
			if name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %q in %v", want, gotNames)
		}
	}

	// Spot-check content of the second file (TEXTAREA → code-fenced answer).
	body, err := os.ReadFile(filepath.Join(dir, "02-anmerkungen.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(body), "```\nAlles gut\n```") {
		t.Errorf("expected fenced TEXTAREA answer, got:\n%s", body)
	}
}

func TestQuestionHeading_PrefersTitleThenNameThenID(t *testing.T) {
	cases := []struct {
		in   tally.SubmissionQuestion
		want string
	}{
		{tally.SubmissionQuestion{Name: "role", Title: "Was ist Ihre Rolle?"}, "Was ist Ihre Rolle?"},
		{tally.SubmissionQuestion{Name: "role"}, "role"},
		{tally.SubmissionQuestion{ID: "qX"}, "qX"},
	}
	for _, c := range cases {
		if got := questionHeading(c.in); got != c.want {
			t.Errorf("questionHeading(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildQuestionMarkdown_MatrixSerializesRows(t *testing.T) {
	q := tally.SubmissionQuestion{
		ID:    "qMTX",
		Type:  "MATRIX",
		Title: "Bewerten",
	}
	cfg := questionConfig{
		IsRequired:    true,
		MatrixColumns: []string{"Stimme zu", "Stimme nicht zu"},
		MatrixRows: []optionEntry{
			{UUID: "row-a", Label: "Teamkultur"},
			{UUID: "row-b", Label: "Prozesse"},
		},
	}
	subs := []tally.Submission{
		{ID: "s1", Responses: []tally.SubmissionResponse{
			{QuestionID: "qMTX", Answer: map[string]any{
				"row-a": []any{"Stimme zu"},
				"row-b": []any{"Stimme nicht zu"},
			}},
		}},
	}
	rowLabels := map[string]string{"row-a": "Teamkultur", "row-b": "Prozesse"}

	md := buildQuestionMarkdown("form-7", q, cfg, subs, rowLabels, 1)

	wantContains := []string{
		"matrix_columns:",
		"- Stimme zu",
		"- Stimme nicht zu",
		"matrix_rows:",
		"- uuid: row-a",
		"label: Teamkultur",
		"- uuid: row-b",
		"label: Prozesse",
		"## s1",
		"- **Prozesse:** Stimme nicht zu",
		"- **Teamkultur:** Stimme zu",
	}
	for _, needle := range wantContains {
		if !strings.Contains(md, needle) {
			t.Errorf("matrix markdown missing %q\n--- markdown ---\n%s", needle, md)
		}
	}
}

func TestWriteQuestionMarkdownFiles_UsesFormForRichFrontmatter(t *testing.T) {
	dir := t.TempDir()
	subs := &tally.SubmissionsResponse{
		Questions: []tally.SubmissionQuestion{
			{
				ID:    "qROLE",
				Type:  "MULTIPLE_CHOICE",
				Title: "Rolle?",
				Fields: []tally.SubmissionField{
					{UUID: "opt-a", BlockGroupUUID: "cg1"},
				},
			},
		},
		Submissions: []tally.Submission{
			{ID: "s1", Responses: []tally.SubmissionResponse{
				{QuestionID: "qROLE", Answer: []any{"QA"}},
			}},
		},
	}
	form := &tally.TallyForm{
		Blocks: []tally.TallyBlock{
			{UUID: "opt-a", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "cg1", Payload: map[string]any{
				"text":       "QA",
				"isRequired": true,
			}},
			{UUID: "opt-b", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "cg1", Payload: map[string]any{
				"text":       "DevOps",
				"isRequired": true,
			}},
		},
	}
	if err := writeQuestionMarkdownFiles("form-1", dir, subs, form, nil); err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "01-rolle.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	got := string(body)
	for _, needle := range []string{
		"is_required: true",
		"options:",
		"- uuid: opt-a",
		"label: QA",
		"- uuid: opt-b",
		"label: DevOps",
	} {
		if !strings.Contains(got, needle) {
			t.Errorf("file missing %q in:\n%s", needle, got)
		}
	}
}

func TestHasFormData(t *testing.T) {
	tests := []struct {
		name string
		cfg  questionConfig
		want bool
	}{
		{"empty", questionConfig{}, false},
		{"required true", questionConfig{IsRequired: true}, true},
		{"placeholder", questionConfig{Placeholder: "x"}, true},
		{"options", questionConfig{Options: []optionEntry{{UUID: "u", Label: "l"}}}, true},
		{"matrix rows", questionConfig{MatrixRows: []optionEntry{{UUID: "r"}}}, true},
		{"matrix cols only", questionConfig{MatrixColumns: []string{"c"}}, true},
		{"scale", questionConfig{ScaleStart: intPtr(0)}, true},
		{"stars", questionConfig{Stars: intPtr(5)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasFormData(tt.cfg); got != tt.want {
				t.Errorf("hasFormData(%+v) = %v, want %v", tt.cfg, got, tt.want)
			}
		})
	}
}
