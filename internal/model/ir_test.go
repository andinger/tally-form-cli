package model

import "testing"

func TestBlockInterface(t *testing.T) {
	tests := []struct {
		name     string
		block    Block
		wantType string
	}{
		{"HeadingBlock", &HeadingBlock{Text: "Title", Level: 2}, "heading"},
		{"TextBlock", &TextBlock{HTML: "<p>Hello</p>"}, "text"},
		{"Question", &Question{ID: "F1", Text: "Q?", Type: SingleChoice}, "question"},
		{"Conditional", &Conditional{Targets: []string{"F3"}, Operator: "AND"}, "conditional"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.BlockType(); got != tt.wantType {
				t.Errorf("BlockType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

func TestQuestionTypes(t *testing.T) {
	types := []QuestionType{
		SingleChoice, MultiChoice, Dropdown, LongText, ShortText,
		Number, Email, Phone, URL, Date, Time, Rating, Scale,
		Matrix, FileUpload, Signature,
	}
	seen := make(map[QuestionType]bool)
	for _, qt := range types {
		if seen[qt] {
			t.Errorf("duplicate QuestionType: %s", qt)
		}
		seen[qt] = true
		if qt == "" {
			t.Error("empty QuestionType")
		}
	}
}

func TestFormConstruction(t *testing.T) {
	form := &Form{
		Name:      "Test Form",
		Workspace: "ws123",
		Password:  "secret",
		FormID:    "abc123",
		Pages: []Page{
			{
				ButtonLabel: "Next",
				Blocks: []Block{
					&HeadingBlock{Text: "Section 1", Level: 2},
					&TextBlock{HTML: "Intro text"},
					&Question{
						ID:       "F1",
						Text:     "Your role?",
						Type:     SingleChoice,
						Required: true,
						Options: []Option{
							{Text: "Manager", IsOther: false},
							{Text: "Other", IsOther: true},
						},
					},
				},
			},
			{
				Blocks: []Block{
					&Question{
						ID:         "F2",
						Text:       "Rate documents",
						Type:       Matrix,
						MatrixCols: []string{"Low", "Medium", "High"},
						MatrixRows: []string{"Reports", "Protocols"},
					},
				},
			},
		},
	}

	if form.Name != "Test Form" {
		t.Errorf("Name = %q, want %q", form.Name, "Test Form")
	}
	if len(form.Pages) != 2 {
		t.Fatalf("Pages count = %d, want 2", len(form.Pages))
	}
	if len(form.Pages[0].Blocks) != 3 {
		t.Fatalf("Page 0 blocks = %d, want 3", len(form.Pages[0].Blocks))
	}

	q := form.Pages[0].Blocks[2].(*Question)
	if q.ID != "F1" {
		t.Errorf("Question ID = %q, want %q", q.ID, "F1")
	}
	if len(q.Options) != 2 {
		t.Fatalf("Options count = %d, want 2", len(q.Options))
	}
	if !q.Options[1].IsOther {
		t.Error("Expected second option to be IsOther")
	}

	mq := form.Pages[1].Blocks[0].(*Question)
	if len(mq.MatrixCols) != 3 {
		t.Errorf("MatrixCols = %d, want 3", len(mq.MatrixCols))
	}
}

func TestConditionalConstruction(t *testing.T) {
	c := &Conditional{
		Targets:  []string{"F3", "F4"},
		Operator: "AND",
		Conditions: []Condition{
			{
				Field:      "F2",
				Comparison: "is_not_any_of",
				Values:     []string{"Option A", "Option B"},
			},
			{
				Field:      "F2",
				Comparison: "is_not_empty",
				Values:     nil,
			},
		},
	}

	if len(c.Targets) != 2 {
		t.Errorf("Targets count = %d, want 2", len(c.Targets))
	}
	if len(c.Conditions) != 2 {
		t.Errorf("Conditions count = %d, want 2", len(c.Conditions))
	}
	if c.Conditions[0].Comparison != "is_not_any_of" {
		t.Errorf("Comparison = %q, want %q", c.Conditions[0].Comparison, "is_not_any_of")
	}
}
