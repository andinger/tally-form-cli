package tally

import (
	"testing"

	"github.com/andinger/tally-form-cli/internal/model"
)

func TestDecompileMinimal(t *testing.T) {
	tf := &TallyForm{
		ID:       "abc123",
		Name:     "My Form",
		Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "ft", Type: "FORM_TITLE", GroupUUID: "g1", GroupType: "TEXT", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"My Form"}},
			}},
			{UUID: "t1", Type: "TITLE", GroupUUID: "g2", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Question?"}},
			}},
			{UUID: "ta", Type: "TEXTAREA", GroupUUID: "g3", GroupType: "TEXTAREA", Payload: map[string]any{
				"isRequired": true,
			}},
		},
	}

	form, err := Decompile(tf)
	if err != nil {
		t.Fatalf("Decompile error: %v", err)
	}
	if form.Name != "My Form" {
		t.Errorf("Name = %q", form.Name)
	}
	if form.FormID != "abc123" {
		t.Errorf("FormID = %q", form.FormID)
	}
	if len(form.Pages) != 1 {
		t.Fatalf("Pages = %d, want 1", len(form.Pages))
	}
	if len(form.Pages[0].Blocks) != 1 {
		t.Fatalf("Blocks = %d, want 1", len(form.Pages[0].Blocks))
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.ID != "F1" {
		t.Errorf("ID = %q", q.ID)
	}
	if q.Text != "Question?" {
		t.Errorf("Text = %q", q.Text)
	}
	if q.Type != model.LongText {
		t.Errorf("Type = %q", q.Type)
	}
}

func TestDecompilePassword(t *testing.T) {
	tf := &TallyForm{
		ID:       "pw",
		Name:     "PW Form",
		Settings: map[string]any{"password": "secret"},
		Blocks:   []TallyBlock{},
	}
	form, _ := Decompile(tf)
	if form.Password != "secret" {
		t.Errorf("Password = %q", form.Password)
	}
}

func TestDecompileMultiPage(t *testing.T) {
	tf := &TallyForm{
		ID:       "mp",
		Name:     "Multi",
		Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "ft", Type: "FORM_TITLE", GroupUUID: "g0", GroupType: "TEXT", Payload: map[string]any{}},
			{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Q1?"}},
			}},
			{UUID: "in1", Type: "INPUT_TEXT", GroupUUID: "g2", GroupType: "INPUT_TEXT", Payload: map[string]any{
				"isRequired": true,
			}},
			{UUID: "pb", Type: "PAGE_BREAK", GroupUUID: "g3", GroupType: "PAGE_BREAK", Payload: map[string]any{
				"button": map[string]any{"label": "Next"},
			}},
			{UUID: "t2", Type: "TITLE", GroupUUID: "g4", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Q2?"}},
			}},
			{UUID: "ta2", Type: "TEXTAREA", GroupUUID: "g5", GroupType: "TEXTAREA", Payload: map[string]any{
				"isRequired": false,
			}},
		},
	}

	form, _ := Decompile(tf)
	if len(form.Pages) != 2 {
		t.Fatalf("Pages = %d, want 2", len(form.Pages))
	}
	if form.Pages[1].ButtonLabel != "Next" {
		t.Errorf("ButtonLabel = %q, want Next", form.Pages[1].ButtonLabel)
	}
	// Default button "Weiter" should not be stored
	tf.Blocks[3].Payload["button"] = map[string]any{"label": "Weiter"}
	form2, _ := Decompile(tf)
	if form2.Pages[1].ButtonLabel != "" {
		t.Errorf("Default Weiter should not be stored, got %q", form2.Pages[1].ButtonLabel)
	}
}

func TestDecompileAllQuestionTypes(t *testing.T) {
	tests := []struct {
		blockType string
		groupType string
		wantType  model.QuestionType
	}{
		{"MULTIPLE_CHOICE_OPTION", "MULTIPLE_CHOICE", model.SingleChoice},
		{"CHECKBOX", "CHECKBOXES", model.MultiChoice},
		{"DROPDOWN_OPTION", "DROPDOWN", model.Dropdown},
		{"TEXTAREA", "TEXTAREA", model.LongText},
		{"INPUT_TEXT", "INPUT_TEXT", model.ShortText},
		{"INPUT_NUMBER", "INPUT_NUMBER", model.Number},
		{"INPUT_EMAIL", "INPUT_EMAIL", model.Email},
		{"INPUT_PHONE_NUMBER", "INPUT_PHONE_NUMBER", model.Phone},
		{"INPUT_LINK", "INPUT_LINK", model.URL},
		{"INPUT_DATE", "INPUT_DATE", model.Date},
		{"INPUT_TIME", "INPUT_TIME", model.Time},
		{"RATING", "RATING", model.Rating},
		{"LINEAR_SCALE", "LINEAR_SCALE", model.Scale},
		{"FILE_UPLOAD", "FILE_UPLOAD", model.FileUpload},
		{"SIGNATURE", "SIGNATURE", model.Signature},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			tf := &TallyForm{
				Name: "T", Settings: map[string]any{},
				Blocks: []TallyBlock{
					{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
						"safeHTMLSchema": []any{[]any{"Q?"}},
					}},
					{UUID: "c1", Type: tt.blockType, GroupUUID: "g2", GroupType: tt.groupType, Payload: map[string]any{
						"text": "Opt", "isRequired": true,
					}},
				},
			}
			form, err := Decompile(tf)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			q := form.Pages[0].Blocks[0].(*model.Question)
			if q.Type != tt.wantType {
				t.Errorf("type = %q, want %q", q.Type, tt.wantType)
			}
		})
	}
}

func TestDecompileMatrix(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Rate?"}},
			}},
			{UUID: "mc1", Type: "MATRIX_COLUMN", GroupUUID: "g2", GroupType: "MATRIX", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Low"}},
			}},
			{UUID: "mc2", Type: "MATRIX_COLUMN", GroupUUID: "g2", GroupType: "MATRIX", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"High"}},
			}},
			{UUID: "mr1", Type: "MATRIX_ROW", GroupUUID: "g2", GroupType: "MATRIX", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Item A"}},
			}},
			{UUID: "mr2", Type: "MATRIX_ROW", GroupUUID: "g2", GroupType: "MATRIX", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Item B"}},
			}},
		},
	}

	form, _ := Decompile(tf)
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Type != model.Matrix {
		t.Errorf("Type = %q, want matrix", q.Type)
	}
	if len(q.MatrixCols) != 2 {
		t.Errorf("MatrixCols = %d, want 2", len(q.MatrixCols))
	}
	if len(q.MatrixRows) != 2 {
		t.Errorf("MatrixRows = %d, want 2", len(q.MatrixRows))
	}
	if q.MatrixCols[0] != "Low" || q.MatrixCols[1] != "High" {
		t.Errorf("Cols = %v", q.MatrixCols)
	}
}

func TestDecompileMatrixWithContainer(t *testing.T) {
	// Legacy forms may have a MATRIX container block
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Rate?"}},
			}},
			{UUID: "mx", Type: "MATRIX", GroupUUID: "g2", GroupType: "QUESTION", Payload: map[string]any{
				"isRequired": true,
			}},
			{UUID: "mc1", Type: "MATRIX_COLUMN", GroupUUID: "g2", GroupType: "MATRIX", Payload: map[string]any{
				"text": "Col1",
			}},
			{UUID: "mr1", Type: "MATRIX_ROW", GroupUUID: "g2", GroupType: "MATRIX", Payload: map[string]any{
				"text": "Row1",
			}},
		},
	}

	form, _ := Decompile(tf)
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Type != model.Matrix {
		t.Errorf("Type = %q, want matrix", q.Type)
	}
	if len(q.MatrixCols) != 1 || q.MatrixCols[0] != "Col1" {
		t.Errorf("Cols = %v", q.MatrixCols)
	}
}

func TestDecompileHiddenFieldAndHint(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Q?"}},
				"isHidden":       true,
			}},
			// Hint TEXT block followed by content
			{UUID: "hint", Type: "TEXT", GroupUUID: "gh", GroupType: "TEXT", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"<i>Helper text</i>", []any{[]any{"tag", "span"}, []any{"font-style", "italic"}}}},
			}},
			{UUID: "ta", Type: "TEXTAREA", GroupUUID: "g2", GroupType: "TEXTAREA", Payload: map[string]any{
				"isRequired": false, "isHidden": true,
			}},
		},
	}

	form, _ := Decompile(tf)
	q := form.Pages[0].Blocks[0].(*model.Question)
	if !q.Hidden {
		t.Error("Expected hidden=true")
	}
	if q.Hint != "<i>Helper text</i>" {
		t.Errorf("Hint = %q", q.Hint)
	}
}

func TestDecompileChoiceOptions(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Pick?"}},
			}},
			{UUID: "o1", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "g2", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
				"text": "Yes", "isRequired": true, "isOtherOption": false, "isHidden": false,
				"hasMaxChoices": true, "maxChoices": float64(3),
			}},
			{UUID: "o2", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "g2", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
				"text": "Other", "isRequired": true, "isOtherOption": true,
			}},
		},
	}

	form, _ := Decompile(tf)
	q := form.Pages[0].Blocks[0].(*model.Question)
	if len(q.Options) != 2 {
		t.Fatalf("Options = %d, want 2", len(q.Options))
	}
	if q.Options[0].Text != "Yes" || q.Options[0].IsOther {
		t.Errorf("Option 0 = %+v", q.Options[0])
	}
	if !q.Options[1].IsOther {
		t.Error("Option 1 should be isOther")
	}
	if q.Properties["max"] != 3 {
		t.Errorf("max = %v", q.Properties["max"])
	}
}

func TestDecompileInputWithPlaceholder(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "g1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Name?"}},
			}},
			{UUID: "in", Type: "INPUT_TEXT", GroupUUID: "g2", GroupType: "INPUT_TEXT", Payload: map[string]any{
				"isRequired": false, "placeholder": "Enter name",
			}},
		},
	}

	form, _ := Decompile(tf)
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Placeholder != "Enter name" {
		t.Errorf("Placeholder = %q", q.Placeholder)
	}
	if q.Required {
		t.Error("Should not be required")
	}
}

func TestDecompileHeadingsAndText(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "h1", Type: "HEADING_1", GroupUUID: "g1", GroupType: "HEADING_1", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Title"}},
			}},
			{UUID: "h2", Type: "HEADING_2", GroupUUID: "g2", GroupType: "HEADING_2", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Subtitle"}},
			}},
			{UUID: "tx", Type: "TEXT", GroupUUID: "g3", GroupType: "TEXT", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Paragraph text"}},
			}},
		},
	}

	form, _ := Decompile(tf)
	blocks := form.Pages[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(blocks))
	}

	h1 := blocks[0].(*model.HeadingBlock)
	if h1.Level != 1 || h1.Text != "Title" {
		t.Errorf("H1 = %+v", h1)
	}
	h2 := blocks[1].(*model.HeadingBlock)
	if h2.Level != 2 || h2.Text != "Subtitle" {
		t.Errorf("H2 = %+v", h2)
	}
	txt := blocks[2].(*model.TextBlock)
	if txt.HTML != "Paragraph text" {
		t.Errorf("Text = %q", txt.HTML)
	}
}

func TestDecompileConditional(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "gq1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Source?"}},
			}},
			{UUID: "o1", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "gc1", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
				"text": "Yes", "isRequired": true,
			}},
			{UUID: "o2", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "gc1", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
				"text": "No", "isRequired": true,
			}},
			{UUID: "cond", Type: "CONDITIONAL_LOGIC", GroupUUID: "gcl", GroupType: "CONDITIONAL_LOGIC", Payload: map[string]any{
				"logicalOperator": "AND",
				"conditionals": []any{
					map[string]any{
						"uuid": "cu1",
						"type": "SINGLE",
						"payload": map[string]any{
							"field": map[string]any{
								"uuid":           "gc1",
								"blockGroupUuid": "gc1",
								"questionType":   "MULTIPLE_CHOICE",
								"title":          "Source?",
							},
							"comparison": "IS",
							"value":      "o1",
						},
					},
				},
				"actions": []any{
					map[string]any{
						"uuid": "au1",
						"type": "SHOW_BLOCKS",
						"payload": map[string]any{
							"showBlocks": []any{"t2", "ta2"},
						},
					},
				},
			}},
			{UUID: "t2", Type: "TITLE", GroupUUID: "gq2", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Details?"}},
				"isHidden":       true,
			}},
			{UUID: "ta2", Type: "TEXTAREA", GroupUUID: "gc2", GroupType: "TEXTAREA", Payload: map[string]any{
				"isRequired": false,
			}},
		},
	}

	form, _ := Decompile(tf)
	// Should have 3 blocks: Q1, conditional, Q2
	blocks := form.Pages[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(blocks))
	}

	cond := blocks[1].(*model.Conditional)
	if cond.Operator != "AND" {
		t.Errorf("Operator = %q", cond.Operator)
	}
	if len(cond.Conditions) != 1 {
		t.Fatalf("Conditions = %d", len(cond.Conditions))
	}
	c := cond.Conditions[0]
	if c.Field != "F1" {
		t.Errorf("Field = %q, want F1", c.Field)
	}
	if c.Comparison != "is" {
		t.Errorf("Comparison = %q", c.Comparison)
	}
	if len(c.Values) != 1 || c.Values[0] != "Yes" {
		t.Errorf("Values = %v, want [Yes]", c.Values)
	}
	if len(cond.Targets) != 1 || cond.Targets[0] != "F2" {
		t.Errorf("Targets = %v, want [F2]", cond.Targets)
	}
}

func TestDecompileConditionalOR(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "gq1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Q?"}},
			}},
			{UUID: "in", Type: "INPUT_TEXT", GroupUUID: "gc1", GroupType: "INPUT_TEXT", Payload: map[string]any{}},
			{UUID: "cond", Type: "CONDITIONAL_LOGIC", GroupUUID: "gcl", GroupType: "CONDITIONAL_LOGIC", Payload: map[string]any{
				"logicalOperator": "OR",
				"conditionals": []any{
					map[string]any{
						"uuid": "cu1", "type": "SINGLE",
						"payload": map[string]any{
							"field":      map[string]any{"blockGroupUuid": "gc1"},
							"comparison": "IS_NOT_EMPTY",
							"value":      "",
						},
					},
				},
				"actions": []any{},
			}},
		},
	}

	form, _ := Decompile(tf)
	cond := form.Pages[0].Blocks[1].(*model.Conditional)
	if cond.Operator != "OR" {
		t.Errorf("Operator = %q, want OR", cond.Operator)
	}
	if cond.Conditions[0].Comparison != "is_not_empty" {
		t.Errorf("Comparison = %q", cond.Conditions[0].Comparison)
	}
	// No targets resolved → defaults to F?
	if cond.Targets[0] != "F?" {
		t.Errorf("Targets = %v, want [F?]", cond.Targets)
	}
}

func TestDecompileConditionalMultipleValues(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "t1", Type: "TITLE", GroupUUID: "gq1", GroupType: "QUESTION", Payload: map[string]any{
				"safeHTMLSchema": []any{[]any{"Q?"}},
			}},
			{UUID: "o1", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "gc1", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
				"text": "A",
			}},
			{UUID: "o2", Type: "MULTIPLE_CHOICE_OPTION", GroupUUID: "gc1", GroupType: "MULTIPLE_CHOICE", Payload: map[string]any{
				"text": "B",
			}},
			{UUID: "cond", Type: "CONDITIONAL_LOGIC", GroupUUID: "gcl", GroupType: "CONDITIONAL_LOGIC", Payload: map[string]any{
				"logicalOperator": "AND",
				"conditionals": []any{
					map[string]any{
						"uuid": "cu1", "type": "SINGLE",
						"payload": map[string]any{
							"field":      map[string]any{"blockGroupUuid": "gc1"},
							"comparison": "IS_ANY_OF",
							"value":      []any{"o1", "o2"},
						},
					},
				},
				"actions": []any{},
			}},
		},
	}

	form, _ := Decompile(tf)
	cond := form.Pages[0].Blocks[1].(*model.Conditional)
	vals := cond.Conditions[0].Values
	if len(vals) != 2 || vals[0] != "A" || vals[1] != "B" {
		t.Errorf("Values = %v, want [A B]", vals)
	}
}

func TestDecompileUnknownBlockType(t *testing.T) {
	tf := &TallyForm{
		Name: "T", Settings: map[string]any{},
		Blocks: []TallyBlock{
			{UUID: "x", Type: "UNKNOWN_TYPE", GroupUUID: "g1", GroupType: "UNKNOWN", Payload: map[string]any{}},
		},
	}
	form, err := Decompile(tf)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Unknown blocks are skipped
	if len(form.Pages[0].Blocks) != 0 {
		t.Errorf("Blocks = %d, want 0", len(form.Pages[0].Blocks))
	}
}

func TestExtractFromSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema []any
		want   string
	}{
		{"plain", []any{[]any{"Hello"}}, "Hello"},
		{"bold", []any{[]any{"B", []any{[]any{"tag", "span"}, []any{"font-weight", "bold"}}}}, "<b>B</b>"},
		{"italic", []any{[]any{"I", []any{[]any{"tag", "span"}, []any{"font-style", "italic"}}}}, "<i>I</i>"},
		{"href", []any{[]any{"Click", []any{[]any{"href", "https://example.com"}}}}, `<a href="https://example.com">Click</a>`},
		{"mixed", []any{[]any{"A "}, []any{"B", []any{[]any{"tag", "span"}, []any{"font-weight", "bold"}}}, []any{" C"}}, "A <b>B</b> C"},
		{"empty", []any{}, ""},
		{"invalid_item", []any{"not_an_array"}, ""},
		{"invalid_text", []any{[]any{123}}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFromSchema(tt.schema)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractText(t *testing.T) {
	// With safeHTMLSchema
	b := TallyBlock{Payload: map[string]any{
		"safeHTMLSchema": []any{[]any{"Schema text"}},
	}}
	if got := extractText(b); got != "Schema text" {
		t.Errorf("extractText(schema) = %q", got)
	}

	// Fallback to title
	b2 := TallyBlock{Payload: map[string]any{"title": "Title text"}}
	if got := extractText(b2); got != "Title text" {
		t.Errorf("extractText(title) = %q", got)
	}

	// Empty
	b3 := TallyBlock{Payload: map[string]any{}}
	if got := extractText(b3); got != "" {
		t.Errorf("extractText(empty) = %q", got)
	}
}

func TestGetPayloadText(t *testing.T) {
	// Prefers "text" field
	b := TallyBlock{Payload: map[string]any{
		"text":           "Direct text",
		"safeHTMLSchema": []any{[]any{"Schema text"}},
	}}
	if got := getPayloadText(b); got != "Direct text" {
		t.Errorf("got %q, want Direct text", got)
	}

	// Falls back to extractText
	b2 := TallyBlock{Payload: map[string]any{
		"safeHTMLSchema": []any{[]any{"Fallback"}},
	}}
	if got := getPayloadText(b2); got != "Fallback" {
		t.Errorf("got %q, want Fallback", got)
	}
}

func TestHasStyleProp(t *testing.T) {
	styles := []any{[]any{"tag", "span"}, []any{"font-weight", "bold"}}
	if !hasStyleProp(styles, "font-weight", "bold") {
		t.Error("should find font-weight bold")
	}
	if hasStyleProp(styles, "font-weight", "italic") {
		t.Error("should not find font-weight italic")
	}
	if hasStyleProp(styles, "nonexistent", "x") {
		t.Error("should not find nonexistent")
	}
	// Invalid items
	if hasStyleProp([]any{"not_array"}, "a", "b") {
		t.Error("should handle non-array items")
	}
	if hasStyleProp([]any{[]any{"only_one"}}, "a", "b") {
		t.Error("should handle short arrays")
	}
}

func TestGetStyleValue(t *testing.T) {
	styles := []any{[]any{"href", "https://example.com"}, []any{"tag", "span"}}
	if got := getStyleValue(styles, "href"); got != "https://example.com" {
		t.Errorf("got %q", got)
	}
	if got := getStyleValue(styles, "nonexistent"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	// Invalid items
	if got := getStyleValue([]any{"not_array"}, "x"); got != "" {
		t.Error("should handle non-array")
	}
}

func TestIsQuestionContent(t *testing.T) {
	contentTypes := []string{
		"MULTIPLE_CHOICE_OPTION", "CHECKBOX", "DROPDOWN_OPTION",
		"TEXTAREA", "INPUT_TEXT", "INPUT_NUMBER", "INPUT_EMAIL",
		"INPUT_PHONE_NUMBER", "INPUT_LINK", "INPUT_DATE", "INPUT_TIME",
		"RATING", "LINEAR_SCALE", "FILE_UPLOAD", "SIGNATURE",
		"MATRIX", "MATRIX_COLUMN", "MATRIX_ROW",
	}
	for _, ct := range contentTypes {
		if !isQuestionContent(ct) {
			t.Errorf("isQuestionContent(%q) = false, want true", ct)
		}
	}

	nonContent := []string{"FORM_TITLE", "PAGE_BREAK", "HEADING_1", "HEADING_2", "TEXT", "CONDITIONAL_LOGIC", "UNKNOWN"}
	for _, ct := range nonContent {
		if isQuestionContent(ct) {
			t.Errorf("isQuestionContent(%q) = true, want false", ct)
		}
	}
}
