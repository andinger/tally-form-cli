package markdown

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/andinger/tally-form-cli/internal/model"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", name)
}

func mustReadFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", name, err)
	}
	return string(data)
}

func TestParseMinimal(t *testing.T) {
	form, err := Parse(mustReadFile(t, "minimal.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if form.Name != "Test Form" {
		t.Errorf("Name = %q, want %q", form.Name, "Test Form")
	}
	if len(form.Pages) != 1 {
		t.Fatalf("Pages = %d, want 1", len(form.Pages))
	}
	blocks := form.Pages[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("Blocks = %d, want 1", len(blocks))
	}
	q, ok := blocks[0].(*model.Question)
	if !ok {
		t.Fatalf("Block 0 is %T, want *Question", blocks[0])
	}
	if q.ID != "F1" {
		t.Errorf("ID = %q, want %q", q.ID, "F1")
	}
	if q.Type != model.ShortText {
		t.Errorf("Type = %q, want %q", q.Type, model.ShortText)
	}
	if !q.Required {
		t.Error("Expected Required = true")
	}
}

func TestParseSingleChoice(t *testing.T) {
	form, err := Parse(mustReadFile(t, "single_choice.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Type != model.SingleChoice {
		t.Errorf("Type = %q, want %q", q.Type, model.SingleChoice)
	}
	if len(q.Options) != 3 {
		t.Fatalf("Options = %d, want 3", len(q.Options))
	}
	if q.Options[0].Text != "Manager" {
		t.Errorf("Option 0 = %q, want %q", q.Options[0].Text, "Manager")
	}
	if q.Options[2].Text != "Designer" {
		t.Errorf("Option 2 = %q, want %q", q.Options[2].Text, "Designer")
	}
}

func TestParseMultiChoiceOther(t *testing.T) {
	form, err := Parse(mustReadFile(t, "multi_choice_other.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Type != model.MultiChoice {
		t.Errorf("Type = %q, want %q", q.Type, model.MultiChoice)
	}
	if len(q.Options) != 4 {
		t.Fatalf("Options = %d, want 4", len(q.Options))
	}
	last := q.Options[3]
	if !last.IsOther {
		t.Error("Expected last option to be IsOther")
	}
	if last.Text != "Andere" {
		t.Errorf("Other text = %q, want %q", last.Text, "Andere")
	}
	if q.Properties["max"] != 3 {
		t.Errorf("max = %v, want 3", q.Properties["max"])
	}
}

func TestParseMultiPage(t *testing.T) {
	form, err := Parse(mustReadFile(t, "multi_page.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(form.Pages) != 3 {
		t.Fatalf("Pages = %d, want 3", len(form.Pages))
	}
	if form.Pages[1].ButtonLabel != "Weiter zu Seite 2 / 3" {
		t.Errorf("Page 1 button = %q, want %q", form.Pages[1].ButtonLabel, "Weiter zu Seite 2 / 3")
	}
	if form.Pages[2].ButtonLabel != "Absenden" {
		t.Errorf("Page 2 button = %q, want %q", form.Pages[2].ButtonLabel, "Absenden")
	}
	// Each page has a heading + question
	for i, page := range form.Pages {
		hasQuestion := false
		for _, b := range page.Blocks {
			if _, ok := b.(*model.Question); ok {
				hasQuestion = true
			}
		}
		if !hasQuestion {
			t.Errorf("Page %d has no question", i)
		}
	}
}

func TestParseConditional(t *testing.T) {
	form, err := Parse(mustReadFile(t, "conditional.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	blocks := form.Pages[0].Blocks
	// Expect: F1 (question), conditional, F2 (question)
	var cond *model.Conditional
	for _, b := range blocks {
		if c, ok := b.(*model.Conditional); ok {
			cond = c
			break
		}
	}
	if cond == nil {
		t.Fatal("No conditional found")
	}
	if len(cond.Targets) != 1 || cond.Targets[0] != "F2" {
		t.Errorf("Targets = %v, want [F2]", cond.Targets)
	}
	if len(cond.Conditions) != 1 {
		t.Fatalf("Conditions = %d, want 1", len(cond.Conditions))
	}
	c := cond.Conditions[0]
	if c.Field != "F1" {
		t.Errorf("Field = %q, want %q", c.Field, "F1")
	}
	if c.Comparison != "is" {
		t.Errorf("Comparison = %q, want %q", c.Comparison, "is")
	}
	if len(c.Values) != 1 || c.Values[0] != "Yes, actively" {
		t.Errorf("Values = %v, want [Yes, actively]", c.Values)
	}

	// F2 should be hidden
	var f2 *model.Question
	for _, b := range blocks {
		if q, ok := b.(*model.Question); ok && q.ID == "F2" {
			f2 = q
		}
	}
	if f2 == nil {
		t.Fatal("F2 not found")
	}
	if !f2.Hidden {
		t.Error("Expected F2 to be hidden")
	}
}

func TestParseConditionalComplex(t *testing.T) {
	form, err := Parse(mustReadFile(t, "conditional_complex.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	var cond *model.Conditional
	for _, b := range form.Pages[0].Blocks {
		if c, ok := b.(*model.Conditional); ok {
			cond = c
			break
		}
	}
	if cond == nil {
		t.Fatal("No conditional found")
	}
	if len(cond.Targets) != 2 {
		t.Errorf("Targets = %v, want 2 targets", cond.Targets)
	}
	if cond.Operator != "AND" {
		t.Errorf("Operator = %q, want %q", cond.Operator, "AND")
	}
	if len(cond.Conditions) != 2 {
		t.Fatalf("Conditions = %d, want 2", len(cond.Conditions))
	}
	if cond.Conditions[0].Comparison != "is_not_any_of" {
		t.Errorf("Condition 0 comparison = %q, want %q", cond.Conditions[0].Comparison, "is_not_any_of")
	}
	if len(cond.Conditions[0].Values) != 2 {
		t.Errorf("Condition 0 values = %d, want 2", len(cond.Conditions[0].Values))
	}
	if cond.Conditions[1].Comparison != "is_not_empty" {
		t.Errorf("Condition 1 comparison = %q, want %q", cond.Conditions[1].Comparison, "is_not_empty")
	}
}

func TestParseMatrix(t *testing.T) {
	form, err := Parse(mustReadFile(t, "matrix.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Type != model.Matrix {
		t.Errorf("Type = %q, want %q", q.Type, model.Matrix)
	}
	if len(q.MatrixCols) != 4 {
		t.Errorf("MatrixCols = %d, want 4", len(q.MatrixCols))
	}
	if q.MatrixCols[0] != "Low" {
		t.Errorf("Col 0 = %q, want %q", q.MatrixCols[0], "Low")
	}
	if len(q.MatrixRows) != 3 {
		t.Errorf("MatrixRows = %d, want 3", len(q.MatrixRows))
	}
}

func TestParseHeadingAndText(t *testing.T) {
	form, err := Parse(mustReadFile(t, "heading_and_text.md"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	blocks := form.Pages[0].Blocks
	if len(blocks) < 3 {
		t.Fatalf("Blocks = %d, want >= 3", len(blocks))
	}

	// First block should be text with italic converted
	tb, ok := blocks[0].(*model.TextBlock)
	if !ok {
		t.Fatalf("Block 0 is %T, want *TextBlock", blocks[0])
	}
	if tb.HTML != "Welcome to this form. Please answer <i>honestly</i>." {
		t.Errorf("HTML = %q", tb.HTML)
	}

	// Second block should be heading
	hb, ok := blocks[1].(*model.HeadingBlock)
	if !ok {
		t.Fatalf("Block 1 is %T, want *HeadingBlock", blocks[1])
	}
	if hb.Text != "Section One" {
		t.Errorf("Heading = %q, want %q", hb.Text, "Section One")
	}

	// Third block should be text
	tb2, ok := blocks[2].(*model.TextBlock)
	if !ok {
		t.Fatalf("Block 2 is %T, want *TextBlock", blocks[2])
	}
	if tb2.HTML != "This section covers <i>important</i> topics." {
		t.Errorf("HTML = %q", tb2.HTML)
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := `---
name: "KI-Hebel-Check — RKS"
workspace: mOJGz8
password: "rks-check"
form_id: "81d6KA"
---

F1: Test?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if form.Name != "KI-Hebel-Check — RKS" {
		t.Errorf("Name = %q", form.Name)
	}
	if form.Workspace != "mOJGz8" {
		t.Errorf("Workspace = %q", form.Workspace)
	}
	if form.Password != "rks-check" {
		t.Errorf("Password = %q", form.Password)
	}
	if form.FormID != "81d6KA" {
		t.Errorf("FormID = %q", form.FormID)
	}
}

func TestParseOtherVariant2(t *testing.T) {
	content := `---
name: "Other Test"
---

F1: Where are your data?
> type: multi-choice
> other: true
- System A
- System B
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if len(q.Options) != 3 {
		t.Fatalf("Options = %d, want 3 (2 + auto other)", len(q.Options))
	}
	if !q.Options[2].IsOther {
		t.Error("Expected auto-added other option")
	}
	if q.Options[2].Text != "Andere" {
		t.Errorf("Other text = %q, want %q", q.Options[2].Text, "Andere")
	}
}

func TestParseHint(t *testing.T) {
	content := `---
name: "Hint Test"
---

F1: Describe your work
> type: long-text
> hint: "For example: daily tasks, recurring meetings"
> placeholder: "Type here..."
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Hint != "For example: daily tasks, recurring meetings" {
		t.Errorf("Hint = %q", q.Hint)
	}
	if q.Placeholder != "Type here..." {
		t.Errorf("Placeholder = %q", q.Placeholder)
	}
}

func TestParseScaleProperties(t *testing.T) {
	content := `---
name: "Scale Test"
---

F1: How satisfied?
> type: scale
> start: 1
> end: 10
> step: 1
> left-label: "Not at all"
> right-label: "Very much"
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Type != model.Scale {
		t.Errorf("Type = %q, want %q", q.Type, model.Scale)
	}
	if q.Properties["start"] != 1 {
		t.Errorf("start = %v, want 1", q.Properties["start"])
	}
	if q.Properties["end"] != 10 {
		t.Errorf("end = %v, want 10", q.Properties["end"])
	}
	if q.Properties["left-label"] != "Not at all" {
		t.Errorf("left-label = %v", q.Properties["left-label"])
	}
}

func TestParseItalicInOptions(t *testing.T) {
	content := `---
name: "Italic Test"
---

F1: Which tasks?
> type: multi-choice
- Data entry — *e.g. emails, PDFs*
- Research — *e.g. finding documents*
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Options[0].Text != "Data entry — <i>e.g. emails, PDFs</i>" {
		t.Errorf("Option 0 = %q", q.Options[0].Text)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	content := `F1: Question?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if form.Name != "" {
		t.Errorf("Name = %q, want empty", form.Name)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Text != "Question?" {
		t.Errorf("Text = %q", q.Text)
	}
}

func TestParseMalformedFrontmatter(t *testing.T) {
	// Missing closing ---
	content := `---
name: "Test"
F1: Question?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// Should treat whole thing as body since no closing ---
	if form.Name != "" {
		t.Errorf("Name = %q, want empty (no closing ---)", form.Name)
	}
}

func TestParseConditionalOR(t *testing.T) {
	content := `---
name: "Test"
---

F1: Q?
> type: single-choice
- A
- B
- C

> show F2 when F1 is "A" or F1 is "B"

F2: Details?
> type: long-text
> hidden: true
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	var cond *model.Conditional
	for _, b := range form.Pages[0].Blocks {
		if c, ok := b.(*model.Conditional); ok {
			cond = c
			break
		}
	}
	if cond == nil {
		t.Fatal("No conditional found")
	}
	if cond.Operator != "OR" {
		t.Errorf("Operator = %q, want OR", cond.Operator)
	}
	if len(cond.Conditions) != 2 {
		t.Fatalf("Conditions = %d, want 2", len(cond.Conditions))
	}
}

func TestParseAllMetadata(t *testing.T) {
	content := `---
name: "Test"
---

F1: Scale?
> type: scale
> required: false
> placeholder: "Type here"
> hidden: true
> stars: 5
> start: 1
> end: 10
> step: 2
> left-label: "Low"
> right-label: "High"
> min: 1
> max: 5
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Required {
		t.Error("should not be required")
	}
	if q.Placeholder != "Type here" {
		t.Errorf("Placeholder = %q", q.Placeholder)
	}
	if !q.Hidden {
		t.Error("should be hidden")
	}
	if q.Properties["stars"] != 5 {
		t.Errorf("stars = %v", q.Properties["stars"])
	}
	if q.Properties["start"] != 1 {
		t.Errorf("start = %v", q.Properties["start"])
	}
	if q.Properties["end"] != 10 {
		t.Errorf("end = %v", q.Properties["end"])
	}
	if q.Properties["step"] != 2 {
		t.Errorf("step = %v", q.Properties["step"])
	}
	if q.Properties["left-label"] != "Low" {
		t.Errorf("left-label = %v", q.Properties["left-label"])
	}
	if q.Properties["right-label"] != "High" {
		t.Errorf("right-label = %v", q.Properties["right-label"])
	}
	if q.Properties["min"] != 1 {
		t.Errorf("min = %v", q.Properties["min"])
	}
	if q.Properties["max"] != 5 {
		t.Errorf("max = %v", q.Properties["max"])
	}
}

func TestParseLinkInText(t *testing.T) {
	content := `---
name: "Test"
---

Visit [our site](https://example.com) for details.
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	tb := form.Pages[0].Blocks[0].(*model.TextBlock)
	expected := `Visit <a href="https://example.com">our site</a> for details.`
	if tb.HTML != expected {
		t.Errorf("HTML = %q, want %q", tb.HTML, expected)
	}
}

func TestParseBoldInText(t *testing.T) {
	content := `---
name: "Test"
---

This is **important** text.
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	tb := form.Pages[0].Blocks[0].(*model.TextBlock)
	if tb.HTML != "This is <b>important</b> text." {
		t.Errorf("HTML = %q", tb.HTML)
	}
}

func TestParseInvalidCondition(t *testing.T) {
	content := `---
name: "Test"
---

F1: Q?
> type: short-text

> show F2 when invalid
`
	_, err := Parse(content)
	if err == nil {
		t.Error("Expected error for invalid condition with only 1 part")
	}
}

func TestParseFrontmatterWithSettings(t *testing.T) {
	content := `---
name: "Test"
workspace: "ws123"
password: "secret"
language: "de"
---

F1: Q?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if form.Workspace != "ws123" {
		t.Errorf("Workspace = %q", form.Workspace)
	}
	if form.Password != "secret" {
		t.Errorf("Password = %q", form.Password)
	}
	if form.Settings["language"] != "de" {
		t.Errorf("Settings = %v", form.Settings)
	}
}

func TestParseStripPrefixDefault(t *testing.T) {
	// No strip_prefix field: default behavior strips `F<n>:`.
	content := `---
name: "Test"
---

F1: In welcher Rolle?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.ID != "F1" {
		t.Errorf("ID = %q, want %q", q.ID, "F1")
	}
	if q.Text != "In welcher Rolle?" {
		t.Errorf("Text = %q, want %q", q.Text, "In welcher Rolle?")
	}
}

func TestParseStripPrefixEmpty(t *testing.T) {
	// strip_prefix: "" keeps the prefix visible in the displayed text.
	content := `---
name: "Test"
strip_prefix: ""
---

F1: In welcher Rolle?
> type: short-text
F2: Welche Abteilung?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q1 := form.Pages[0].Blocks[0].(*model.Question)
	if q1.ID != "F1" {
		t.Errorf("F1 ID = %q", q1.ID)
	}
	if q1.Text != "F1: In welcher Rolle?" {
		t.Errorf("F1 Text = %q, want %q", q1.Text, "F1: In welcher Rolle?")
	}
	q2 := form.Pages[0].Blocks[1].(*model.Question)
	if q2.Text != "F2: Welche Abteilung?" {
		t.Errorf("F2 Text = %q, want %q", q2.Text, "F2: Welche Abteilung?")
	}
}

func TestParseStripPrefixCustom(t *testing.T) {
	// Custom regex: strip leading digits and colon but keep the "F".
	content := `---
name: "Test"
strip_prefix: "^F\\d+:\\s*"
---

F1: In welcher Rolle?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	q := form.Pages[0].Blocks[0].(*model.Question)
	if q.Text != "In welcher Rolle?" {
		t.Errorf("Text = %q, want %q", q.Text, "In welcher Rolle?")
	}
}

func TestParseStripPrefixInvalidRegex(t *testing.T) {
	content := `---
name: "Test"
strip_prefix: "[unclosed"
---

F1: Q?
> type: short-text
`
	_, err := Parse(content)
	if err == nil {
		t.Fatal("Expected error for invalid strip_prefix regex")
	}
}

func TestParseStripPrefixNotLeakedToSettings(t *testing.T) {
	// strip_prefix is a build-time option and must not be forwarded to Tally.
	content := `---
name: "Test"
strip_prefix: ""
language: "de"
---

F1: Q?
> type: short-text
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if _, ok := form.Settings["strip_prefix"]; ok {
		t.Error("strip_prefix leaked into Settings")
	}
	if form.Settings["language"] != "de" {
		t.Errorf("language missing from Settings: %v", form.Settings)
	}
}

func TestParseStripPrefixConditionalStillReferencesID(t *testing.T) {
	// With strip_prefix: "", the F-ID still drives conditional references.
	content := `---
name: "Test"
strip_prefix: ""
---

F1: Rolle?
> type: single-choice
- Chef
- Mitarbeiter

> show F2 when F1 is_any_of "Chef"

F2: Details?
> type: short-text
> hidden: true
`
	form, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	blocks := form.Pages[0].Blocks
	q1 := blocks[0].(*model.Question)
	if q1.Text != "F1: Rolle?" {
		t.Errorf("F1 Text = %q", q1.Text)
	}
	cond, ok := blocks[1].(*model.Conditional)
	if !ok {
		t.Fatalf("Expected Conditional at index 1, got %T", blocks[1])
	}
	if cond.Targets[0] != "F2" || cond.Conditions[0].Field != "F1" {
		t.Errorf("Conditional = %+v", cond)
	}
}
