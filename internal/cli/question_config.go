package cli

import (
	"github.com/andinger/tally-form-cli/internal/tally"
)

// questionConfig captures the per-question configuration that we surface in
// the rich frontmatter for `--group-by question`. Fields are populated only
// when present in the form's blocks — the YAML renderer omits zero-valued
// fields so dropdowns don't get a stray placeholder line, etc.
type questionConfig struct {
	IsRequired    bool
	Placeholder   string
	Options       []optionEntry // MULTIPLE_CHOICE / CHECKBOXES / DROPDOWN
	MatrixRows    []optionEntry // MATRIX rows (uuid → label)
	MatrixColumns []string      // MATRIX column labels
	ScaleStart    *int          // LINEAR_SCALE
	ScaleEnd      *int
	ScaleStep     *int
	ScaleLeft     string
	ScaleRight    string
	Stars         *int // RATING
}

type optionEntry struct {
	UUID  string
	Label string
}

// extractQuestionConfig walks the form's blocks and returns the configuration
// for the question identified by q. It uses q.Fields[].BlockGroupUUID as the
// anchor: for non-matrix questions, that is the content group's groupUuid; for
// matrix questions, the BlockGroupUUID is the row block's UUID and we resolve
// the surrounding matrix blocks from the row's groupUuid.
//
// Returns a zero-valued config when no matching blocks are found (e.g. when
// the form is not available) — callers can still write a frontmatter without
// the config fields.
func extractQuestionConfig(q tally.SubmissionQuestion, blocks []tally.TallyBlock) questionConfig {
	cfg := questionConfig{}
	if len(q.Fields) == 0 {
		return cfg
	}

	uuidIdx := make(map[string]tally.TallyBlock, len(blocks))
	groupIdx := make(map[string][]tally.TallyBlock, len(blocks))
	for _, b := range blocks {
		uuidIdx[b.UUID] = b
		groupIdx[b.GroupUUID] = append(groupIdx[b.GroupUUID], b)
	}

	if q.Type == "MATRIX" {
		// Resolve the matrix's content groupUuid via any row's block.
		var matrixGroup string
		for _, f := range q.Fields {
			if rowBlock, ok := uuidIdx[f.BlockGroupUUID]; ok {
				matrixGroup = rowBlock.GroupUUID
				break
			}
		}
		if matrixGroup == "" {
			return cfg
		}
		for _, b := range groupIdx[matrixGroup] {
			switch b.Type {
			case "MATRIX":
				if v, ok := b.Payload["isRequired"].(bool); ok {
					cfg.IsRequired = v
				}
			case "MATRIX_COLUMN":
				if label := extractSafeHTML(b.Payload["safeHTMLSchema"]); label != "" {
					cfg.MatrixColumns = append(cfg.MatrixColumns, label)
				}
			case "MATRIX_ROW":
				label := extractSafeHTML(b.Payload["safeHTMLSchema"])
				cfg.MatrixRows = append(cfg.MatrixRows, optionEntry{UUID: b.UUID, Label: label})
			}
		}
		return cfg
	}

	// Non-matrix: walk all content blocks sharing the field's groupUuid.
	contentGroup := q.Fields[0].BlockGroupUUID
	for _, b := range groupIdx[contentGroup] {
		// is_required is set on the first block we encounter; all blocks in
		// a question carry the same value.
		if v, ok := b.Payload["isRequired"].(bool); ok {
			cfg.IsRequired = v
		}
		switch b.Type {
		case "MULTIPLE_CHOICE_OPTION", "CHECKBOX", "DROPDOWN_OPTION":
			label, _ := b.Payload["text"].(string)
			cfg.Options = append(cfg.Options, optionEntry{UUID: b.UUID, Label: label})
		case "TEXTAREA", "INPUT_TEXT", "INPUT_EMAIL", "INPUT_NUMBER",
			"INPUT_PHONE_NUMBER", "INPUT_LINK", "INPUT_DATE", "INPUT_TIME":
			if v, ok := b.Payload["placeholder"].(string); ok {
				cfg.Placeholder = v
			}
		case "LINEAR_SCALE":
			cfg.ScaleStart = intPayload(b.Payload, "start")
			cfg.ScaleEnd = intPayload(b.Payload, "end")
			cfg.ScaleStep = intPayload(b.Payload, "step")
			if v, ok := b.Payload["leftLabel"].(string); ok {
				cfg.ScaleLeft = v
			}
			if v, ok := b.Payload["rightLabel"].(string); ok {
				cfg.ScaleRight = v
			}
		case "RATING":
			cfg.Stars = intPayload(b.Payload, "stars")
		}
	}

	return cfg
}

// intPayload reads a numeric payload field as *int. JSON numbers decode as
// float64, so we accept both float64 and int. Returns nil when the key is
// absent or holds a non-numeric value, so the renderer can omit the field.
func intPayload(payload map[string]any, key string) *int {
	v, ok := payload[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case int:
		return &n
	}
	return nil
}
