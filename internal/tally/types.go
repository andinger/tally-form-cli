package tally

import "strings"

// TallyBlock represents a single block in the Tally API.
type TallyBlock struct {
	UUID      string         `json:"uuid"`
	Type      string         `json:"type"`
	GroupUUID string         `json:"groupUuid"`
	GroupType string         `json:"groupType"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// CreateFormRequest is the request body for POST /forms.
type CreateFormRequest struct {
	WorkspaceID string       `json:"workspaceId"`
	Name        string       `json:"name,omitempty"`
	Status      string       `json:"status,omitempty"`
	Password    string       `json:"password,omitempty"`
	Settings    any          `json:"settings,omitempty"`
	Blocks      []TallyBlock `json:"blocks"`
}

// UpdateFormRequest is the request body for PATCH /forms/{id}.
type UpdateFormRequest = CreateFormRequest

// TallyForm is the response from GET /forms/{id}.
type TallyForm struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Status   string         `json:"status"`
	Settings map[string]any `json:"settings"`
	Blocks   []TallyBlock   `json:"blocks"`
}

// SubmissionsResponse is the response from GET /forms/{id}/submissions.
type SubmissionsResponse struct {
	Questions []SubmissionQuestion `json:"questions"`
	Page      int                  `json:"page"`
	Limit     int                  `json:"limit"`
	HasMore   bool                 `json:"hasMore"`
	TotalNumberOfSubmissions int  `json:"totalNumberOfSubmissions"`
	Submissions []Submission       `json:"submissions"`
}

// SubmissionQuestion describes a form question in submission context.
// For matrix questions, Fields contains one entry per row with the row label
// embedded in the title (e.g. "Question text [Row label]"); Answer under the
// matrix QuestionID is keyed by field.blockGroupUuid.
type SubmissionQuestion struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Title  string                 `json:"title,omitempty"`
	Name   string                 `json:"name,omitempty"`
	Fields []SubmissionField      `json:"fields,omitempty"`
}

// SubmissionField is a single input field within a question (usually one per
// question; for matrix questions one per row).
type SubmissionField struct {
	UUID           string `json:"uuid"`
	Type           string `json:"type"`
	QuestionType   string `json:"questionType"`
	BlockGroupUUID string `json:"blockGroupUuid"`
	Title          string `json:"title,omitempty"`
}

// Submission represents a single form submission.
type Submission struct {
	ID          string               `json:"id"`
	SubmittedAt string               `json:"submittedAt"`
	IsCompleted bool                 `json:"isCompleted,omitempty"`
	Responses   []SubmissionResponse `json:"responses"`
}

// SubmissionResponse is a single answer within a submission.
// Answer holds the raw value as returned by the Tally API. It can be:
//   - string (short-text, email, date, time, URL)
//   - number (rating, scale, input-number)
//   - []string (choice / checkbox / dropdown — always as an array)
//   - []map[string]any (file-upload, signature — each item has id, name, url, mimeType, size)
//   - map[string]any (matrix — keys are row UUIDs, values are arrays of selected column labels)
// FormattedAnswer is Tally's pre-formatted string (often empty when the list
// endpoint does not compute it), so consumers should prefer Answer.
type SubmissionResponse struct {
	QuestionID      string `json:"questionId"`
	Answer          any    `json:"answer,omitempty"`
	FormattedAnswer string `json:"formattedAnswer,omitempty"`
}

// SafeHTMLSchema builds the safeHTMLSchema array for plain text.
func SafeHTMLSchema(text string) []any {
	return []any{[]any{text}}
}

// SafeHTMLSchemaFromHTML builds safeHTMLSchema for text that may contain <b> and <i> tags.
// It produces the structured segment format that Tally expects:
//
//	[["plain"], ["bold", [["tag","span"],["font-weight","bold"]]], ...]
func SafeHTMLSchemaFromHTML(html string) []any {
	return parseHTMLToSchema(html)
}

// parseHTMLToSchema splits HTML with <b>/<i>/<a> tags into safeHTMLSchema segments.
func parseHTMLToSchema(s string) []any {
	var segments []any
	for len(s) > 0 {
		// Find the next tag
		boldIdx := indexOf(s, "<b>")
		italicIdx := indexOf(s, "<i>")
		linkIdx := indexOf(s, "<a href=\"")

		// Pick the nearest tag
		type tagMatch struct {
			idx      int
			openEnd  string // closing delimiter of the open tag (e.g. ">" or "">")
			closeTag string
			styles   func(s string) []any
		}

		candidates := []tagMatch{}
		if boldIdx >= 0 {
			candidates = append(candidates, tagMatch{
				idx: boldIdx, closeTag: "</b>",
				styles: func(string) []any {
					return []any{[]any{"tag", "span"}, []any{"font-weight", "bold"}}
				},
			})
		}
		if italicIdx >= 0 {
			candidates = append(candidates, tagMatch{
				idx: italicIdx, closeTag: "</i>",
				styles: func(string) []any {
					return []any{[]any{"tag", "span"}, []any{"font-style", "italic"}}
				},
			})
		}
		if linkIdx >= 0 {
			candidates = append(candidates, tagMatch{
				idx: linkIdx, closeTag: "</a>",
				styles: func(openTag string) []any {
					// Extract href from <a href="url">
					href := extractHref(openTag)
					return []any{[]any{"href", href}}
				},
			})
		}

		if len(candidates) == 0 {
			if s != "" {
				segments = append(segments, []any{s})
			}
			break
		}

		// Pick the earliest tag
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.idx < best.idx {
				best = c
			}
		}

		// Text before the tag
		if best.idx > 0 {
			segments = append(segments, []any{s[:best.idx]})
		}

		// Find the end of the opening tag (">")
		afterOpen := s[best.idx:]
		gtIdx := indexOf(afterOpen, ">")
		if gtIdx < 0 {
			segments = append(segments, []any{s[best.idx:]})
			break
		}
		openTag := afterOpen[:gtIdx+1]
		inner := afterOpen[gtIdx+1:]

		// Find closing tag
		closeIdx := indexOf(inner, best.closeTag)
		if closeIdx < 0 {
			segments = append(segments, []any{s[best.idx:]})
			break
		}

		// Styled segment — check for nested tags inside
		innerText := inner[:closeIdx]
		styles := best.styles(openTag)
		if containsHTMLTag(innerText) {
			// Recursively parse inner content and apply styles to each sub-segment
			subSegments := parseHTMLToSchema(innerText)
			for _, sub := range subSegments {
				subArr := sub.([]any)
				if len(subArr) == 1 {
					// Plain text sub-segment — apply parent styles
					segments = append(segments, []any{subArr[0], styles})
				} else {
					// Already styled sub-segment — keep its own styles (don't nest)
					segments = append(segments, sub)
				}
			}
		} else {
			segments = append(segments, []any{innerText, styles})
		}
		s = inner[closeIdx+len(best.closeTag):]
	}

	if len(segments) == 0 {
		return []any{[]any{""}}
	}
	return segments
}

// containsHTMLTag checks if a string contains any HTML-like tags.
func containsHTMLTag(s string) bool {
	return indexOf(s, "<") >= 0 && indexOf(s, ">") >= 0
}

// extractHref extracts the URL from an opening <a href="url"> tag.
func extractHref(openTag string) string {
	prefix := `<a href="`
	if !strings.HasPrefix(openTag, prefix) {
		return ""
	}
	rest := openTag[len(prefix):]
	endQuote := indexOf(rest, `"`)
	if endQuote < 0 {
		return rest
	}
	return rest[:endQuote]
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
