package cli

import (
	"testing"

	"github.com/andinger/tally-form-cli/internal/tally"
)

func TestExtractQuestionConfig_MultipleChoice(t *testing.T) {
	blocks := []tally.TallyBlock{
		{UUID: "title-1", Type: "TITLE", GroupUUID: "qg1", GroupType: "QUESTION"},
		{UUID: "opt-a", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "cg1", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
			"text":       "Sehr zufrieden",
			"isRequired": true,
		}},
		{UUID: "opt-b", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "cg1", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
			"text":       "Zufrieden",
			"isRequired": true,
		}},
	}
	q := tally.SubmissionQuestion{
		ID:   "q1",
		Type: "MULTIPLE_CHOICE",
		Fields: []tally.SubmissionField{
			{UUID: "opt-a", BlockGroupUUID: "cg1"},
		},
	}
	cfg := extractQuestionConfig(q, blocks)

	if !cfg.IsRequired {
		t.Errorf("IsRequired = false, want true")
	}
	if len(cfg.Options) != 2 {
		t.Fatalf("Options len = %d, want 2", len(cfg.Options))
	}
	if cfg.Options[0].Label != "Sehr zufrieden" || cfg.Options[0].UUID != "opt-a" {
		t.Errorf("first option = %+v, want {opt-a, Sehr zufrieden}", cfg.Options[0])
	}
}

func TestExtractQuestionConfig_TextAreaWithPlaceholder(t *testing.T) {
	blocks := []tally.TallyBlock{
		{UUID: "txt-1", Type: "TEXTAREA", GroupUUID: "cg2", GroupType: "TEXTAREA", Payload: map[string]any{
			"isRequired":  false,
			"placeholder": "Bitte beschreiben Sie ...",
		}},
	}
	q := tally.SubmissionQuestion{
		Type:   "TEXTAREA",
		Fields: []tally.SubmissionField{{UUID: "txt-1", BlockGroupUUID: "cg2"}},
	}
	cfg := extractQuestionConfig(q, blocks)
	if cfg.IsRequired {
		t.Errorf("IsRequired = true, want false")
	}
	if cfg.Placeholder != "Bitte beschreiben Sie ..." {
		t.Errorf("Placeholder = %q, want %q", cfg.Placeholder, "Bitte beschreiben Sie ...")
	}
}

func TestExtractQuestionConfig_LinearScale(t *testing.T) {
	blocks := []tally.TallyBlock{
		{UUID: "scale-1", Type: "LINEAR_SCALE", GroupUUID: "cg3", GroupType: "LINEAR_SCALE", Payload: map[string]any{
			"isRequired":   true,
			"start":        float64(0),
			"end":          float64(10),
			"step":         float64(1),
			"leftLabel":    "schlecht",
			"rightLabel":   "gut",
			"hasLeftLabel": true,
		}},
	}
	q := tally.SubmissionQuestion{
		Type:   "LINEAR_SCALE",
		Fields: []tally.SubmissionField{{UUID: "scale-1", BlockGroupUUID: "cg3"}},
	}
	cfg := extractQuestionConfig(q, blocks)
	if cfg.ScaleStart == nil || *cfg.ScaleStart != 0 {
		t.Errorf("ScaleStart = %v, want 0", cfg.ScaleStart)
	}
	if cfg.ScaleEnd == nil || *cfg.ScaleEnd != 10 {
		t.Errorf("ScaleEnd = %v, want 10", cfg.ScaleEnd)
	}
	if cfg.ScaleStep == nil || *cfg.ScaleStep != 1 {
		t.Errorf("ScaleStep = %v, want 1", cfg.ScaleStep)
	}
	if cfg.ScaleLeft != "schlecht" || cfg.ScaleRight != "gut" {
		t.Errorf("scale labels = (%q, %q), want (schlecht, gut)", cfg.ScaleLeft, cfg.ScaleRight)
	}
}

func TestExtractQuestionConfig_Rating(t *testing.T) {
	blocks := []tally.TallyBlock{
		{UUID: "r-1", Type: "RATING", GroupUUID: "cg4", GroupType: "RATING", Payload: map[string]any{
			"isRequired": true,
			"stars":      float64(5),
		}},
	}
	q := tally.SubmissionQuestion{
		Type:   "RATING",
		Fields: []tally.SubmissionField{{UUID: "r-1", BlockGroupUUID: "cg4"}},
	}
	cfg := extractQuestionConfig(q, blocks)
	if cfg.Stars == nil || *cfg.Stars != 5 {
		t.Errorf("Stars = %v, want 5", cfg.Stars)
	}
}

func TestExtractQuestionConfig_Matrix(t *testing.T) {
	blocks := []tally.TallyBlock{
		{UUID: "m-container", Type: "MATRIX", GroupUUID: "mg1", GroupType: "MATRIX", Payload: map[string]any{
			"isRequired": true,
		}},
		{UUID: "m-col-1", Type: "MATRIX_COLUMN", GroupUUID: "mg1", GroupType: "MATRIX", Payload: map[string]any{
			"safeHTMLSchema": []any{[]any{"Stimme zu"}},
		}},
		{UUID: "m-col-2", Type: "MATRIX_COLUMN", GroupUUID: "mg1", GroupType: "MATRIX", Payload: map[string]any{
			"safeHTMLSchema": []any{[]any{"Stimme nicht zu"}},
		}},
		{UUID: "m-row-1", Type: "MATRIX_ROW", GroupUUID: "mg1", GroupType: "MATRIX", Payload: map[string]any{
			"safeHTMLSchema": []any{[]any{"Teamkultur"}},
		}},
		{UUID: "m-row-2", Type: "MATRIX_ROW", GroupUUID: "mg1", GroupType: "MATRIX", Payload: map[string]any{
			"safeHTMLSchema": []any{[]any{"Prozesse"}},
		}},
	}
	// Matrix submissions have one Field per row; BlockGroupUUID is the row's
	// block UUID (Tally's terminology is misleading here).
	q := tally.SubmissionQuestion{
		Type: "MATRIX",
		Fields: []tally.SubmissionField{
			{UUID: "m-row-1", BlockGroupUUID: "m-row-1"},
			{UUID: "m-row-2", BlockGroupUUID: "m-row-2"},
		},
	}
	cfg := extractQuestionConfig(q, blocks)

	if !cfg.IsRequired {
		t.Errorf("IsRequired = false, want true")
	}
	if got := cfg.MatrixColumns; len(got) != 2 || got[0] != "Stimme zu" || got[1] != "Stimme nicht zu" {
		t.Errorf("MatrixColumns = %v, want [Stimme zu, Stimme nicht zu]", got)
	}
	if got := cfg.MatrixRows; len(got) != 2 || got[0].Label != "Teamkultur" || got[1].Label != "Prozesse" {
		t.Errorf("MatrixRows = %v, want Teamkultur then Prozesse", got)
	}
}

func TestExtractQuestionConfig_NoFieldsReturnsZero(t *testing.T) {
	blocks := []tally.TallyBlock{
		{UUID: "x", Type: "INPUT_TEXT", GroupUUID: "g", Payload: map[string]any{"isRequired": true}},
	}
	q := tally.SubmissionQuestion{Type: "INPUT_TEXT"} // no Fields
	cfg := extractQuestionConfig(q, blocks)
	if cfg.IsRequired {
		t.Errorf("expected zero config when q.Fields is empty")
	}
}

func TestExtractQuestionConfig_MissingMatrixRowReturnsZero(t *testing.T) {
	// Form blocks don't contain the row UUID referenced by q.Fields →
	// matrixGroup stays empty and we bail with zero config.
	blocks := []tally.TallyBlock{
		{UUID: "other", Type: "TITLE", GroupUUID: "qg", Payload: map[string]any{}},
	}
	q := tally.SubmissionQuestion{
		Type:   "MATRIX",
		Fields: []tally.SubmissionField{{UUID: "missing", BlockGroupUUID: "missing"}},
	}
	cfg := extractQuestionConfig(q, blocks)
	if cfg.IsRequired || len(cfg.MatrixRows) != 0 {
		t.Errorf("expected zero config when row block is missing, got %+v", cfg)
	}
}

func TestIntPayload(t *testing.T) {
	tests := []struct {
		name string
		p    map[string]any
		key  string
		want *int
	}{
		{"missing key", map[string]any{}, "x", nil},
		{"float64 value", map[string]any{"x": float64(7)}, "x", intPtr(7)},
		{"int value", map[string]any{"x": 9}, "x", intPtr(9)},
		{"non-numeric value", map[string]any{"x": "string"}, "x", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intPayload(tt.p, tt.key)
			switch {
			case got == nil && tt.want == nil:
				return
			case got == nil || tt.want == nil:
				t.Errorf("got=%v want=%v", got, tt.want)
			case *got != *tt.want:
				t.Errorf("got=%d want=%d", *got, *tt.want)
			}
		})
	}
}

func intPtr(v int) *int { return &v }
