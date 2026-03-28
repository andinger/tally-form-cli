package model

// QuestionType represents the type of a question in the form.
type QuestionType string

const (
	SingleChoice QuestionType = "single-choice"
	MultiChoice  QuestionType = "multi-choice"
	Dropdown     QuestionType = "dropdown"
	LongText     QuestionType = "long-text"
	ShortText    QuestionType = "short-text"
	Number       QuestionType = "number"
	Email        QuestionType = "email"
	Phone        QuestionType = "phone"
	URL          QuestionType = "url"
	Date         QuestionType = "date"
	Time         QuestionType = "time"
	Rating       QuestionType = "rating"
	Scale        QuestionType = "scale"
	Matrix       QuestionType = "matrix"
	FileUpload   QuestionType = "file-upload"
	Signature    QuestionType = "signature"
)

// Block is the interface all form blocks implement.
type Block interface {
	BlockType() string
}

// Form is the intermediate representation of a Tally form.
type Form struct {
	Name      string
	FormID    string
	Workspace string
	Password  string
	Settings  map[string]any
	Pages     []Page
}

// Page represents a single page in a multi-page form.
type Page struct {
	ButtonLabel string
	Blocks      []Block
}

// HeadingBlock represents a heading (## in Markdown).
type HeadingBlock struct {
	Text  string
	Level int // 1 or 2
}

func (h *HeadingBlock) BlockType() string { return "heading" }

// TextBlock represents a plain text paragraph.
type TextBlock struct {
	HTML string
}

func (t *TextBlock) BlockType() string { return "text" }

// Option represents a single choice option in a question.
type Option struct {
	Text    string
	IsOther bool
}

// Question represents a form question with its metadata.
type Question struct {
	ID          string
	Text        string
	Type        QuestionType
	Required    bool
	Hidden      bool
	Hint        string
	Placeholder string
	Options     []Option
	MatrixCols  []string
	MatrixRows  []string
	Properties  map[string]any
}

func (q *Question) BlockType() string { return "question" }

// Conditional represents conditional visibility logic.
type Conditional struct {
	Targets    []string // e.g., ["F3", "F4"]
	Conditions []Condition
	Operator   string // "AND" or "OR"
}

func (c *Conditional) BlockType() string { return "conditional" }

// Condition is a single condition within a Conditional.
type Condition struct {
	Field      string   // e.g., "F2"
	Comparison string   // e.g., "is_not_any_of"
	Values     []string // e.g., ["Option A", "Option B"]
}
