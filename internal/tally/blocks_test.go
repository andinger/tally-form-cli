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
	if cb.Payload["maxChoices"] != 2 {
		t.Errorf("maxChoices = %v, want 2", cb.Payload["maxChoices"])
	}

	// TITLE and checkboxes must have different groupUUIDs
	if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
		t.Error("TITLE and CHECKBOX must have different groupUUIDs")
	}
	// All checkboxes share the same groupUUID
	if req.Blocks[2].GroupUUID != req.Blocks[3].GroupUUID {
		t.Error("All CHECKBOXes must share the same groupUUID")
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

	// TITLE and TEXTAREA must have different groupUUIDs
	if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
		t.Error("TITLE and TEXTAREA must have different groupUUIDs")
	}
}

func TestCompileShortText(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Name?",
					Type:     model.ShortText,
					Required: false,
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TITLE + INPUT_TEXT = 3
	if len(req.Blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(req.Blocks))
	}

	inp := req.Blocks[2]
	if inp.Type != "INPUT_TEXT" {
		t.Errorf("Type = %q, want INPUT_TEXT", inp.Type)
	}
	if inp.Payload["isRequired"] != false {
		t.Error("isRequired should be false")
	}

	// TITLE and INPUT_TEXT must have different groupUUIDs
	if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
		t.Error("TITLE and INPUT_TEXT must have different groupUUIDs")
	}
}

func TestCompileQuestionWithHint(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Email?",
					Type:     model.Email,
					Required: true,
					Hint:     "Your work email",
				},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// FORM_TITLE + TITLE + TEXT(hint) + INPUT_EMAIL = 4
	if len(req.Blocks) != 4 {
		t.Fatalf("Blocks = %d, want 4", len(req.Blocks))
	}

	// Hint is a TEXT block with italic safeHTMLSchema
	hint := req.Blocks[2]
	if hint.Type != "TEXT" {
		t.Errorf("Hint type = %q, want TEXT", hint.Type)
	}
	if hint.GroupType != "TEXT" {
		t.Errorf("Hint groupType = %q, want TEXT", hint.GroupType)
	}

	// Input block
	inp := req.Blocks[3]
	if inp.Type != "INPUT_EMAIL" {
		t.Errorf("Input type = %q, want INPUT_EMAIL", inp.Type)
	}

	// TITLE, hint, and input all have different groupUUIDs
	titleGroup := req.Blocks[1].GroupUUID
	hintGroup := req.Blocks[2].GroupUUID
	inputGroup := req.Blocks[3].GroupUUID
	if titleGroup == inputGroup {
		t.Error("TITLE and INPUT_EMAIL must have different groupUUIDs")
	}
	if hintGroup == titleGroup || hintGroup == inputGroup {
		t.Error("Hint TEXT must have its own groupUUID")
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

	// FORM_TITLE + TITLE + 2 cols + 2 rows = 6 (no MATRIX container)
	if len(req.Blocks) != 6 {
		t.Fatalf("Blocks = %d, want 6", len(req.Blocks))
	}

	// Columns start at index 2 (directly after TITLE)
	if req.Blocks[2].Type != "MATRIX_COLUMN" {
		t.Errorf("Block 2 type = %q, want MATRIX_COLUMN", req.Blocks[2].Type)
	}
	if req.Blocks[2].Payload["isRequired"] != true {
		t.Error("MATRIX_COLUMN should have isRequired")
	}
	// Check safeHTMLSchema instead of text
	schema, ok := req.Blocks[2].Payload["safeHTMLSchema"].([]any)
	if !ok || len(schema) == 0 {
		t.Error("MATRIX_COLUMN should have safeHTMLSchema")
	}

	// Rows
	if req.Blocks[4].Type != "MATRIX_ROW" {
		t.Errorf("Block 4 type = %q, want MATRIX_ROW", req.Blocks[4].Type)
	}
	if req.Blocks[4].Payload["isRequired"] != true {
		t.Error("MATRIX_ROW should have isRequired")
	}
	rowSchema, ok := req.Blocks[4].Payload["safeHTMLSchema"].([]any)
	if !ok || len(rowSchema) == 0 {
		t.Error("MATRIX_ROW should have safeHTMLSchema")
	}

	// TITLE and matrix content must have different groupUUIDs
	if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
		t.Error("TITLE and MATRIX_COLUMN must have different groupUUIDs")
	}

	// All matrix columns and rows share the same groupUUID
	contentGroup := req.Blocks[2].GroupUUID
	for i := 2; i < 6; i++ {
		if req.Blocks[i].GroupUUID != contentGroup {
			t.Errorf("Block %d groupUUID = %s, want %s (all matrix blocks share one group)", i, req.Blocks[i].GroupUUID, contentGroup)
		}
	}

	// Verify column/row index ordering
	if req.Blocks[2].Payload["isFirst"] != true {
		t.Error("First column should have isFirst=true")
	}
	if req.Blocks[2].Payload["isLast"] != false {
		t.Error("First column should have isLast=false")
	}
	if req.Blocks[3].Payload["isFirst"] != false {
		t.Error("Second column should have isFirst=false")
	}
	if req.Blocks[3].Payload["isLast"] != true {
		t.Error("Second column should have isLast=true")
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

	// Verify field reference uses the content groupUUID
	conditionals := condBlock.Payload["conditionals"].([]any)
	if len(conditionals) != 1 {
		t.Fatalf("conditionals = %d, want 1", len(conditionals))
	}
	cond := conditionals[0].(map[string]any)
	payload := cond["payload"].(map[string]any)
	field := payload["field"].(map[string]any)

	// field.uuid and field.blockGroupUuid must be the content group UUID (not a block uuid)
	fieldUUID := field["uuid"].(string)
	blockGroupUUID := field["blockGroupUuid"].(string)
	if fieldUUID != blockGroupUUID {
		t.Errorf("field.uuid (%s) != field.blockGroupUuid (%s), must be equal", fieldUUID, blockGroupUUID)
	}

	// The content group UUID must match the groupUuid of the option blocks
	optionGroupUUID := req.Blocks[2].GroupUUID // first MULTIPLE_CHOICE_OPTION
	if fieldUUID != optionGroupUUID {
		t.Errorf("field.uuid (%s) != option groupUuid (%s), must reference content group", fieldUUID, optionGroupUUID)
	}

	// Must not be a block uuid
	for _, b := range req.Blocks {
		if b.UUID == fieldUUID {
			t.Errorf("field.uuid (%s) matches a block UUID, must be a groupUuid", fieldUUID)
		}
	}

	// Verify value resolves to option UUID
	value := payload["value"].(string)
	yesOptUUID := req.Blocks[2].UUID // "Yes" option
	if value != yesOptUUID {
		t.Errorf("value = %s, want Yes option UUID %s", value, yesOptUUID)
	}

	if field["questionType"] != "MULTIPLE_CHOICE" {
		t.Errorf("questionType = %v, want MULTIPLE_CHOICE", field["questionType"])
	}
	if field["type"] != "INPUT_FIELD" {
		t.Errorf("type = %v, want INPUT_FIELD", field["type"])
	}
	if field["title"] != "Role?" {
		t.Errorf("title = %v, want Role?", field["title"])
	}

	// Verify actions
	actions := condBlock.Payload["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("Actions = %d, want 1", len(actions))
	}
	action := actions[0].(map[string]any)
	if action["type"] != "SHOW_BLOCKS" {
		t.Errorf("Action type = %v, want SHOW_BLOCKS", action["type"])
	}

	// showBlocks must reference the target question's block UUIDs
	showBlocks := action["payload"].(map[string]any)["showBlocks"].([]string)
	// F2 is long-text: TITLE + TEXTAREA = 2 block UUIDs
	if len(showBlocks) != 2 {
		t.Errorf("showBlocks = %d, want 2 (TITLE + TEXTAREA)", len(showBlocks))
	}
}

func TestCompileConditionalIsNotEmpty(t *testing.T) {
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
						{Field: "F1", Comparison: "is_not_empty"},
					},
				},
				&model.Question{
					ID:     "F2",
					Text:   "Which?",
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

	conditionals := condBlock.Payload["conditionals"].([]any)
	payload := conditionals[0].(map[string]any)["payload"].(map[string]any)

	if payload["comparison"] != "IS_NOT_EMPTY" {
		t.Errorf("comparison = %v, want IS_NOT_EMPTY", payload["comparison"])
	}
	// value must be empty string for IS_NOT_EMPTY
	if payload["value"] != "" {
		t.Errorf("value = %v, want empty string", payload["value"])
	}

	field := payload["field"].(map[string]any)
	if field["questionType"] != "CHECKBOXES" {
		t.Errorf("questionType = %v, want CHECKBOXES", field["questionType"])
	}
}

func TestCompileConditionalMultipleValues(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:   "F1",
					Text: "Pick?",
					Type: model.SingleChoice,
					Options: []model.Option{
						{Text: "A"},
						{Text: "B"},
						{Text: "C"},
					},
				},
				&model.Conditional{
					Targets:  []string{"F2"},
					Operator: "AND",
					Conditions: []model.Condition{
						{Field: "F1", Comparison: "is_any_of", Values: []string{"A", "B"}},
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

	conditionals := condBlock.Payload["conditionals"].([]any)
	payload := conditionals[0].(map[string]any)["payload"].(map[string]any)

	if payload["comparison"] != "IS_ANY_OF" {
		t.Errorf("comparison = %v, want IS_ANY_OF", payload["comparison"])
	}

	// Multiple values should be a slice
	values, ok := payload["value"].([]string)
	if !ok {
		t.Fatalf("value is not []string: %T", payload["value"])
	}
	if len(values) != 2 {
		t.Fatalf("values = %d, want 2", len(values))
	}

	// Values should be option UUIDs, not text
	optAUUID := req.Blocks[2].UUID
	optBUUID := req.Blocks[3].UUID
	if values[0] != optAUUID || values[1] != optBUUID {
		t.Errorf("values = %v, want [%s, %s]", values, optAUUID, optBUUID)
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

func TestCompileDropdown(t *testing.T) {
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

	// FORM_TITLE + TITLE + 2 options = 4
	if len(req.Blocks) != 4 {
		t.Fatalf("Blocks = %d, want 4", len(req.Blocks))
	}

	// Check dropdown option blocks
	for _, b := range req.Blocks {
		if b.Type != "DROPDOWN_OPTION" {
			continue
		}
		if b.GroupType != "DROPDOWN" {
			t.Errorf("GroupType = %q, want DROPDOWN", b.GroupType)
		}
		// Must not have MULTIPLE_CHOICE-specific fields
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

	// TITLE and dropdown options must have different groupUUIDs
	if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
		t.Error("TITLE and DROPDOWN_OPTION must have different groupUUIDs")
	}
	// All dropdown options share the same groupUUID
	if req.Blocks[2].GroupUUID != req.Blocks[3].GroupUUID {
		t.Error("All DROPDOWN_OPTIONs must share the same groupUUID")
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
		{"link", `Click <a href="https://example.com">here</a> now`, 3, "Click "},
		{"link_only", `<a href="mailto:a@b.com">a@b.com</a>`, 1, "a@b.com"},
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

	// Check link segment has correct href style
	linkSchema := SafeHTMLSchemaFromHTML(`Visit <a href="https://example.com">Example</a>!`)
	if len(linkSchema) != 3 {
		t.Fatalf("link segments = %d, want 3", len(linkSchema))
	}
	linkSeg := linkSchema[1].([]any)
	if linkSeg[0] != "Example" {
		t.Errorf("link text = %q, want Example", linkSeg[0])
	}
	linkStyles := linkSeg[1].([]any)
	if len(linkStyles) != 1 {
		t.Fatalf("link styles len = %d, want 1", len(linkStyles))
	}
	hrefPair := linkStyles[0].([]any)
	if hrefPair[0] != "href" || hrefPair[1] != "https://example.com" {
		t.Errorf("href = %v, want [href, https://example.com]", hrefPair)
	}
}

func TestSafeHTMLSchemaEdgeCases(t *testing.T) {
	// Empty string
	s := SafeHTMLSchemaFromHTML("")
	if len(s) != 1 {
		t.Errorf("empty: segments = %d, want 1", len(s))
	}

	// Unclosed bold tag — "Hello " + rest as plain
	s = SafeHTMLSchemaFromHTML("Hello <b>world")
	if len(s) < 1 {
		t.Error("unclosed bold: should produce at least 1 segment")
	}

	// Unclosed link tag
	s = SafeHTMLSchemaFromHTML(`Click <a href="url">here`)
	if len(s) < 1 {
		t.Error("unclosed link: should produce at least 1 segment")
	}

	// Nested: italic containing link
	s = SafeHTMLSchemaFromHTML(`<i>text <a href="url">link</a> more</i>`)
	// Should produce: italic "text ", link "link", italic " more"
	if len(s) != 3 {
		t.Fatalf("nested: segments = %d, want 3: %v", len(s), s)
	}
}

func TestApplyThankYouPage(t *testing.T) {
	settings := make(map[string]any)

	// Heading + text blocks
	page := model.Page{
		Blocks: []model.Block{
			&model.HeadingBlock{Text: "Thank you!", Level: 1},
			&model.TextBlock{HTML: "Your answers help us."},
			&model.TextBlock{HTML: "We will contact you."},
		},
	}
	applyThankYouPage(page, settings)
	if settings["closeMessageTitle"] != "Thank you!" {
		t.Errorf("title = %q", settings["closeMessageTitle"])
	}
	if settings["closeMessageDescription"] != "Your answers help us.\n\nWe will contact you." {
		t.Errorf("desc = %q", settings["closeMessageDescription"])
	}

	// Text-only (first text becomes title)
	settings2 := make(map[string]any)
	page2 := model.Page{
		Blocks: []model.Block{
			&model.TextBlock{HTML: "<b>Thanks</b>"},
			&model.TextBlock{HTML: "See you."},
		},
	}
	applyThankYouPage(page2, settings2)
	if settings2["closeMessageTitle"] != "Thanks" {
		t.Errorf("title = %q (should strip HTML)", settings2["closeMessageTitle"])
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct{ in, want string }{
		{"<b>bold</b>", "bold"},
		{"<i>italic</i>", "italic"},
		{"<b>A</b> and <i>B</i>", "A and B"},
		{"no tags", "no tags"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := stripHTMLTags(tt.in); got != tt.want {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPageHasQuestions(t *testing.T) {
	withQ := model.Page{Blocks: []model.Block{
		&model.TextBlock{HTML: "Intro"},
		&model.Question{ID: "F1", Text: "Q?", Type: model.ShortText},
	}}
	if !pageHasQuestions(withQ) {
		t.Error("should have questions")
	}

	withoutQ := model.Page{Blocks: []model.Block{
		&model.TextBlock{HTML: "Thanks"},
		&model.HeadingBlock{Text: "Done", Level: 1},
	}}
	if pageHasQuestions(withoutQ) {
		t.Error("should not have questions")
	}

	empty := model.Page{}
	if pageHasQuestions(empty) {
		t.Error("empty should not have questions")
	}
}

func TestBuildPageBreak(t *testing.T) {
	c := testCompiler()

	// Default button label
	pb := c.buildPageBreak("", 0, false)
	btn := pb.Payload["button"].(map[string]any)["label"]
	if btn != "Weiter" {
		t.Errorf("default label = %q, want Weiter", btn)
	}
	if pb.Payload["isFirst"] != true {
		t.Error("index 0 should be isFirst")
	}
	if pb.Payload["isLast"] != false {
		t.Error("should not be isLast")
	}

	// Custom label, isLast
	pb2 := c.buildPageBreak("Submit", 3, true)
	btn2 := pb2.Payload["button"].(map[string]any)["label"]
	if btn2 != "Submit" {
		t.Errorf("label = %q", btn2)
	}
	if pb2.Payload["isLast"] != true {
		t.Error("should be isLast")
	}
	if pb2.Payload["isFirst"] != false {
		t.Error("index 3 should not be isFirst")
	}
}

func TestCompileHeadingLevel1(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.HeadingBlock{Text: "Main Title", Level: 1},
			},
		}},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	h := req.Blocks[1] // after FORM_TITLE
	if h.Type != "HEADING_1" {
		t.Errorf("Type = %q, want HEADING_1", h.Type)
	}
	if h.GroupType != "HEADING_1" {
		t.Errorf("GroupType = %q, want HEADING_1", h.GroupType)
	}
}

func TestCompileThankYouPage(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{
			{Blocks: []model.Block{
				&model.Question{ID: "F1", Text: "Q?", Type: model.ShortText},
			}},
			{Blocks: []model.Block{
				&model.TextBlock{HTML: "<b>Thanks!</b>"},
				&model.TextBlock{HTML: "Your feedback matters."},
			}},
		},
	}

	c := testCompiler()
	req, err := c.Compile(form, testConfig())
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// Should NOT have TEXT blocks for thank-you — they go to settings
	for _, b := range req.Blocks {
		if b.Type == "TEXT" {
			t.Error("Thank-you TEXT blocks should not be in blocks array")
		}
	}

	settings := req.Settings.(map[string]any)
	if settings["closeMessageTitle"] != "Thanks!" {
		t.Errorf("closeMessageTitle = %q", settings["closeMessageTitle"])
	}
	if settings["closeMessageDescription"] != "Your feedback matters." {
		t.Errorf("closeMessageDescription = %q", settings["closeMessageDescription"])
	}
}

func TestCompileAllInputTypes(t *testing.T) {
	// Test all input question types compile to correct block types
	tests := []struct {
		qType     model.QuestionType
		blockType string
	}{
		{model.Number, "INPUT_NUMBER"},
		{model.Email, "INPUT_EMAIL"},
		{model.Phone, "INPUT_PHONE_NUMBER"},
		{model.URL, "INPUT_LINK"},
		{model.Date, "INPUT_DATE"},
		{model.Time, "INPUT_TIME"},
		{model.Rating, "RATING"},
		{model.FileUpload, "FILE_UPLOAD"},
		{model.Signature, "SIGNATURE"},
	}

	for _, tt := range tests {
		t.Run(string(tt.qType), func(t *testing.T) {
			form := &model.Form{
				Name: "Test",
				Pages: []model.Page{{
					Blocks: []model.Block{
						&model.Question{ID: "F1", Text: "Q?", Type: tt.qType, Required: true},
					},
				}},
			}
			c := testCompiler()
			req, err := c.Compile(form, testConfig())
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			// FORM_TITLE + TITLE + input = 3
			if len(req.Blocks) != 3 {
				t.Fatalf("Blocks = %d, want 3", len(req.Blocks))
			}
			if req.Blocks[2].Type != tt.blockType {
				t.Errorf("Type = %q, want %q", req.Blocks[2].Type, tt.blockType)
			}
			// groupUUID separation
			if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
				t.Error("TITLE and input must have different groupUUIDs")
			}
		})
	}
}

func TestBuildSettings(t *testing.T) {
	form := &model.Form{
		Name:     "Test",
		Password: "secret",
		Settings: map[string]any{"language": "de"},
		Pages:    []model.Page{{}},
	}

	cfg := testConfig()
	cfg.Settings = map[string]any{"hasProgressBar": true}
	cfg.Styles = `{"theme":"CUSTOM"}`

	c := testCompiler()
	req, err := c.Compile(form, cfg)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	settings := req.Settings.(map[string]any)
	if settings["hasProgressBar"] != true {
		t.Error("missing hasProgressBar from config")
	}
	if settings["styles"] != `{"theme":"CUSTOM"}` {
		t.Errorf("styles = %v", settings["styles"])
	}
	if settings["language"] != "de" {
		t.Error("missing language from form settings")
	}
	if req.Password != "secret" {
		t.Errorf("password = %q", req.Password)
	}
}

func TestCompileScale(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "How likely?",
					Type:     model.Scale,
					Required: true,
					Properties: map[string]any{
						"start":       0,
						"end":         10,
						"step":        1,
						"left-label":  "Not at all",
						"right-label": "Definitely",
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

	// FORM_TITLE + TITLE + LINEAR_SCALE = 3
	if len(req.Blocks) != 3 {
		t.Fatalf("Blocks = %d, want 3", len(req.Blocks))
	}

	scale := req.Blocks[2]
	if scale.Type != "LINEAR_SCALE" {
		t.Errorf("Type = %q, want LINEAR_SCALE", scale.Type)
	}
	if scale.Payload["startNumber"] != 0 {
		t.Errorf("startNumber = %v, want 0", scale.Payload["startNumber"])
	}
	if scale.Payload["endNumber"] != 10 {
		t.Errorf("endNumber = %v, want 10", scale.Payload["endNumber"])
	}
	if scale.Payload["leftLabel"] != "Not at all" {
		t.Errorf("leftLabel = %v", scale.Payload["leftLabel"])
	}
	if scale.Payload["rightLabel"] != "Definitely" {
		t.Errorf("rightLabel = %v", scale.Payload["rightLabel"])
	}

	// TITLE and scale must have different groupUUIDs
	if req.Blocks[1].GroupUUID == req.Blocks[2].GroupUUID {
		t.Error("TITLE and LINEAR_SCALE must have different groupUUIDs")
	}
}

func TestCompileConditionalFieldReferencesContentGroup(t *testing.T) {
	// Comprehensive test: build a form with multiple question types and conditionals,
	// verify all conditional field references use content groupUUIDs consistently.
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:   "F1",
					Text: "Choices?",
					Type: model.SingleChoice,
					Options: []model.Option{
						{Text: "A"},
						{Text: "B"},
					},
				},
				&model.Conditional{
					Targets:  []string{"F2"},
					Operator: "AND",
					Conditions: []model.Condition{
						{Field: "F1", Comparison: "is", Values: []string{"A"}},
					},
				},
				&model.Question{
					ID:     "F2",
					Text:   "Hidden text",
					Type:   model.ShortText,
					Hidden: true,
				},
				&model.Question{
					ID:   "F3",
					Text: "Long input?",
					Type: model.LongText,
				},
				&model.Conditional{
					Targets:  []string{"F4"},
					Operator: "AND",
					Conditions: []model.Condition{
						{Field: "F3", Comparison: "is_not_empty"},
					},
				},
				&model.Question{
					ID:     "F4",
					Text:   "Follow-up",
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

	// Collect all blocks by type for reference
	blocksByUUID := make(map[string]*TallyBlock)
	groupMembers := make(map[string][]string) // groupUUID → []blockType
	for i := range req.Blocks {
		b := &req.Blocks[i]
		blocksByUUID[b.UUID] = b
		groupMembers[b.GroupUUID] = append(groupMembers[b.GroupUUID], b.Type)
	}

	// Find all conditionals and verify their field references
	for i, b := range req.Blocks {
		if b.Type != "CONDITIONAL_LOGIC" {
			continue
		}
		conditionals := b.Payload["conditionals"].([]any)
		for _, c := range conditionals {
			payload := c.(map[string]any)["payload"].(map[string]any)
			field := payload["field"].(map[string]any)
			fuuid := field["uuid"].(string)
			bgUUID := field["blockGroupUuid"].(string)

			// uuid == blockGroupUuid
			if fuuid != bgUUID {
				t.Errorf("[%d] field.uuid != field.blockGroupUuid: %s != %s", i, fuuid, bgUUID)
			}

			// Must be a groupUUID, not a block UUID
			if _, isBlockUUID := blocksByUUID[fuuid]; isBlockUUID {
				t.Errorf("[%d] field.uuid %s is a block UUID, must be a content groupUUID", i, fuuid)
			}

			// Must match some blocks' groupUUID
			if _, hasMembers := groupMembers[fuuid]; !hasMembers {
				t.Errorf("[%d] field.uuid %s doesn't match any block's groupUUID", i, fuuid)
			}
		}
	}
}
