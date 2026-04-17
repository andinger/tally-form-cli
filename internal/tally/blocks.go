package tally

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/model"
)

// textInputTypes are input question types where a hint can be rendered as the
// native placeholder attribute. For choice/matrix/scale/rating questions, hints
// have no equivalent Tally representation and are silently dropped with a
// warning.
func hintSupportsPlaceholder(t model.QuestionType) bool {
	switch t {
	case model.LongText, model.ShortText, model.Number, model.Email,
		model.Phone, model.URL, model.Date, model.Time:
		return true
	}
	return false
}

type deferredCond struct {
	cond     *model.Conditional
	insertAt int
}

// Compiler converts an IR Form into Tally API blocks.
type Compiler struct {
	NewUUID func() string

	// Registry maps F<n> IDs to their generated UUIDs
	questionGroupUUIDs map[string]string             // F1 → groupUuid of the TITLE block
	contentGroupUUIDs  map[string]string             // F1 → groupUuid of the content blocks (options/inputs)
	questionBlockUUIDs map[string][]string           // F1 → all block UUIDs belonging to that question
	optionUUIDs        map[string]map[string]string  // F1 → { "Option Text" → option block UUID }
	firstOptionUUID    map[string]string             // F1 → UUID of the first option/input block
	questionTexts      map[string]string             // F1 → question text (for conditional field.title)
	questionTypes      map[string]model.QuestionType // F1 → question type
}

// NewCompiler creates a compiler with random UUID generation.
func NewCompiler() *Compiler {
	return &Compiler{
		NewUUID: func() string { return uuid.New().String() },
	}
}

// Compile converts an IR Form to a Tally API request.
func (c *Compiler) Compile(form *model.Form, cfg *config.Merged) (*CreateFormRequest, error) {
	c.questionGroupUUIDs = make(map[string]string)
	c.contentGroupUUIDs = make(map[string]string)
	c.questionBlockUUIDs = make(map[string][]string)
	c.optionUUIDs = make(map[string]map[string]string)
	c.firstOptionUUID = make(map[string]string)
	c.questionTexts = make(map[string]string)
	c.questionTypes = make(map[string]model.QuestionType)

	var blocks []TallyBlock

	// First pass: register all questions across all pages (needed for conditional references)
	for _, page := range form.Pages {
		for _, block := range page.Blocks {
			if q, ok := block.(*model.Question); ok {
				c.registerQuestion(q)
			}
		}
	}

	// Apply frontmatter logo override (form.Settings flows from YAML inline fields)
	if form.Settings != nil && cfg != nil {
		if logo, ok := form.Settings["logo"].(string); ok && logo != "" {
			cfg.Logo = logo
		}
	}

	// FORM_TITLE block
	titleBlock := c.buildFormTitle(form.Name, cfg)
	blocks = append(blocks, titleBlock)

	// Second pass: compile all blocks, collecting deferred conditionals
	var deferredConditionals []*deferredCond

	lastPage := len(form.Pages) - 1
	// Check if the last page is a thank-you page (text only, no questions).
	// If so, extract it into closeMessageTitle/closeMessageDescription settings
	// instead of emitting it as blocks.
	hasThankYouPage := lastPage > 0 && !pageHasQuestions(form.Pages[lastPage])
	pagesToCompile := form.Pages
	if hasThankYouPage {
		pagesToCompile = form.Pages[:lastPage]
	}

	for i, page := range pagesToCompile {
		if i > 0 {
			pb := c.buildPageBreak(page.ButtonLabel, i-1, i == len(pagesToCompile)-1)
			blocks = append(blocks, pb)
		}

		for _, block := range page.Blocks {
			if cond, ok := block.(*model.Conditional); ok {
				// Defer conditionals until all questions are compiled
				deferredConditionals = append(deferredConditionals, &deferredCond{
					cond:     cond,
					insertAt: len(blocks), // will insert at current position
				})
				// Reserve a slot
				blocks = append(blocks, TallyBlock{})
				continue
			}
			compiled, err := c.compileBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, compiled...)
		}
	}

	// Third pass: compile deferred conditionals (now all questions are registered)
	for _, dc := range deferredConditionals {
		compiled, err := c.compileConditional(dc.cond)
		if err != nil {
			return nil, err
		}
		blocks[dc.insertAt] = compiled[0]
	}

	// Build settings (including thank-you page content if present)
	settings := c.buildSettings(form, cfg)
	if hasThankYouPage {
		applyThankYouPage(form.Pages[lastPage], settings)
	}

	req := &CreateFormRequest{
		WorkspaceID: cfg.Workspace,
		Name:        form.Name,
		Status:      "PUBLISHED",
		Blocks:      blocks,
		Settings:    settings,
	}

	if form.Password != "" {
		req.Password = form.Password
	}

	return req, nil
}

func (c *Compiler) registerQuestion(q *model.Question) {
	groupUUID := c.NewUUID()
	c.questionGroupUUIDs[q.ID] = groupUUID
	c.optionUUIDs[q.ID] = make(map[string]string)
	c.questionTexts[q.ID] = q.Text
	c.questionTypes[q.ID] = q.Type
}

func (c *Compiler) buildFormTitle(name string, cfg *config.Merged) TallyBlock {
	payload := map[string]any{
		"safeHTMLSchema": SafeHTMLSchema(name),
		"title":          name,
		"button": map[string]any{
			"label": "Weiter",
		},
	}
	if cfg != nil && cfg.Logo != "" {
		payload["logo"] = cfg.Logo
	}

	return TallyBlock{
		UUID:      c.NewUUID(),
		Type:      "FORM_TITLE",
		GroupUUID: c.NewUUID(),
		GroupType: "TEXT",
		Payload:   payload,
	}
}

func (c *Compiler) buildPageBreak(buttonLabel string, index int, isLast bool) TallyBlock {
	if buttonLabel == "" {
		buttonLabel = "Weiter"
	}
	return TallyBlock{
		UUID:      c.NewUUID(),
		Type:      "PAGE_BREAK",
		GroupUUID: c.NewUUID(),
		GroupType: "PAGE_BREAK",
		Payload: map[string]any{
			"index":   index,
			"isFirst": index == 0,
			"isLast":  isLast,
			"button": map[string]any{
				"label": buttonLabel,
			},
		},
	}
}

func (c *Compiler) compileBlock(block model.Block) ([]TallyBlock, error) {
	switch b := block.(type) {
	case *model.HeadingBlock:
		return c.compileHeading(b), nil
	case *model.TextBlock:
		return c.compileText(b), nil
	case *model.Question:
		return c.compileQuestion(b), nil
	case *model.Conditional:
		return c.compileConditional(b)
	default:
		return nil, fmt.Errorf("unknown block type: %T", block)
	}
}

func (c *Compiler) compileHeading(h *model.HeadingBlock) []TallyBlock {
	hType := "HEADING_2"
	if h.Level == 1 {
		hType = "HEADING_1"
	}
	return []TallyBlock{{
		UUID:      c.NewUUID(),
		Type:      hType,
		GroupUUID: c.NewUUID(),
		GroupType: hType,
		Payload: map[string]any{
			"safeHTMLSchema": SafeHTMLSchema(h.Text),
		},
	}}
}

func (c *Compiler) compileText(t *model.TextBlock) []TallyBlock {
	return []TallyBlock{{
		UUID:      c.NewUUID(),
		Type:      "TEXT",
		GroupUUID: c.NewUUID(),
		GroupType: "TEXT",
		Payload: map[string]any{
			"safeHTMLSchema": SafeHTMLSchemaFromHTML(t.HTML),
		},
	}}
}

func (c *Compiler) compileQuestion(q *model.Question) []TallyBlock {
	groupUUID := c.questionGroupUUIDs[q.ID]
	var blocks []TallyBlock

	// TITLE block
	titleUUID := c.NewUUID()
	titlePayload := map[string]any{
		"safeHTMLSchema": SafeHTMLSchema(q.Text),
	}

	titleBlock := TallyBlock{
		UUID:      titleUUID,
		Type:      "TITLE",
		GroupUUID: groupUUID,
		GroupType: "QUESTION",
		Payload:   titlePayload,
	}
	if q.Hidden {
		titleBlock.Payload["isHidden"] = true
	}
	blocks = append(blocks, titleBlock)
	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], titleUUID)

	// Hint handling: Tally forms produced by the editor do NOT emit a separate
	// TEXT block between TITLE and the input — such an orphan block breaks the
	// question group, causing the editor to treat the question as a fragment
	// (e.g. deleting a conditional that references such a question cascades
	// into deleting the question itself). Instead, hints are folded into the
	// input's `placeholder` for text-like inputs. For choice/matrix/scale
	// questions there is no clean placeholder equivalent, so we warn and drop.
	if q.Hint != "" {
		if hintSupportsPlaceholder(q.Type) {
			if q.Placeholder == "" {
				q.Placeholder = q.Hint
			}
		} else {
			fmt.Fprintf(os.Stderr,
				"Warning: question %s (%s) has a hint but its type does not support placeholders — hint is dropped. Consider embedding help text into option labels or the question title.\n",
				q.ID, q.Type)
		}
	}

	// Content blocks (options, inputs, matrix, etc.) need their own groupUUID,
	// separate from the TITLE's groupUUID. Tally's editor requires this separation
	// to recognize questions as editable question blocks.
	contentGroupUUID := c.NewUUID()
	c.contentGroupUUIDs[q.ID] = contentGroupUUID

	switch q.Type {
	case model.SingleChoice:
		blocks = append(blocks, c.compileChoiceOptions(q, "MULTIPLE_CHOICE_OPTION", "MULTIPLE_CHOICE", false, contentGroupUUID)...)
	case model.MultiChoice:
		blocks = append(blocks, c.compileChoiceOptions(q, "CHECKBOX", "CHECKBOXES", true, contentGroupUUID)...)
	case model.Dropdown:
		blocks = append(blocks, c.compileChoiceOptions(q, "DROPDOWN_OPTION", "DROPDOWN", false, contentGroupUUID)...)
	case model.Matrix:
		blocks = append(blocks, c.compileMatrix(q, contentGroupUUID)...)
	case model.LongText:
		blocks = append(blocks, c.compileInputBlock(q, "TEXTAREA", "TEXTAREA", contentGroupUUID)...)
	case model.ShortText:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_TEXT", "INPUT_TEXT", contentGroupUUID)...)
	case model.Number:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_NUMBER", "INPUT_NUMBER", contentGroupUUID)...)
	case model.Email:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_EMAIL", "INPUT_EMAIL", contentGroupUUID)...)
	case model.Phone:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_PHONE_NUMBER", "INPUT_PHONE_NUMBER", contentGroupUUID)...)
	case model.URL:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_LINK", "INPUT_LINK", contentGroupUUID)...)
	case model.Date:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_DATE", "INPUT_DATE", contentGroupUUID)...)
	case model.Time:
		blocks = append(blocks, c.compileInputBlock(q, "INPUT_TIME", "INPUT_TIME", contentGroupUUID)...)
	case model.Rating:
		blocks = append(blocks, c.compileRatingBlock(q, contentGroupUUID)...)
	case model.Scale:
		blocks = append(blocks, c.compileScaleBlock(q, contentGroupUUID)...)
	case model.FileUpload:
		blocks = append(blocks, c.compileFileUploadBlock(q, contentGroupUUID)...)
	case model.Signature:
		blocks = append(blocks, c.compileSignatureBlock(q, contentGroupUUID)...)
	}

	return blocks
}

func (c *Compiler) compileChoiceOptions(q *model.Question, blockType, groupType string, allowMultiple bool, contentGroupUUID string) []TallyBlock {
	var blocks []TallyBlock
	hasOther := false
	for _, opt := range q.Options {
		if opt.IsOther {
			hasOther = true
			break
		}
	}

	for i, opt := range q.Options {
		optUUID := c.NewUUID()
		payload := map[string]any{
			"index":            i,
			"isFirst":          i == 0,
			"isLast":           i == len(q.Options)-1,
			"isRequired":       q.Required,
			"randomize":        false,
			"isOtherOption":    opt.IsOther,
			"hasMaxChoices":    q.Properties["max"] != nil,
			"hasDefaultAnswer": false,
			"hasOtherOption":   hasOther,
			"text":             opt.Text,
		}
		// MULTIPLE_CHOICE_OPTION-specific fields (not valid for DROPDOWN_OPTION)
		if blockType == "MULTIPLE_CHOICE_OPTION" {
			payload["allowMultiple"] = false
			payload["colorCodeOptions"] = false
			payload["hasBadge"] = true
			payload["badgeType"] = "LETTERS"
		}
		if q.Properties["max"] != nil {
			payload["maxChoices"] = q.Properties["max"]
		}
		if q.Hidden {
			payload["isHidden"] = true
		}

		block := TallyBlock{
			UUID:      optUUID,
			Type:      blockType,
			GroupUUID: contentGroupUUID,
			GroupType: groupType,
			Payload:   payload,
		}
		blocks = append(blocks, block)
		c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], optUUID)

		// Register option UUID for conditional reference
		c.optionUUIDs[q.ID][opt.Text] = optUUID
		if i == 0 {
			c.firstOptionUUID[q.ID] = optUUID
		}
	}

	return blocks
}

func (c *Compiler) compileInputBlock(q *model.Question, blockType, groupType, groupUUID string) []TallyBlock {
	inputUUID := c.NewUUID()
	payload := map[string]any{
		"isRequired":       q.Required,
		"hasDefaultAnswer": false,
	}
	// Character-limit fields are only accepted by text-based inputs. Email,
	// phone, URL, number, date, time, file-upload, signature etc. reject them
	// with a 400 VALIDATION error.
	if blockType == "TEXTAREA" || blockType == "INPUT_TEXT" {
		payload["hasMinCharacters"] = false
		payload["hasMaxCharacters"] = false
	}
	if q.Placeholder != "" {
		payload["placeholder"] = q.Placeholder
	}
	if q.Hidden {
		payload["isHidden"] = true
	}

	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], inputUUID)
	c.firstOptionUUID[q.ID] = inputUUID

	return []TallyBlock{{
		UUID:      inputUUID,
		Type:      blockType,
		GroupUUID: groupUUID,
		GroupType: groupType,
		Payload:   payload,
	}}
}

func (c *Compiler) compileMatrix(q *model.Question, contentGroupUUID string) []TallyBlock {
	// Matrix layout per Tally's actual editor output (the OpenAPI schema is
	// misleading on several points — verified by pulling a manually-created
	// matrix form and comparing):
	//
	//   MATRIX container:  type=MATRIX,       groupType=MATRIX,  groupUuid = contentGroupUUID (shared)
	//   MATRIX_COLUMN:     type=MATRIX_COLUMN, groupType=MATRIX, groupUuid = contentGroupUUID (shared)
	//   MATRIX_ROW:        type=MATRIX_ROW,    groupType=MATRIX, groupUuid = contentGroupUUID (shared)
	//
	// All three share one phantom groupUuid (same pattern as choice options
	// share one groupUuid). The MATRIX container uses groupType=MATRIX (not
	// QUESTION as the schema suggests). Its payload carries isFirst/isLast/
	// index values like an option block. Without these exact invariants, the
	// Tally editor marks the form as dirty on load and blocks delete
	// operations on other questions until the matrix is removed and the form
	// is re-saved.

	matrixUUID := c.NewUUID()
	matrixPayload := map[string]any{
		"isRequired": q.Required,
		"isFirst":    false,
		"isLast":     true,
		"index":      len(q.MatrixCols),
	}
	if q.Hidden {
		matrixPayload["isHidden"] = true
	}

	blocks := []TallyBlock{{
		UUID:      matrixUUID,
		Type:      "MATRIX",
		GroupUUID: contentGroupUUID,
		GroupType: "MATRIX",
		Payload:   matrixPayload,
	}}
	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], matrixUUID)
	c.firstOptionUUID[q.ID] = matrixUUID

	// MATRIX_COLUMN blocks — share the matrix content groupUuid.
	for i, col := range q.MatrixCols {
		colUUID := c.NewUUID()
		blocks = append(blocks, TallyBlock{
			UUID:      colUUID,
			Type:      "MATRIX_COLUMN",
			GroupUUID: contentGroupUUID,
			GroupType: "MATRIX",
			Payload: map[string]any{
				"safeHTMLSchema": SafeHTMLSchema(col),
				"isRequired":     q.Required,
				"index":          i,
				"isFirst":        i == 0,
				"isLast":         i == len(q.MatrixCols)-1,
			},
		})
		c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], colUUID)
	}

	// MATRIX_ROW blocks — share the matrix content groupUuid.
	for i, row := range q.MatrixRows {
		rowUUID := c.NewUUID()
		blocks = append(blocks, TallyBlock{
			UUID:      rowUUID,
			Type:      "MATRIX_ROW",
			GroupUUID: contentGroupUUID,
			GroupType: "MATRIX",
			Payload: map[string]any{
				"safeHTMLSchema": SafeHTMLSchema(row),
				"isRequired":     q.Required,
				"index":          i,
				"isFirst":        i == 0,
				"isLast":         i == len(q.MatrixRows)-1,
			},
		})
		c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], rowUUID)
	}

	return blocks
}

func (c *Compiler) compileScaleBlock(q *model.Question, groupUUID string) []TallyBlock {
	scaleUUID := c.NewUUID()
	payload := map[string]any{
		"isRequired":       q.Required,
		"hasDefaultAnswer": false,
	}
	if v, ok := q.Properties["start"]; ok {
		payload["start"] = v
	}
	if v, ok := q.Properties["end"]; ok {
		payload["end"] = v
	}
	if v, ok := q.Properties["step"]; ok {
		payload["step"] = v
	}
	if v, ok := q.Properties["left-label"]; ok {
		payload["hasLeftLabel"] = true
		payload["leftLabel"] = v
	}
	if v, ok := q.Properties["right-label"]; ok {
		payload["hasRightLabel"] = true
		payload["rightLabel"] = v
	}
	if q.Hidden {
		payload["isHidden"] = true
	}

	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], scaleUUID)
	c.firstOptionUUID[q.ID] = scaleUUID

	return []TallyBlock{{
		UUID:      scaleUUID,
		Type:      "LINEAR_SCALE",
		GroupUUID: groupUUID,
		GroupType: "LINEAR_SCALE",
		Payload:   payload,
	}}
}

func (c *Compiler) compileRatingBlock(q *model.Question, groupUUID string) []TallyBlock {
	ratingUUID := c.NewUUID()
	payload := map[string]any{
		"isRequired":       q.Required,
		"hasDefaultAnswer": false,
	}
	if v, ok := q.Properties["stars"]; ok {
		payload["stars"] = v
	}
	if q.Hidden {
		payload["isHidden"] = true
	}

	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], ratingUUID)
	c.firstOptionUUID[q.ID] = ratingUUID

	return []TallyBlock{{
		UUID:      ratingUUID,
		Type:      "RATING",
		GroupUUID: groupUUID,
		GroupType: "RATING",
		Payload:   payload,
	}}
}

func (c *Compiler) compileFileUploadBlock(q *model.Question, groupUUID string) []TallyBlock {
	uploadUUID := c.NewUUID()
	payload := map[string]any{
		"isRequired":       q.Required,
		"hasMultipleFiles": false,
		"hasMinFiles":      false,
		"hasMaxFiles":      false,
		"hasMaxFileSize":   false,
	}
	if q.Hidden {
		payload["isHidden"] = true
	}

	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], uploadUUID)
	c.firstOptionUUID[q.ID] = uploadUUID

	return []TallyBlock{{
		UUID:      uploadUUID,
		Type:      "FILE_UPLOAD",
		GroupUUID: groupUUID,
		GroupType: "FILE_UPLOAD",
		Payload:   payload,
	}}
}

func (c *Compiler) compileSignatureBlock(q *model.Question, groupUUID string) []TallyBlock {
	sigUUID := c.NewUUID()
	payload := map[string]any{
		"isRequired": q.Required,
	}
	if q.Hidden {
		payload["isHidden"] = true
	}

	c.questionBlockUUIDs[q.ID] = append(c.questionBlockUUIDs[q.ID], sigUUID)
	c.firstOptionUUID[q.ID] = sigUUID

	return []TallyBlock{{
		UUID:      sigUUID,
		Type:      "SIGNATURE",
		GroupUUID: groupUUID,
		GroupType: "SIGNATURE",
		Payload:   payload,
	}}
}

func (c *Compiler) compileConditional(cond *model.Conditional) ([]TallyBlock, error) {
	// Build show blocks list
	var showBlocks []string
	for _, target := range cond.Targets {
		uuids, ok := c.questionBlockUUIDs[target]
		if !ok {
			return nil, fmt.Errorf("conditional target %q not found", target)
		}
		showBlocks = append(showBlocks, uuids...)
	}

	// Build conditionals array
	var conditionals []any
	for _, condition := range cond.Conditions {
		// The conditional field reference uses the content group UUID
		// (the groupUuid shared by all option/input blocks of a question).
		// Both uuid and blockGroupUuid must be this content group UUID.
		contentGroup := c.contentGroupUUIDs[condition.Field]
		if contentGroup == "" {
			return nil, fmt.Errorf("conditional field %q not found", condition.Field)
		}
		questionType := c.getQuestionType(condition.Field)

		comparison := strings.ToUpper(condition.Comparison)

		// Validate operator compatibility with field type
		if err := validateConditionalOperator(questionType, comparison, condition.Field); err != nil {
			return nil, err
		}

		fieldTitle := c.questionTexts[condition.Field]

		condPayload := map[string]any{
			"field": map[string]any{
				"uuid":           contentGroup,
				"type":           "INPUT_FIELD",
				"questionType":   questionType,
				"blockGroupUuid": contentGroup,
				"title":          fieldTitle,
			},
			"comparison": comparison,
		}

		// Resolve values: option text → option UUID
		if len(condition.Values) > 0 {
			var resolvedValues []string
			for _, val := range condition.Values {
				optUUID, found := c.optionUUIDs[condition.Field][val]
				if found {
					resolvedValues = append(resolvedValues, optUUID)
				} else {
					// For non-choice fields, use the value directly
					resolvedValues = append(resolvedValues, val)
				}
			}
			// ANY_OF comparisons always take an array, even for a single value.
			// Scalar-typed comparisons (IS, IS_NOT, CONTAINS, DOES_NOT_CONTAIN)
			// take a single string value.
			if comparison == "IS_ANY_OF" || comparison == "IS_NOT_ANY_OF" {
				condPayload["value"] = resolvedValues
			} else if len(resolvedValues) == 1 {
				condPayload["value"] = resolvedValues[0]
			} else {
				condPayload["value"] = resolvedValues
			}
		} else {
			// value is always required by the API, even for IS_EMPTY/IS_NOT_EMPTY
			condPayload["value"] = ""
		}

		conditionals = append(conditionals, map[string]any{
			"uuid":    c.NewUUID(),
			"type":    "SINGLE",
			"payload": condPayload,
		})
	}

	return []TallyBlock{{
		UUID:      c.NewUUID(),
		Type:      "CONDITIONAL_LOGIC",
		GroupUUID: c.NewUUID(),
		GroupType: "CONDITIONAL_LOGIC",
		Payload: map[string]any{
			"logicalOperator": cond.Operator,
			"conditionals":    conditionals,
			"actions": []any{
				map[string]any{
					"uuid": c.NewUUID(),
					"type": "SHOW_BLOCKS",
					"payload": map[string]any{
						"showBlocks": showBlocks,
					},
				},
			},
		},
	}}, nil
}

func (c *Compiler) getQuestionType(fieldID string) string {
	qt := c.questionTypes[fieldID]
	switch qt {
	case model.SingleChoice:
		return "MULTIPLE_CHOICE"
	case model.MultiChoice:
		return "CHECKBOXES"
	case model.Dropdown:
		return "DROPDOWN"
	case model.LongText:
		return "TEXTAREA"
	case model.ShortText:
		return "INPUT_TEXT"
	case model.Number:
		return "INPUT_NUMBER"
	case model.Matrix:
		return "MATRIX"
	default:
		// Fallback based on options
		if len(c.optionUUIDs[fieldID]) > 0 {
			return "MULTIPLE_CHOICE"
		}
		return "TEXTAREA"
	}
}

func (c *Compiler) buildSettings(form *model.Form, cfg *config.Merged) map[string]any {
	settings := make(map[string]any)

	// Apply config defaults
	if cfg != nil {
		for k, v := range cfg.Settings {
			settings[k] = v
		}
		// Build styles from primary color
		if cfg.PrimaryColor != "" {
			settings["styles"] = map[string]any{
				"direction": "ltr",
				"color": map[string]any{
					"accent":           cfg.PrimaryColor,
					"buttonBackground": cfg.PrimaryColor,
				},
			}
		}
	}

	// Apply form-level overrides
	password := form.Password
	if password == "" && cfg != nil {
		password = cfg.Password
	}
	if password != "" {
		settings["password"] = password
	}
	if form.Settings != nil {
		// Map snake_case frontmatter keys to Tally camelCase API keys
		fmKeyMap := map[string]string{
			"has_progress_bar":        "hasProgressBar",
			"has_partial_submissions": "hasPartialSubmissions",
			"save_for_later":          "saveForLater",
		}
		for k, v := range form.Settings {
			if mapped, ok := fmKeyMap[k]; ok {
				settings[mapped] = v
			} else if k == "primary_color" {
				// primary_color in frontmatter overrides config
				if color, ok := v.(string); ok && color != "" {
					settings["styles"] = map[string]any{
						"direction": "ltr",
						"color": map[string]any{
							"accent":           color,
							"buttonBackground": color,
						},
					}
				}
			} else if k == "logo" {
				// logo goes into settings, not here — handled by buildFormTitle
			} else {
				settings[k] = v
			}
		}
	}

	return settings
}

// pageHasQuestions returns true if the page contains any Question blocks.
func pageHasQuestions(page model.Page) bool {
	for _, block := range page.Blocks {
		if _, ok := block.(*model.Question); ok {
			return true
		}
	}
	return false
}

// applyThankYouPage extracts text from a text-only page and sets
// closeMessageTitle and closeMessageDescription in settings.
func applyThankYouPage(page model.Page, settings map[string]any) {
	var title string
	var descParts []string

	for _, block := range page.Blocks {
		switch b := block.(type) {
		case *model.HeadingBlock:
			if title == "" {
				title = b.Text
			} else {
				descParts = append(descParts, b.Text)
			}
		case *model.TextBlock:
			text := b.HTML
			// Strip HTML tags for settings (plain text)
			text = stripHTMLTags(text)
			if title == "" {
				title = text
			} else {
				descParts = append(descParts, text)
			}
		}
	}

	if title != "" {
		settings["closeMessageTitle"] = title
	}
	if len(descParts) > 0 {
		settings["closeMessageDescription"] = strings.Join(descParts, "\n\n")
	}
}

// stripHTMLTags removes simple HTML tags like <b>, <i> from text for settings fields.
func stripHTMLTags(s string) string {
	result := strings.NewReplacer(
		"<b>", "", "</b>", "",
		"<i>", "", "</i>", "",
	).Replace(s)
	return result
}

// validateConditionalOperator checks that the comparison operator is supported
// for the given question type. Some operators are not supported by certain field types.
func validateConditionalOperator(questionType, comparison, fieldID string) error {
	switch questionType {
	case "CHECKBOXES":
		switch comparison {
		case "IS_NOT_ANY_OF", "IS_ANY_OF", "IS", "IS_NOT":
			return fmt.Errorf(
				"operator %q is not supported for multi-choice field %s; use is_empty or is_not_empty instead",
				strings.ToLower(comparison), fieldID,
			)
		}
	}
	return nil
}
