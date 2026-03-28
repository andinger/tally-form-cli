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

// SafeHTMLSchemaFromHTML builds safeHTMLSchema for text that may contain <i> tags.
// For simplicity, we use the plain text in the schema.
func SafeHTMLSchemaFromHTML(html string) []any {
	return []any{[]any{html}}
}
