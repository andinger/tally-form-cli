package tally

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
type SubmissionQuestion struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// Submission represents a single form submission.
type Submission struct {
	ID          string               `json:"id"`
	SubmittedAt string               `json:"submittedAt"`
	Responses   []SubmissionResponse `json:"responses"`
}

// SubmissionResponse is a single answer within a submission.
type SubmissionResponse struct {
	QuestionID      string `json:"questionId"`
	FormattedAnswer string `json:"formattedAnswer"`
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

// parseHTMLToSchema splits HTML with <b>/<i> tags into safeHTMLSchema segments.
func parseHTMLToSchema(s string) []any {
	var segments []any
	for len(s) > 0 {
		// Find the next tag
		boldIdx := indexOf(s, "<b>")
		italicIdx := indexOf(s, "<i>")

		// Pick the nearest tag
		nextIdx := -1
		var openTag, closeTag string
		var styles []any
		if boldIdx >= 0 && (italicIdx < 0 || boldIdx < italicIdx) {
			nextIdx = boldIdx
			openTag = "<b>"
			closeTag = "</b>"
			styles = []any{[]any{"tag", "span"}, []any{"font-weight", "bold"}}
		} else if italicIdx >= 0 {
			nextIdx = italicIdx
			openTag = "<i>"
			closeTag = "</i>"
			styles = []any{[]any{"tag", "span"}, []any{"font-style", "italic"}}
		}

		if nextIdx < 0 {
			// No more tags — rest is plain text
			if s != "" {
				segments = append(segments, []any{s})
			}
			break
		}

		// Text before the tag
		if nextIdx > 0 {
			segments = append(segments, []any{s[:nextIdx]})
		}

		// Find closing tag
		inner := s[nextIdx+len(openTag):]
		closeIdx := indexOf(inner, closeTag)
		if closeIdx < 0 {
			// Unclosed tag — treat rest as plain text
			segments = append(segments, []any{s[nextIdx:]})
			break
		}

		// Styled segment
		segments = append(segments, []any{inner[:closeIdx], styles})
		s = inner[closeIdx+len(closeTag):]
	}

	if len(segments) == 0 {
		return []any{[]any{""}}
	}
	return segments
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
