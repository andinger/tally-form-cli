package tally

import (
	"fmt"
	"sort"
	"strings"

	"github.com/andinger/tally-form-cli/internal/model"
)

// Decompile converts a TallyForm API response into an IR Form.
func Decompile(tf *TallyForm) (*model.Form, error) {
	form := &model.Form{
		Name:   tf.Name,
		FormID: tf.ID,
	}

	if pw, ok := tf.Settings["password"].(string); ok {
		form.Password = pw
	}

	// Process blocks sequentially
	blocks := tf.Blocks
	questionCounter := 0
	currentPage := &model.Page{}
	var pages []model.Page

	// Maps for conditional resolution
	groupToQID := make(map[string]string)       // groupUuid → question ID (for both TITLE and option groups)
	optionUUIDToText := make(map[string]string)  // option block UUID → option text
	blockUUIDToQID := make(map[string]string)    // any block UUID → question ID (for showBlocks)

	// First pass: build maps
	i := 0
	for i < len(blocks) {
		b := blocks[i]

		switch b.Type {
		case "FORM_TITLE":
			i++
			continue

		case "PAGE_BREAK":
			i++
			continue

		case "HEADING_1", "HEADING_2", "TEXT":
			i++
			continue

		case "CONDITIONAL_LOGIC":
			i++
			continue

		case "TITLE":
			questionCounter++
			qID := fmt.Sprintf("F%d", questionCounter)
			groupToQID[b.GroupUUID] = qID
			blockUUIDToQID[b.UUID] = qID

			// Look ahead for companion blocks (including hint TEXT blocks)
			i++
			for i < len(blocks) {
				nb := blocks[i]
				if isQuestionContent(nb.Type) {
					groupToQID[nb.GroupUUID] = qID
					blockUUIDToQID[nb.UUID] = qID
					if text, ok := nb.Payload["text"].(string); ok {
						optionUUIDToText[nb.UUID] = text
					}
					i++
				} else if nb.Type == "TEXT" && i+1 < len(blocks) && isQuestionContent(blocks[i+1].Type) {
					// Hint TEXT block between TITLE and content
					blockUUIDToQID[nb.UUID] = qID
					i++
				} else {
					break
				}
			}
			continue

		default:
			i++
		}
	}

	// Second pass: build form structure
	questionCounter = 0
	i = 0
	for i < len(blocks) {
		b := blocks[i]

		switch b.Type {
		case "FORM_TITLE":
			i++

		case "PAGE_BREAK":
			pages = append(pages, *currentPage)
			currentPage = &model.Page{}
			if btn, ok := b.Payload["button"].(map[string]any); ok {
				if label, ok := btn["label"].(string); ok && label != "Weiter" {
					currentPage.ButtonLabel = label
				}
			}
			i++

		case "HEADING_1", "HEADING_2":
			level := 2
			if b.Type == "HEADING_1" {
				level = 1
			}
			currentPage.Blocks = append(currentPage.Blocks, &model.HeadingBlock{
				Text:  extractText(b),
				Level: level,
			})
			i++

		case "TEXT":
			currentPage.Blocks = append(currentPage.Blocks, &model.TextBlock{
				HTML: extractText(b),
			})
			i++

		case "CONDITIONAL_LOGIC":
			cond := decompileConditional(b, groupToQID, optionUUIDToText, blockUUIDToQID)
			if cond != nil {
				currentPage.Blocks = append(currentPage.Blocks, cond)
			}
			i++

		case "TITLE":
			questionCounter++
			qID := fmt.Sprintf("F%d", questionCounter)

			q := &model.Question{
				ID:         qID,
				Text:       extractText(b),
				Required:   true,
				Properties: make(map[string]any),
			}
			if hidden, ok := b.Payload["isHidden"].(bool); ok {
				q.Hidden = hidden
			}

			i++

			// Consume companion blocks (including hint TEXT blocks between TITLE and content)
			for i < len(blocks) {
				nb := blocks[i]

				// TEXT block between TITLE and content = hint
				if nb.Type == "TEXT" {
					// Check if the next block after this TEXT is question content
					if i+1 < len(blocks) && isQuestionContent(blocks[i+1].Type) {
						hintText := extractText(nb)
						// Strip italic wrapper if present
						hintText = strings.TrimPrefix(hintText, "<i>")
						hintText = strings.TrimSuffix(hintText, "</i>")
						q.Hint = hintText
						blockUUIDToQID[nb.UUID] = qID
						i++
						continue
					}
					break
				}

				if !isQuestionContent(nb.Type) {
					break
				}

				switch nb.Type {
				case "MULTIPLE_CHOICE_OPTION":
					q.Type = model.SingleChoice
					opt := model.Option{Text: getPayloadText(nb)}
					if isOther, ok := nb.Payload["isOtherOption"].(bool); ok && isOther {
						opt.IsOther = true
					}
					q.Options = append(q.Options, opt)
					if req, ok := nb.Payload["isRequired"].(bool); ok {
						q.Required = req
					}
					if hidden, ok := nb.Payload["isHidden"].(bool); ok && hidden {
						q.Hidden = true
					}
					if hasMax, ok := nb.Payload["hasMaxChoices"].(bool); ok && hasMax {
						if max, ok := nb.Payload["maxChoices"].(float64); ok {
							q.Properties["max"] = int(max)
						}
					}

				case "CHECKBOX":
					q.Type = model.MultiChoice
					opt := model.Option{Text: getPayloadText(nb)}
					if isOther, ok := nb.Payload["isOtherOption"].(bool); ok && isOther {
						opt.IsOther = true
					}
					q.Options = append(q.Options, opt)
					if req, ok := nb.Payload["isRequired"].(bool); ok {
						q.Required = req
					}
					if hidden, ok := nb.Payload["isHidden"].(bool); ok && hidden {
						q.Hidden = true
					}
					if hasMax, ok := nb.Payload["hasMaxChoices"].(bool); ok && hasMax {
						if max, ok := nb.Payload["maxChoices"].(float64); ok {
							q.Properties["max"] = int(max)
						}
					}

				case "DROPDOWN_OPTION":
					q.Type = model.Dropdown
					q.Options = append(q.Options, model.Option{Text: getPayloadText(nb)})

				case "TEXTAREA":
					q.Type = model.LongText
					if ph, ok := nb.Payload["placeholder"].(string); ok {
						q.Placeholder = ph
					}
					if req, ok := nb.Payload["isRequired"].(bool); ok {
						q.Required = req
					}

				case "INPUT_TEXT":
					q.Type = model.ShortText
					if ph, ok := nb.Payload["placeholder"].(string); ok {
						q.Placeholder = ph
					}
					if req, ok := nb.Payload["isRequired"].(bool); ok {
						q.Required = req
					}

				case "INPUT_NUMBER":
					q.Type = model.Number
				case "INPUT_EMAIL":
					q.Type = model.Email
				case "INPUT_PHONE_NUMBER":
					q.Type = model.Phone
				case "INPUT_LINK":
					q.Type = model.URL
				case "INPUT_DATE":
					q.Type = model.Date
				case "INPUT_TIME":
					q.Type = model.Time
				case "RATING":
					q.Type = model.Rating
				case "LINEAR_SCALE":
					q.Type = model.Scale
				case "FILE_UPLOAD":
					q.Type = model.FileUpload
				case "SIGNATURE":
					q.Type = model.Signature

				case "MATRIX":
					q.Type = model.Matrix
				case "MATRIX_COLUMN":
					q.Type = model.Matrix
					q.MatrixCols = append(q.MatrixCols, getPayloadText(nb))
				case "MATRIX_ROW":
					q.Type = model.Matrix
					q.MatrixRows = append(q.MatrixRows, getPayloadText(nb))
				}

				i++
			}

			currentPage.Blocks = append(currentPage.Blocks, q)

		default:
			i++
		}
	}

	pages = append(pages, *currentPage)
	form.Pages = pages

	return form, nil
}

func isQuestionContent(blockType string) bool {
	switch blockType {
	case "MULTIPLE_CHOICE_OPTION", "CHECKBOX", "DROPDOWN_OPTION",
		"TEXTAREA", "INPUT_TEXT", "INPUT_NUMBER", "INPUT_EMAIL",
		"INPUT_PHONE_NUMBER", "INPUT_LINK", "INPUT_DATE", "INPUT_TIME",
		"RATING", "LINEAR_SCALE", "FILE_UPLOAD", "SIGNATURE",
		"MATRIX", "MATRIX_COLUMN", "MATRIX_ROW":
		return true
	}
	return false
}

func decompileConditional(b TallyBlock, groupToQID map[string]string, optionUUIDToText map[string]string, blockUUIDToQID map[string]string) *model.Conditional {
	cond := &model.Conditional{
		Operator: "AND",
	}

	if op, ok := b.Payload["logicalOperator"].(string); ok {
		cond.Operator = op
	}

	// Parse conditionals → conditions
	if conditionals, ok := b.Payload["conditionals"].([]any); ok {
		for _, c := range conditionals {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			payload, ok := cm["payload"].(map[string]any)
			if !ok {
				continue
			}
			field, ok := payload["field"].(map[string]any)
			if !ok {
				continue
			}

			blockGroupUUID, _ := field["blockGroupUuid"].(string)
			fieldQID := groupToQID[blockGroupUUID]
			if fieldQID == "" {
				fieldQID = "F?"
			}

			comparison, _ := payload["comparison"].(string)
			comparison = strings.ToLower(comparison)

			var values []string
			switch v := payload["value"].(type) {
			case string:
				if v != "" {
					if text, ok := optionUUIDToText[v]; ok {
						values = append(values, text)
					} else {
						values = append(values, v)
					}
				}
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						if text, ok := optionUUIDToText[s]; ok {
							values = append(values, text)
						} else {
							values = append(values, s)
						}
					}
				}
			}

			cond.Conditions = append(cond.Conditions, model.Condition{
				Field:      fieldQID,
				Comparison: comparison,
				Values:     values,
			})
		}
	}

	// Parse targets from showBlocks
	if actions, ok := b.Payload["actions"].([]any); ok {
		targetSet := make(map[string]bool)
		for _, a := range actions {
			am, ok := a.(map[string]any)
			if !ok {
				continue
			}
			if am["type"] == "SHOW_BLOCKS" {
				if payload, ok := am["payload"].(map[string]any); ok {
					if showBlocks, ok := payload["showBlocks"].([]any); ok {
						for _, sb := range showBlocks {
							if sbStr, ok := sb.(string); ok {
								if qID, ok := blockUUIDToQID[sbStr]; ok {
									targetSet[qID] = true
								}
							}
						}
					}
				}
			}
		}
		for qID := range targetSet {
			cond.Targets = append(cond.Targets, qID)
		}
		sort.Strings(cond.Targets)
	}

	if len(cond.Targets) == 0 {
		cond.Targets = []string{"F?"}
	}

	return cond
}

func extractText(b TallyBlock) string {
	if schema, ok := b.Payload["safeHTMLSchema"].([]any); ok {
		return extractFromSchema(schema)
	}
	if title, ok := b.Payload["title"].(string); ok {
		return title
	}
	return ""
}

func extractFromSchema(schema []any) string {
	var parts []string
	for _, item := range schema {
		arr, ok := item.([]any)
		if !ok || len(arr) == 0 {
			continue
		}
		text, ok := arr[0].(string)
		if !ok {
			continue
		}

		// Check for style properties in second element
		if len(arr) >= 2 {
			if styles, ok := arr[1].([]any); ok {
				if hasStyleProp(styles, "font-weight", "bold") {
					text = "<b>" + text + "</b>"
				} else if hasStyleProp(styles, "font-style", "italic") {
					text = "<i>" + text + "</i>"
				} else if href := getStyleValue(styles, "href"); href != "" {
					text = `<a href="` + href + `">` + text + "</a>"
				}
			}
		}

		parts = append(parts, text)
	}
	return strings.Join(parts, "")
}

// hasStyleProp checks if a styles array contains a specific property-value pair.
// Styles format: [["tag","span"],["font-weight","bold"]]
func hasStyleProp(styles []any, prop, value string) bool {
	for _, item := range styles {
		pair, ok := item.([]any)
		if !ok || len(pair) != 2 {
			continue
		}
		k, _ := pair[0].(string)
		v, _ := pair[1].(string)
		if k == prop && v == value {
			return true
		}
	}
	return false
}

// getStyleValue returns the value for a style key (e.g. "href") or empty string.
func getStyleValue(styles []any, key string) string {
	for _, item := range styles {
		pair, ok := item.([]any)
		if !ok || len(pair) != 2 {
			continue
		}
		k, _ := pair[0].(string)
		if k == key {
			v, _ := pair[1].(string)
			return v
		}
	}
	return ""
}

func getPayloadText(b TallyBlock) string {
	if text, ok := b.Payload["text"].(string); ok {
		return text
	}
	return extractText(b)
}
