package markdown

import (
	"strings"
	"testing"

	"github.com/andinger/tally-form-cli/internal/model"
)

func TestWriteMinimal(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Name?",
					Type:     model.ShortText,
					Required: true,
				},
			},
		}},
	}

	output := Write(form)
	if !strings.Contains(output, `name: "Test"`) {
		t.Errorf("Missing name in frontmatter: %s", output)
	}
	if !strings.Contains(output, "F1: Name?") {
		t.Errorf("Missing question: %s", output)
	}
	if !strings.Contains(output, "> type: short-text") {
		t.Errorf("Missing type: %s", output)
	}
}

func TestWriteSingleChoice(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:       "F1",
					Text:     "Role?",
					Type:     model.SingleChoice,
					Required: true,
					Options: []model.Option{
						{Text: "Manager"},
						{Text: "Other", IsOther: true},
					},
				},
			},
		}},
	}

	output := Write(form)
	if !strings.Contains(output, "- Manager") {
		t.Errorf("Missing option: %s", output)
	}
	if !strings.Contains(output, "- Other {other}") {
		t.Errorf("Missing other option: %s", output)
	}
}

func TestWriteMultiPage(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{
			{Blocks: []model.Block{
				&model.Question{ID: "F1", Text: "Q1", Type: model.ShortText, Required: true},
			}},
			{ButtonLabel: "Next", Blocks: []model.Block{
				&model.Question{ID: "F2", Text: "Q2", Type: model.ShortText, Required: true},
			}},
		},
	}

	output := Write(form)
	if !strings.Contains(output, "\n---\n") {
		t.Errorf("Missing page break: %s", output)
	}
	if !strings.Contains(output, `> button: "Next"`) {
		t.Errorf("Missing button label: %s", output)
	}
}

func TestWriteConditional(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Conditional{
					Targets:  []string{"F2", "F3"},
					Operator: "AND",
					Conditions: []model.Condition{
						{Field: "F1", Comparison: "is_not_any_of", Values: []string{"Option A", "Option B"}},
					},
				},
			},
		}},
	}

	output := Write(form)
	if !strings.Contains(output, `> show F2, F3 when F1 is_not_any_of "Option A", "Option B"`) {
		t.Errorf("Wrong conditional: %s", output)
	}
}

func TestWriteMatrix(t *testing.T) {
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

	output := Write(form)
	if !strings.Contains(output, "> columns: Low, High") {
		t.Errorf("Missing columns: %s", output)
	}
	if !strings.Contains(output, "- Reports") {
		t.Errorf("Missing row: %s", output)
	}
}

func TestWriteItalicRoundTrip(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.TextBlock{HTML: "Hello <i>world</i>!"},
			},
		}},
	}

	output := Write(form)
	if !strings.Contains(output, "Hello *world*!") {
		t.Errorf("Italic conversion: %s", output)
	}
}

func TestRoundTrip(t *testing.T) {
	input := mustReadFile(t, "single_choice.md")
	form1, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse 1: %v", err)
	}

	md := Write(form1)
	form2, err := Parse(md)
	if err != nil {
		t.Fatalf("Parse 2: %v", err)
	}

	// Compare structures
	if form1.Name != form2.Name {
		t.Errorf("Name: %q vs %q", form1.Name, form2.Name)
	}
	if len(form1.Pages) != len(form2.Pages) {
		t.Fatalf("Pages: %d vs %d", len(form1.Pages), len(form2.Pages))
	}

	q1 := form1.Pages[0].Blocks[0].(*model.Question)
	q2 := form2.Pages[0].Blocks[0].(*model.Question)
	if q1.Type != q2.Type {
		t.Errorf("Type: %q vs %q", q1.Type, q2.Type)
	}
	if len(q1.Options) != len(q2.Options) {
		t.Errorf("Options: %d vs %d", len(q1.Options), len(q2.Options))
	}
}

func TestWriteHeadingLevel1(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.HeadingBlock{Text: "Main", Level: 1},
			},
		}},
	}
	out := Write(form)
	if !strings.Contains(out, "# Main\n") {
		t.Errorf("Missing # heading: %q", out)
	}
}

func TestWriteAllQuestionMetadata(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:          "F1",
					Text:        "Scale Q",
					Type:        model.Scale,
					Required:    false,
					Hidden:      true,
					Placeholder: "Enter",
					Hint:        "A hint",
					Properties: map[string]any{
						"max": 5, "min": 1, "stars": 3,
						"start": 0, "end": 10, "step": 2,
						"left-label": "Low", "right-label": "High",
					},
				},
			},
		}},
	}
	out := Write(form)
	for _, want := range []string{
		"> type: scale\n",
		"> required: false\n",
		"> hidden: true\n",
		`> hint: "A hint"`,
		`> placeholder: "Enter"`,
		"> max: 5\n",
		"> min: 1\n",
		"> stars: 3\n",
		"> start: 0\n",
		"> end: 10\n",
		"> step: 2\n",
		`> left-label: "Low"`,
		`> right-label: "High"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Missing %q in output:\n%s", want, out)
		}
	}
}

func TestWriteMatrixRows(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:         "F1",
					Text:       "Rate?",
					Type:       model.Matrix,
					Required:   true,
					MatrixCols: []string{"Low", "High"},
					MatrixRows: []string{"Item A", "Item B"},
				},
			},
		}},
	}
	out := Write(form)
	if !strings.Contains(out, "> columns: Low, High\n") {
		t.Errorf("Missing columns: %q", out)
	}
	if !strings.Contains(out, "- Item A\n") || !strings.Contains(out, "- Item B\n") {
		t.Errorf("Missing rows: %q", out)
	}
}

func TestWriteConditionalOR(t *testing.T) {
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Conditional{
					Targets:  []string{"F2"},
					Operator: "OR",
					Conditions: []model.Condition{
						{Field: "F1", Comparison: "is", Values: []string{"A"}},
						{Field: "F1", Comparison: "is", Values: []string{"B"}},
					},
				},
			},
		}},
	}
	out := Write(form)
	if !strings.Contains(out, " or ") {
		t.Errorf("Missing 'or': %q", out)
	}
}

func TestWritePassword(t *testing.T) {
	form := &model.Form{
		Name:     "Test",
		Password: "secret",
		Pages:    []model.Page{{}},
	}
	out := Write(form)
	if !strings.Contains(out, `password: "secret"`) {
		t.Errorf("Missing password: %q", out)
	}
}

func TestWriteOtherOption(t *testing.T) {
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
						{Text: "Other", IsOther: true},
					},
				},
			},
		}},
	}
	out := Write(form)
	if !strings.Contains(out, "- Other {other}\n") {
		t.Errorf("Missing {other}: %q", out)
	}
}

func TestHtmlToMarkdownLink(t *testing.T) {
	got := htmlToMarkdown(`Visit <a href="https://example.com">our site</a> now`)
	want := "Visit [our site](https://example.com) now"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteStripsExistingPrefix(t *testing.T) {
	// Guards against double-prefix when pulling a form whose stored title
	// already contains `F<n>:` (e.g. pushed with strip_prefix: "").
	form := &model.Form{
		Name: "Test",
		Pages: []model.Page{{
			Blocks: []model.Block{
				&model.Question{
					ID:   "F1",
					Text: "F1: Real question?",
					Type: model.ShortText,
				},
			},
		}},
	}
	out := Write(form)
	if strings.Contains(out, "F1: F1:") {
		t.Errorf("Double prefix in output:\n%s", out)
	}
	if !strings.Contains(out, "F1: Real question?\n") {
		t.Errorf("Expected normalized question line:\n%s", out)
	}
}

func TestHtmlToMarkdownBoldAndItalic(t *testing.T) {
	got := htmlToMarkdown("<b>bold</b> and <i>italic</i>")
	want := "**bold** and *italic*"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
