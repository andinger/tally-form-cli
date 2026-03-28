package tally

import (
	"fmt"
	"strings"
	"testing"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/model"
)

func sequentialUUID() func() string {
	counter := 0
	return func() string {
		counter++
		return fmt.Sprintf("uuid-%04d", counter)
	}
}

func testCompiler() *Compiler {
	c := NewCompiler()
	c.NewUUID = sequentialUUID()
	return c
}

func testConfig() *config.Merged {
	return &config.Merged{
		Workspace: "test-ws",
		Logo:      "https://example.com/logo.svg",
	}
}

func TestCompileSingleChoice(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Your role?",
					Type:     model.SingleChoice,
					Required: true,
					Options: []model.Option{
						{Text: "Manager"},
						{Text: "Developer"},
						{Text: "Other", IsOther: true},
					},
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TITLE + 3 options = 5 blocks
	if len(req.Blocks) != 5 {
		t.Fatalf("Blocks = %d, want 5", len(req.Blocks))
	}

	// Check FORM_TITLE
	if req.Blocks[0].Type != "FORM_TITLE" {
		t.Errorf("Block 0 type = %q, want FORM_TITLE", req.Blocks[0].Type)
	}

	// Check TITLE
	if req.Blocks[1].Type != "TITLE" {
		t.Errorf("Block 1 type = %q, want TITLE", req.Blocks[1].Type)
	}
	if req.Blocks[1].GroupType != "QUESTION" {
		t.Errorf("Block 1 groupType = %q, want QUESTION", req.Blocks[1].GroupType)
	}

	// Check options
	for i := 2; i < 5; i++ {
		b := req.Blocks[i]
		if b.Type != "MULTIPLE_CHOICE_OPTION" {
			t.Errorf("Block %d type = %q, want MULTIPLE_CHOICE_OPTION", i, b.Type)
		}
		if b.GroupType != "MULTIPLE_CHOICE" {
			t.Errorf("Block %d groupType = %q, want MULTIPLE_CHOICE", i, b.GroupType)
		}
	}

	// TITLE and options must have different groupUUIDs (Tally editor requires this)
	titleGroup := req.Blocks[1].GroupUUID
	optionGroup := req.Blocks[2].GroupUUID
	if titleGroup == optionGroup {
		t.Errorf("TITLE and options must have different groupUUIDs, both are %q", titleGroup)
	}

	// Check other option
	lastOpt := req.Blocks[4]
	if lastOpt.Payload["isOtherOption"] != true {
		t.Error("Last option should be isOtherOption")
	}
	if lastOpt.Payload["hasOtherOption"] != true {
		t.Error("Last option should have hasOtherOption")
	}
	// All options should have hasOtherOption
	if req.Blocks[2].Payload["hasOtherOption"] != true {
		t.Error("First option should have hasOtherOption")
	}
}

func TestCompileMultiChoice(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Tools?",
					Type:     model.MultiChoice,
					Required: true,
					Options: []model.Option{
						{Text: "Excel"},
						{Text: "Word"},
					},
					Properties: map[string]any{"max": 2},
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TITLE + 2 checkboxes = 4
	if len(req.Blocks) != 4 {
		t.Fatalf("Blocks = %d, want 4", len(req.Blocks))
	}

	cb := req.Blocks[2]
	if cb.Type != "CHECKBOX" {
		t.Errorf("Type = %q, want CHECKBOX", cb.Type)
	}
	if cb.GroupType != "CHECKBOXES" {
		t.Errorf("GroupType = %q, want CHECKBOXES", cb.GroupType)
	}
	if cb.Payload["hasMaxChoices"] != true {
		t.Error("Expected hasMaxChoices = true")
	}
}

func TestCompileLongText(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:          "F1",
					Text:        "Describe",
					Type:        model.LongText,
					Required:    true,
					Placeholder: "Type here",
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TITLE + TEXTAREA = 3
	if len(req.Blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(req.Blocks))
	}

	ta := req.Blocks[2]
	if ta.Type != "TEXTAREA" {
		t.Errorf("Type = %q, want TEXTAREA", ta.Type)
	}
	if ta.Payload["placeholder"] != "Type here" {
		t.Errorf("placeholder = %v", ta.Payload["placeholder"])
	}
}

func TestCompileMatrix(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:         "F1",
					Text:       "Rate",
					Type:       model.Matrix,
					Required:   true,
					MatrixCols: []string{"Low", "High"},
					MatrixRows: []string{"Reports", "Docs"},
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TITLE + MATRIX + 2 cols + 2 rows = 7
	if len(req.Blocks) != 7 {
		t.Fatalf("Blocks = %d, want 7", len(req.Blocks))
	}

	// MATRIX block
	if req.Blocks[2].Type != "MATRIX" {
		t.Errorf("Block 2 type = %q, want MATRIX", req.Blocks[2].Type)
	}
	// Columns
	if req.Blocks[3].Type != "MATRIX_COLUMN" {
		t.Errorf("Block 3 type = %q, want MATRIX_COLUMN", req.Blocks[3].Type)
	}
	// Rows
	if req.Blocks[5].Type != "MATRIX_ROW" {
		t.Errorf("Block 5 type = %q, want MATRIX_ROW", req.Blocks[5].Type)
	}
}

func TestCompileConditional(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:   "F1",
					Text: "Role?",
					Type: model.SingleChoice,
					Options: []model.Option{
						{Text: "Yes"},
						{Text: "No"},
					},
				},
				&model.Conditional{
					Targets:  []string{"F2"},
					Operator: "AND",
					Conditions: []model.Condition{
						{Field: "F1", Comparison: "is", Values: []string{"Yes"}},
					},
				},
				&model.Question{
					ID:     "F2",
					Text:   "Details?",
					Type:   model.LongText,
					Hidden: true,
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// Find CONDITIONAL_LOGIC block
	var condBlock *TallyBlock
	for i := range req.Blocks {
		if req.Blocks[i].Type == "CONDITIONAL_LOGIC" {
			condBlock = &req.Blocks[i]
			break
		}
	}
	if condBlock == nil {
		t.Fatal("No CONDITIONAL_LOGIC block found")
	}

	if condBlock.Payload["logicalOperator"] != "AND" {
		t.Errorf("logicalOperator = %v, want AND", condBlock.Payload["logicalOperator"])
	}

	actions := condBlock.Payload["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("Actions = %d, want 1", len(actions))
	}

	action := actions[0].(map[string]any)
	if action["type"] != "SHOW_BLOCKS" {
		t.Errorf("Action type = %v, want SHOW_BLOCKS", action["type"])
	}
}

func TestCompileMultiPage(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{
			{Blocks: []model.Block{
				&model.Question{ID: "F1", Text: "Q1?", Type: model.ShortText},
			}},
			{ButtonLabel: "Next page", Blocks: []model.Block{
				&model.Question{ID: "F2", Text: "Q2?", Type: model.ShortText},
			}},
		},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// Find PAGE_BREAK
	var pb *TallyBlock
	for i := range req.Blocks {
		if req.Blocks[i].Type == "PAGE_BREAK" {
			pb = &req.Blocks[i]
			break
		}
	}
	if pb == nil {
		t.Fatal("No PAGE_BREAK found")
	}
	buttonLabel := pb.Payload["button"].(map[string]any)["label"]
	if buttonLabel != "Next page" {
		t.Errorf("Button label = %v, want 'Next page'", buttonLabel)
	}
}

func TestCompileHiddenField(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:     "F1",
					Text:   "Hidden Q",
					Type:   model.SingleChoice,
					Hidden: true,
					Options: []model.Option{
						{Text: "A"},
					},
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// Title should have isHidden
	title := req.Blocks[1]
	if title.Payload["isHidden"] != true {
		t.Error("TITLE should have isHidden")
	}

	// Option should have isHidden
	opt := req.Blocks[2]
	if opt.Payload["isHidden"] != true {
		t.Error("Option should have isHidden")
	}
}

func TestCompileHeadingAndText(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.TextBlock{HTML: "Welcome"},
				&model.HeadingBlock{Text: "Section", Level: 2},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TEXT + HEADING_2 = 3
	if len(req.Blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(req.Blocks))
	}
	if req.Blocks[1].Type != "TEXT" {
		t.Errorf("Block 1 = %q, want TEXT", req.Blocks[1].Type)
	}
	if req.Blocks[2].Type != "HEADING_2" {
		t.Errorf("Block 2 = %q, want HEADING_2", req.Blocks[2].Type)
	}
}

func TestCompileFormTitle(t *testing.T) {
	form := &model.Form{
		Name:  "My Form",
		Pages: []model.Page{{}},
	}

	cfg := testConfig()
	cfg.Logo = "https://example.com/logo.png"

	c := testCompiler()
	req, err := c.Compile(form, cfg)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	ft := req.Blocks[0]
	if ft.Type != "FORM_TITLE" {
		t.Errorf("Type = %q", ft.Type)
	}
	if ft.Payload["logo"] != "https://example.com/logo.png" {
		t.Errorf("logo = %v", ft.Payload["logo"])
	}
	if ft.Payload["title"] != "My Form" {
		t.Errorf("title = %v", ft.Payload["title"])
	}
}

func TestCompileDropdownNoMultiChoiceFields(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Pick one",
					Type:     model.Dropdown,
					Required: true,
					Options: []model.Option{
						{Text: "Alpha"},
						{Text: "Beta"},
					},
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// Check dropdown option blocks don't have MULTIPLE_CHOICE-specific fields
	for _, b := range req.Blocks {
		if b.Type != "DROPDOWN_OPTION" {
			continue
		}
		if _, ok := b.Payload["allowMultiple"]; ok {
			t.Error("DROPDOWN_OPTION should not have allowMultiple")
		}
		if _, ok := b.Payload["hasBadge"]; ok {
			t.Error("DROPDOWN_OPTION should not have hasBadge")
		}
		if _, ok := b.Payload["badgeType"]; ok {
			t.Error("DROPDOWN_OPTION should not have badgeType")
		}
		if _, ok := b.Payload["colorCodeOptions"]; ok {
			t.Error("DROPDOWN_OPTION should not have colorCodeOptions")
		}
	}
}

func TestCompileConditionalRejectsIncompatibleOperator(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:   "F1",
					Text: "Tools?",
					Type: model.MultiChoice,
					Options: []model.Option{
						{Text: "A"},
						{Text: "B"},
					},
				},
				&model.Conditional{
					Targets:  []string{"F2"},
					Operator: "AND",
					Conditions: []model.Condition{
						{Field: "F1", Comparison: "is_not_any_of", Values: []string{"A"}},
					},
				},
				&model.Question{
					ID:     "F2",
					Text:   "Details?",
					Type:   model.LongText,
					Hidden: true,
				},
			},
		}},
	}

	c := testCompiler()
	_, err := c.Compile(form, testConfig())
	if err == nil {
		t.Fatal("Expected error for is_not_any_of on multi-choice, got nil")
	}
	if !strings.Contains(err.Error(), "is not supported for multi-choice") {
		t.Errorf("Error = %q, expected mention of multi-choice incompatibility", err.Error())
	}
}

func TestSafeHTMLSchemaFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantText string // first segment text
	}{
		{"plain", "Hello world", 1, "Hello world"},
		{"bold", "Hello <b>world</b>!", 3, "Hello "},
		{"italic", "<i>emphasis</i> here", 2, "emphasis"},
		{"mixed", "A <b>bold</b> and <i>italic</i> text", 5, "A "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := SafeHTMLSchemaFromHTML(tt.input)
			if len(schema) != tt.wantLen {
				t.Errorf("segments = %d, want %d: %v", len(schema), tt.wantLen, schema)
			}
			if len(schema) > 0 {
				seg := schema[0].([]any)
				if seg[0].(string) != tt.wantText {
					t.Errorf("first text = %q, want %q", seg[0], tt.wantText)
				}
			}
		})
	}

	// Check bold segment has correct styles
	schema := SafeHTMLSchemaFromHTML("Go <b>bold</b>!")
	boldSeg := schema[1].([]any)
	if boldSeg[0] != "bold" {
		t.Errorf("bold text = %q", boldSeg[0])
	}
	styles := boldSeg[1].([]any)
	if len(styles) != 2 {
		t.Fatalf("styles len = %d, want 2", len(styles))
	}
}
