package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/andinger/tally-form-cli/internal/tally"
)

// writeQuestionMarkdownFiles writes one Markdown file per question in the
// form into dir. Filename pattern is "<NN>-<slug>.md" where NN is zero-padded
// to fit the form's question count (min width 2) and slug is the lowercased
// ASCII transliteration of the question label, capped at 25 chars at a word
// boundary. When the slug would be empty (label was unicode-only or empty),
// the question ID is used as fallback.
//
// Each file contains rich YAML frontmatter (form ID, question metadata,
// answer-statistics) followed by an H1 with the question title and one H2
// per submission. All submissions appear in every file in stable API order
// — questions a submission did not answer get an "_(keine Antwort)_"
// placeholder so a given submission ID always sits at the same offset across
// files (cross-file alignment).
func writeQuestionMarkdownFiles(formID, dir string, subs *tally.SubmissionsResponse, form *tally.TallyForm, rowLabels map[string]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	var blocks []tally.TallyBlock
	if form != nil {
		blocks = form.Blocks
	}

	padWidth := digitWidth(len(subs.Questions))

	for i, q := range subs.Questions {
		cfg := extractQuestionConfig(q, blocks)
		filename := questionFilename(i+1, padWidth, q)
		md := buildQuestionMarkdown(formID, q, cfg, subs.Submissions, rowLabels, len(subs.Submissions))
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	fmt.Fprintf(os.Stderr, "Wrote %d question file(s) to %s\n", len(subs.Questions), dir)
	return nil
}

// questionHeading returns the human-readable text to use for a question's
// H1 in the per-question Markdown file. Unlike questionLabel (used for CSV
// columns and slugs, which prefers the short Name), the H1 wants the full
// Title — that is the prompt the respondent saw. Name is a sensible fallback
// because some forms have only a Name; ID is the last-ditch option.
func questionHeading(q tally.SubmissionQuestion) string {
	if q.Title != "" {
		return q.Title
	}
	if q.Name != "" {
		return q.Name
	}
	return q.ID
}

// digitWidth returns the number of digits needed to represent n, with a
// minimum of 2. This makes index prefixes sort correctly in directory
// listings even with up to 99 questions while still scaling to 100+.
func digitWidth(n int) int {
	w := 1
	for n >= 10 {
		n /= 10
		w++
	}
	if w < 2 {
		w = 2
	}
	return w
}

// questionFilename builds "<NN>-<slug>.md" for the given 1-based question
// index. Falls back to the question ID when no usable slug can be derived
// from Name or Title.
func questionFilename(index, width int, q tally.SubmissionQuestion) string {
	slug := slugify(questionLabel(q), 25)
	if slug == "" {
		slug = q.ID
	}
	return fmt.Sprintf("%0*d-%s.md", width, index, slug)
}

// questionFrontmatter is the typed shape we marshal into YAML for each
// question file. Optional fields use pointers / omitempty so unused config
// (e.g. options on a textarea) does not produce empty keys.
type questionFrontmatter struct {
	FormID          string                `yaml:"form_id"`
	QuestionID      string                `yaml:"question_id"`
	Type            string                `yaml:"type"`
	Title           string                `yaml:"title,omitempty"`
	Name            string                `yaml:"name,omitempty"`
	IsRequired      *bool                 `yaml:"is_required,omitempty"`
	Placeholder     string                `yaml:"placeholder,omitempty"`
	Options         []frontmatterOption   `yaml:"options,omitempty"`
	MatrixColumns   []string              `yaml:"matrix_columns,omitempty"`
	MatrixRows      []frontmatterOption   `yaml:"matrix_rows,omitempty"`
	ScaleStart      *int                  `yaml:"scale_start,omitempty"`
	ScaleEnd        *int                  `yaml:"scale_end,omitempty"`
	ScaleStep       *int                  `yaml:"scale_step,omitempty"`
	ScaleLeftLabel  string                `yaml:"scale_left_label,omitempty"`
	ScaleRightLabel string                `yaml:"scale_right_label,omitempty"`
	Stars           *int                  `yaml:"stars,omitempty"`
	NumResponses    int                   `yaml:"num_responses"`
	NumSubmissions  int                   `yaml:"num_submissions"`
}

type frontmatterOption struct {
	UUID  string `yaml:"uuid"`
	Label string `yaml:"label"`
}

// buildQuestionFrontmatter assembles the typed frontmatter struct from a
// question, its config, and the submission counts. Returns the YAML-serialized
// frontmatter (without the leading/trailing "---" markers — buildQuestionMarkdown
// adds those).
func buildQuestionFrontmatter(formID string, q tally.SubmissionQuestion, cfg questionConfig, numResponses, numSubmissions int) (string, error) {
	fm := questionFrontmatter{
		FormID:          formID,
		QuestionID:      q.ID,
		Type:            q.Type,
		Title:           q.Title,
		Name:            q.Name,
		Placeholder:     cfg.Placeholder,
		MatrixColumns:   cfg.MatrixColumns,
		ScaleStart:      cfg.ScaleStart,
		ScaleEnd:        cfg.ScaleEnd,
		ScaleStep:       cfg.ScaleStep,
		ScaleLeftLabel:  cfg.ScaleLeft,
		ScaleRightLabel: cfg.ScaleRight,
		Stars:           cfg.Stars,
		NumResponses:    numResponses,
		NumSubmissions:  numSubmissions,
	}
	// is_required only makes sense when we actually saw a block — use the
	// config presence (any options, placeholder, scale, matrix, ...) as a
	// proxy for "form data was available."
	if hasFormData(cfg) {
		v := cfg.IsRequired
		fm.IsRequired = &v
	}
	for _, o := range cfg.Options {
		fm.Options = append(fm.Options, frontmatterOption{UUID: o.UUID, Label: o.Label})
	}
	for _, r := range cfg.MatrixRows {
		fm.MatrixRows = append(fm.MatrixRows, frontmatterOption{UUID: r.UUID, Label: r.Label})
	}

	out, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}
	return string(out), nil
}

// hasFormData reports whether extractQuestionConfig found any structural
// information for the question. Without it (e.g. when GetForm failed), we
// must omit is_required entirely rather than asserting "false."
func hasFormData(cfg questionConfig) bool {
	return cfg.Placeholder != "" ||
		len(cfg.Options) > 0 ||
		len(cfg.MatrixRows) > 0 ||
		len(cfg.MatrixColumns) > 0 ||
		cfg.ScaleStart != nil || cfg.ScaleEnd != nil ||
		cfg.Stars != nil ||
		cfg.IsRequired // explicit true also counts
}

// buildQuestionMarkdown renders the full Markdown file for one question:
// frontmatter, H1 with question title, and one H2 + answer per submission.
//
// All submissions are rendered, including those that did not answer this
// question — the placeholder "_(keine Antwort)_" keeps cross-file alignment
// (the Nth submission sits at the same offset in every file).
func buildQuestionMarkdown(formID string, q tally.SubmissionQuestion, cfg questionConfig, submissions []tally.Submission, rowLabels map[string]string, numSubmissions int) string {
	numResponses := countResponses(q.ID, submissions)
	fm, err := buildQuestionFrontmatter(formID, q, cfg, numResponses, numSubmissions)
	if err != nil {
		// Marshal failure is not recoverable per-file; surface a synthetic
		// frontmatter so the file at least carries the identifiers.
		fm = fmt.Sprintf("form_id: %q\nquestion_id: %q\ntype: %q\nerror: %q\n", formID, q.ID, q.Type, err.Error())
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fm)
	b.WriteString("---\n\n")

	b.WriteString("# ")
	b.WriteString(questionHeading(q))
	b.WriteString("\n\n")

	for _, sub := range submissions {
		b.WriteString("## ")
		b.WriteString(sub.ID)
		b.WriteString("\n\n")
		b.WriteString(answerForSubmission(sub, q, rowLabels))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// countResponses returns the number of submissions that answered the given
// question — used for the num_responses statistic in the frontmatter.
func countResponses(qID string, submissions []tally.Submission) int {
	n := 0
	for _, sub := range submissions {
		for _, r := range sub.Responses {
			if r.QuestionID == qID && r.Answer != nil {
				n++
				break
			}
		}
	}
	return n
}

// answerForSubmission returns the Markdown rendering of submission sub's
// answer to question q, with TEXTAREA / INPUT_TEXT free-text answers wrapped
// in a fenced code block to neutralize Markdown injection. Returns the
// "no answer" placeholder when the submission did not respond to q.
func answerForSubmission(sub tally.Submission, q tally.SubmissionQuestion, rowLabels map[string]string) string {
	for _, r := range sub.Responses {
		if r.QuestionID != q.ID {
			continue
		}
		if r.Answer == nil && r.FormattedAnswer == "" {
			return "_(keine Antwort)_"
		}
		// Free-text inputs are the only types where a respondent can break
		// the surrounding Markdown structure; wrap them in a code fence.
		if (q.Type == "TEXTAREA" || q.Type == "INPUT_TEXT") && r.Answer != nil {
			if s, ok := r.Answer.(string); ok {
				if s == "" {
					return "_(keine Antwort)_"
				}
				return codeFence(s)
			}
		}
		return markdownAnswer(r, q.Type, rowLabels)
	}
	return "_(keine Antwort)_"
}
