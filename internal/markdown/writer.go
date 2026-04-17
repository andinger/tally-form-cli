package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/andinger/tally-form-cli/internal/model"
)

var (
	htmlBoldRe   = regexp.MustCompile(`<b>([^<]+)</b>`)
	htmlItalicRe = regexp.MustCompile(`<i>([^<]+)</i>`)
	htmlLinkRe   = regexp.MustCompile(`<a href="([^"]+)">([^<]+)</a>`)

	// existingQuestionPrefixRe strips a leading `F<n>: ` from question text
	// before the writer prepends its own `F<id>: ` prefix. This keeps
	// round-trips idempotent when a form was pushed with `strip_prefix: ""`
	// (the Tally-side title contains the prefix, the writer would otherwise
	// emit it twice on `pull`).
	existingQuestionPrefixRe = regexp.MustCompile(`^F\d+:\s*`)
)

// Write converts an IR Form back to Markdown format.
func Write(form *model.Form) string {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", form.Name))
	if form.FormID != "" {
		b.WriteString(fmt.Sprintf("form_id: %q\n", form.FormID))
	}
	if form.Workspace != "" {
		b.WriteString(fmt.Sprintf("workspace: %q\n", form.Workspace))
	}
	if form.Password != "" {
		b.WriteString(fmt.Sprintf("password: %q\n", form.Password))
	}
	b.WriteString("---\n")

	for i, page := range form.Pages {
		if i > 0 {
			b.WriteString("\n---\n")
			if page.ButtonLabel != "" {
				b.WriteString(fmt.Sprintf("> button: %q\n", page.ButtonLabel))
			}
		}

		for _, block := range page.Blocks {
			b.WriteString("\n")
			writeBlock(&b, block)
		}
	}

	return b.String()
}

func writeBlock(b *strings.Builder, block model.Block) {
	switch bl := block.(type) {
	case *model.HeadingBlock:
		prefix := "##"
		if bl.Level == 1 {
			prefix = "#"
		}
		b.WriteString(fmt.Sprintf("%s %s\n", prefix, bl.Text))

	case *model.TextBlock:
		text := htmlToMarkdown(bl.HTML)
		b.WriteString(text + "\n")

	case *model.Question:
		writeQuestion(b, bl)

	case *model.Conditional:
		writeConditional(b, bl)
	}
}

func writeQuestion(b *strings.Builder, q *model.Question) {
	text := existingQuestionPrefixRe.ReplaceAllString(q.Text, "")
	b.WriteString(fmt.Sprintf("%s: %s\n", q.ID, text))
	b.WriteString(fmt.Sprintf("> type: %s\n", q.Type))

	if !q.Required {
		b.WriteString("> required: false\n")
	}
	if q.Hidden {
		b.WriteString("> hidden: true\n")
	}
	if q.Hint != "" {
		b.WriteString(fmt.Sprintf("> hint: %q\n", q.Hint))
	}
	if q.Placeholder != "" {
		b.WriteString(fmt.Sprintf("> placeholder: %q\n", q.Placeholder))
	}

	// Properties
	if v, ok := q.Properties["max"]; ok {
		b.WriteString(fmt.Sprintf("> max: %d\n", v))
	}
	if v, ok := q.Properties["min"]; ok {
		b.WriteString(fmt.Sprintf("> min: %d\n", v))
	}
	if v, ok := q.Properties["stars"]; ok {
		b.WriteString(fmt.Sprintf("> stars: %d\n", v))
	}
	if v, ok := q.Properties["start"]; ok {
		b.WriteString(fmt.Sprintf("> start: %d\n", v))
	}
	if v, ok := q.Properties["end"]; ok {
		b.WriteString(fmt.Sprintf("> end: %d\n", v))
	}
	if v, ok := q.Properties["step"]; ok {
		b.WriteString(fmt.Sprintf("> step: %d\n", v))
	}
	if v, ok := q.Properties["left-label"]; ok {
		b.WriteString(fmt.Sprintf("> left-label: %q\n", v))
	}
	if v, ok := q.Properties["right-label"]; ok {
		b.WriteString(fmt.Sprintf("> right-label: %q\n", v))
	}

	// Matrix columns
	if len(q.MatrixCols) > 0 {
		b.WriteString(fmt.Sprintf("> columns: %s\n", strings.Join(q.MatrixCols, ", ")))
	}

	// Options
	for _, opt := range q.Options {
		text := htmlToMarkdown(opt.Text)
		if opt.IsOther {
			b.WriteString(fmt.Sprintf("- %s {other}\n", text))
		} else {
			b.WriteString(fmt.Sprintf("- %s\n", text))
		}
	}

	// Matrix rows
	for _, row := range q.MatrixRows {
		b.WriteString(fmt.Sprintf("- %s\n", htmlToMarkdown(row)))
	}
}

func writeConditional(b *strings.Builder, c *model.Conditional) {
	targets := strings.Join(c.Targets, ", ")
	var conditions []string
	for _, cond := range c.Conditions {
		s := cond.Field + " " + cond.Comparison
		if len(cond.Values) > 0 {
			var quoted []string
			for _, v := range cond.Values {
				quoted = append(quoted, fmt.Sprintf("%q", v))
			}
			s += " " + strings.Join(quoted, ", ")
		}
		conditions = append(conditions, s)
	}

	operator := " and "
	if c.Operator == "OR" {
		operator = " or "
	}

	b.WriteString(fmt.Sprintf("> show %s when %s\n", targets, strings.Join(conditions, operator)))
}

func htmlToMarkdown(s string) string {
	s = htmlBoldRe.ReplaceAllString(s, "**$1**")
	s = htmlItalicRe.ReplaceAllString(s, "*$1*")
	s = htmlLinkRe.ReplaceAllString(s, "[$2]($1)")
	return s
}
