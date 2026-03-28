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
