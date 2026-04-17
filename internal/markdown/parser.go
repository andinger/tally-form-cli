package markdown

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/andinger/tally-form-cli/internal/model"
)

var (
	questionRe    = regexp.MustCompile(`^F(\d+):\s+(.+)$`)
	metaRe        = regexp.MustCompile(`^>\s+(\S+):\s+(.+)$`)
	optionRe      = regexp.MustCompile(`^-\s+(.+)$`)
	headingRe     = regexp.MustCompile(`^(#{1,2})\s+(.+)$`)
	buttonRe      = regexp.MustCompile(`^>\s*button:\s*"(.+)"$`)
	conditionalRe = regexp.MustCompile(`^>\s*show\s+(.+?)\s+when\s+(.+)$`)
	boldRe        = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRe      = regexp.MustCompile(`\*([^*]+)\*`)
	linkRe        = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	otherMarkerRe = regexp.MustCompile(`\s*\{other\}\s*$`)

	// DefaultStripPrefix is applied to a question's displayed text when the
	// `strip_prefix` frontmatter field is absent. It removes the `F<n>:` ID
	// marker so the prefix stays out of the pushed Tally form while remaining
	// available as a conditional-logic reference in the Markdown source.
	DefaultStripPrefix = regexp.MustCompile(`^F\d+:\s*`)
)

type frontmatter struct {
	Name        string         `yaml:"name"`
	FormID      string         `yaml:"form_id"`
	Workspace   string         `yaml:"workspace"`
	Password    string         `yaml:"password"`
	StripPrefix *string        `yaml:"strip_prefix"`
	Settings    map[string]any `yaml:",inline"`
}

// Parse converts a Markdown string into an IR Form.
func Parse(content string) (*model.Form, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}

	form := &model.Form{
		Name:      fm.Name,
		FormID:    fm.FormID,
		Workspace: fm.Workspace,
		Password:  fm.Password,
	}

	// Remove known fields from inline settings
	delete(fm.Settings, "name")
	delete(fm.Settings, "form_id")
	delete(fm.Settings, "workspace")
	delete(fm.Settings, "password")
	delete(fm.Settings, "strip_prefix")
	if len(fm.Settings) > 0 {
		form.Settings = fm.Settings
	}

	// Resolve the regex used to strip the question-ID prefix from the displayed
	// text. Three cases:
	//   field absent  → default regex (current behavior: strip `F<n>:`)
	//   field == ""   → no stripping (prefix remains visible in Tally)
	//   custom regex  → compile and apply
	stripPrefix := DefaultStripPrefix
	if fm.StripPrefix != nil {
		if *fm.StripPrefix == "" {
			stripPrefix = nil
		} else {
			stripPrefix, err = regexp.Compile(*fm.StripPrefix)
			if err != nil {
				return nil, fmt.Errorf("strip_prefix regex: %w", err)
			}
		}
	}

	pages := splitPages(body)
	for _, pageContent := range pages {
		page, err := parsePage(pageContent, stripPrefix)
		if err != nil {
			return nil, err
		}
		form.Pages = append(form.Pages, *page)
	}

	if len(form.Pages) == 0 {
		form.Pages = []model.Page{{}}
	}

	return form, nil
}

func splitFrontmatter(content string) (*frontmatter, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return &frontmatter{}, content, nil
	}

	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return &frontmatter{}, content, nil
	}

	yamlContent := rest[:idx]
	body := rest[idx+4:] // skip \n---

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, "", fmt.Errorf("parse YAML: %w", err)
	}

	return &fm, body, nil
}

type pageRaw struct {
	buttonLabel string
	lines       []string
}

func splitPages(body string) []pageRaw {
	lines := strings.Split(body, "\n")
	var pages []pageRaw
	current := pageRaw{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			pages = append(pages, current)
			current = pageRaw{}
			continue
		}
		if m := buttonRe.FindStringSubmatch(trimmed); m != nil {
			current.buttonLabel = m[1]
			continue
		}
		current.lines = append(current.lines, line)
	}
	pages = append(pages, current)

	// Filter empty pages
	var result []pageRaw
	for _, p := range pages {
		hasContent := false
		for _, l := range p.lines {
			if strings.TrimSpace(l) != "" {
				hasContent = true
				break
			}
		}
		if hasContent || p.buttonLabel != "" {
			result = append(result, p)
		}
	}

	return result
}

func parsePage(raw pageRaw, stripPrefix *regexp.Regexp) (*model.Page, error) {
	page := &model.Page{
		ButtonLabel: raw.buttonLabel,
	}

	var currentQuestion *model.Question
	var pendingText []string
	addOther := false

	flushText := func() {
		text := strings.TrimSpace(strings.Join(pendingText, "\n"))
		if text != "" {
			page.Blocks = append(page.Blocks, &model.TextBlock{
				HTML: convertInlineMarkup(text),
			})
		}
		pendingText = nil
	}

	flushQuestion := func() {
		if currentQuestion != nil {
			if addOther {
				currentQuestion.Options = append(currentQuestion.Options, model.Option{
					Text:    "Andere",
					IsOther: true,
				})
				addOther = false
			}
			page.Blocks = append(page.Blocks, currentQuestion)
			currentQuestion = nil
		}
	}

	for _, line := range raw.lines {
		trimmed := strings.TrimSpace(line)

		// Empty line
		if trimmed == "" {
			if currentQuestion == nil && len(pendingText) > 0 {
				flushText()
			}
			continue
		}

		// Conditional
		if m := conditionalRe.FindStringSubmatch(trimmed); m != nil {
			flushQuestion()
			flushText()
			cond, err := parseConditional(m[1], m[2])
			if err != nil {
				return nil, err
			}
			page.Blocks = append(page.Blocks, cond)
			continue
		}

		// Heading
		if m := headingRe.FindStringSubmatch(trimmed); m != nil {
			flushQuestion()
			flushText()
			level := len(m[1])
			page.Blocks = append(page.Blocks, &model.HeadingBlock{
				Text:  m[2],
				Level: level,
			})
			continue
		}

		// Question start
		if m := questionRe.FindStringSubmatch(trimmed); m != nil {
			flushQuestion()
			flushText()
			id := "F" + m[1]
			// Displayed text is derived from the full line by applying the
			// configured strip_prefix regex. `nil` means "keep as-is" so the
			// `F<n>:` marker remains visible in Tally.
			text := trimmed
			if stripPrefix != nil {
				text = stripPrefix.ReplaceAllString(text, "")
			}
			text = strings.TrimSpace(text)
			currentQuestion = &model.Question{
				ID:         id,
				Text:       text,
				Required:   true, // default
				Properties: make(map[string]any),
			}
			addOther = false
			continue
		}

		// Metadata (> key: value)
		if m := metaRe.FindStringSubmatch(trimmed); m != nil {
			if currentQuestion != nil {
				applyMeta(currentQuestion, m[1], m[2], &addOther)
			}
			continue
		}

		// Option (- text)
		if m := optionRe.FindStringSubmatch(trimmed); m != nil {
			if currentQuestion != nil {
				optText := m[1]
				isOther := false
				if otherMarkerRe.MatchString(optText) {
					optText = otherMarkerRe.ReplaceAllString(optText, "")
					isOther = true
				}
				optText = convertInlineMarkup(strings.TrimSpace(optText))

				if currentQuestion.Type == model.Matrix {
					currentQuestion.MatrixRows = append(currentQuestion.MatrixRows, optText)
				} else {
					currentQuestion.Options = append(currentQuestion.Options, model.Option{
						Text:    optText,
						IsOther: isOther,
					})
				}
			}
			continue
		}

		// Regular text
		if currentQuestion == nil {
			pendingText = append(pendingText, trimmed)
		}
	}

	flushQuestion()
	flushText()

	return page, nil
}

func applyMeta(q *model.Question, key, value string, addOther *bool) {
	value = strings.TrimSpace(value)
	// Strip surrounding quotes
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	switch key {
	case "type":
		q.Type = model.QuestionType(value)
	case "required":
		q.Required = value == "true"
	case "hidden":
		q.Hidden = value == "true"
	case "hint":
		q.Hint = value
	case "placeholder":
		q.Placeholder = value
	case "other":
		if value == "true" {
			*addOther = true
		}
	case "columns":
		cols := strings.Split(value, ",")
		for i, c := range cols {
			cols[i] = strings.TrimSpace(c)
		}
		q.MatrixCols = cols
	case "max":
		if n, err := strconv.Atoi(value); err == nil {
			q.Properties["max"] = n
		}
	case "min":
		if n, err := strconv.Atoi(value); err == nil {
			q.Properties["min"] = n
		}
	case "stars":
		if n, err := strconv.Atoi(value); err == nil {
			q.Properties["stars"] = n
		}
	case "start":
		if n, err := strconv.Atoi(value); err == nil {
			q.Properties["start"] = n
		}
	case "end":
		if n, err := strconv.Atoi(value); err == nil {
			q.Properties["end"] = n
		}
	case "step":
		if n, err := strconv.Atoi(value); err == nil {
			q.Properties["step"] = n
		}
	case "left-label":
		q.Properties["left-label"] = value
	case "right-label":
		q.Properties["right-label"] = value
	}
}

func parseConditional(targetStr, conditionStr string) (*model.Conditional, error) {
	// Parse targets: "F3, F4"
	targetParts := strings.Split(targetStr, ",")
	var targets []string
	for _, t := range targetParts {
		targets = append(targets, strings.TrimSpace(t))
	}

	// Split conditions by " and " or " or "
	operator := "AND"
	var condParts []string

	// Try splitting by " and " first, then " or "
	if strings.Contains(conditionStr, " and ") {
		condParts = splitConditions(conditionStr, " and ")
		operator = "AND"
	} else if strings.Contains(conditionStr, " or ") {
		condParts = splitConditions(conditionStr, " or ")
		operator = "OR"
	} else {
		condParts = []string{conditionStr}
	}

	var conditions []model.Condition
	for _, part := range condParts {
		c, err := parseSingleCondition(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("condition %q: %w", part, err)
		}
		conditions = append(conditions, *c)
	}

	return &model.Conditional{
		Targets:    targets,
		Conditions: conditions,
		Operator:   operator,
	}, nil
}

func splitConditions(s, sep string) []string {
	var parts []string
	for {
		idx := strings.Index(s, sep)
		if idx == -1 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+len(sep):]
	}
	return parts
}

// parseSingleCondition parses "F2 is_not_any_of "a", "b"" or "F2 is_not_empty"
func parseSingleCondition(s string) (*model.Condition, error) {
	// Format: <field> <operator> [<values>]
	parts := strings.SplitN(s, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid condition: %s", s)
	}

	field := parts[0]
	comparison := parts[1]
	var values []string

	if len(parts) > 2 {
		values = parseQuotedValues(parts[2])
	}

	return &model.Condition{
		Field:      field,
		Comparison: comparison,
		Values:     values,
	}, nil
}

// parseQuotedValues parses: "Value 1", "Value 2" or "Single value"
func parseQuotedValues(s string) []string {
	s = strings.TrimSpace(s)
	var values []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			if inQuote {
				values = append(values, current.String())
				current.Reset()
				inQuote = false
			} else {
				inQuote = true
			}
		} else if inQuote {
			current.WriteByte(ch)
		}
	}

	// Handle unquoted single value
	if len(values) == 0 && s != "" {
		values = append(values, s)
	}

	return values
}

func convertInlineMarkup(s string) string {
	// Bold before italic — **bold** must be processed first
	s = boldRe.ReplaceAllString(s, "<b>$1</b>")
	s = italicRe.ReplaceAllString(s, "<i>$1</i>")
	s = linkRe.ReplaceAllString(s, `<a href="$2">$1</a>`)
	return s
}
