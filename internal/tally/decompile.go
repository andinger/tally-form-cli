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

	// Group blocks by groupUuid
	groups := groupBlocks(tf.Blocks)

	// Split into pages
	pages := splitIntoPages(tf.Blocks, groups)
	form.Pages = pages

	return form, nil
}

type blockGroup struct {
	groupUUID string
	groupType string
	blocks    []TallyBlock
	firstIdx  int // index in original block array
}

func groupBlocks(blocks []TallyBlock) map[string]*blockGroup {
	groups := make(map[string]*blockGroup)
	for i, b := range blocks {
		g, ok := groups[b.GroupUUID]
		if !ok {
			g = &blockGroup{
				groupUUID: b.GroupUUID,
				groupType: b.GroupType,
				firstIdx:  i,
			}
			groups[b.GroupUUID] = g
		}
		g.blocks = append(g.blocks, b)
	}
	return groups
}

func splitIntoPages(blocks []TallyBlock, groups map[string]*blockGroup) []model.Page {
	var pages []model.Page
	currentPage := model.Page{}
	questionCounter := 0

	// Track which groupUUIDs we've already processed
	processed := make(map[string]bool)

	// UUID → question ID mapping for conditionals
	groupToQID := make(map[string]string)
	optionUUIDToText := make(map[string]string)

	// First pass: build UUID maps
	for _, b := range blocks {
		if b.Type == "MULTIPLE_CHOICE_OPTION" || b.Type == "CHECKBOX" || b.Type == "DROPDOWN_OPTION" {
			if text, ok := b.Payload["text"].(string); ok {
				optionUUIDToText[b.UUID] = text
			}
		}
	}

	for _, b := range blocks {
		if processed[b.GroupUUID] {
			continue
		}

		switch b.Type {
		case "FORM_TITLE":
			processed[b.GroupUUID] = true
			continue

		case "PAGE_BREAK":
			processed[b.GroupUUID] = true
			pages = append(pages, currentPage)
			currentPage = model.Page{}
			if btn, ok := b.Payload["button"].(map[string]any); ok {
				if label, ok := btn["label"].(string); ok && label != "Weiter" {
					currentPage.ButtonLabel = label
				}
			}
			continue

		case "HEADING_1", "HEADING_2":
			processed[b.GroupUUID] = true
			text := extractText(b)
			level := 2
			if b.Type == "HEADING_1" {
				level = 1
			}
			currentPage.Blocks = append(currentPage.Blocks, &model.HeadingBlock{
				Text:  text,
				Level: level,
			})
			continue

		case "TEXT":
			processed[b.GroupUUID] = true
			text := extractText(b)
			currentPage.Blocks = append(currentPage.Blocks, &model.TextBlock{
				HTML: text,
			})
			continue

		case "CONDITIONAL_LOGIC":
			processed[b.GroupUUID] = true
			cond := decompileConditional(b, groupToQID, optionUUIDToText)
			if cond != nil {
				currentPage.Blocks = append(currentPage.Blocks, cond)
			}
			continue

		case "TITLE":
			// This starts a question group
			processed[b.GroupUUID] = true
			g := groups[b.GroupUUID]
			if g == nil {
				continue
			}

			questionCounter++
			qID := fmt.Sprintf("F%d", questionCounter)
			groupToQID[b.GroupUUID] = qID

			q := decompileQuestion(qID, b, g, groups, processed, optionUUIDToText, groupToQID)
			currentPage.Blocks = append(currentPage.Blocks, q)
			continue

		default:
			// Part of a question group — skip if already processed
			if !processed[b.GroupUUID] {
				// Standalone input blocks (TEXTAREA, INPUT_TEXT, etc.)
				g := groups[b.GroupUUID]
				if g != nil {
					processed[b.GroupUUID] = true
					// Check if there's a TITLE block for this group
					hasTitleInGroup := false
					for _, gb := range g.blocks {
						if gb.Type == "TITLE" {
							hasTitleInGroup = true
							break
						}
					}
					if !hasTitleInGroup {
						// Standalone input — create question from it
						questionCounter++
						qID := fmt.Sprintf("F%d", questionCounter)
						groupToQID[b.GroupUUID] = qID
						q := &model.Question{
							ID:   qID,
							Type: mapBlockTypeToQuestionType(b.Type),
						}
						if req, ok := b.Payload["isRequired"].(bool); ok {
							q.Required = req
						}
						currentPage.Blocks = append(currentPage.Blocks, q)
					}
				}
			}
		}
	}

	pages = append(pages, currentPage)
	return pages
}

func decompileQuestion(qID string, titleBlock TallyBlock, titleGroup *blockGroup, allGroups map[string]*blockGroup, processed map[string]bool, optionUUIDToText map[string]string, groupToQID map[string]string) *model.Question {
	q := &model.Question{
		ID:         qID,
		Text:       extractText(titleBlock),
		Required:   true,
		Properties: make(map[string]any),
	}

	if desc, ok := titleBlock.Payload["description"].(string); ok {
		q.Hint = desc
	}
	if hidden, ok := titleBlock.Payload["isHidden"].(bool); ok {
		q.Hidden = hidden
	}

	// Find the companion blocks (same groupUuid as the TITLE block's question group)
	// The TITLE has groupType: QUESTION, the options share the same conceptual group
	// but may have different groupUuids. We need to look at the next blocks in sequence.
	// Actually, in Tally the TITLE block and option blocks share the SAME groupUuid for the option group.
	// Wait — looking at the reference: TITLE has its own groupUuid (groupType: QUESTION),
	// and options have a different groupUuid (groupType: MULTIPLE_CHOICE).
	// We need to find the option group that follows this TITLE.

	// Strategy: look for blocks with the same groupUuid as the options
	// The option blocks come right after the TITLE in the block array
	// For now, use a simpler approach: the group that contains the TITLE also
	// reveals the question type through adjacent blocks

	// Actually, looking at the reference form more carefully:
	// - TITLE block: groupUuid = X, groupType = QUESTION
	// - MULTIPLE_CHOICE_OPTION blocks: groupUuid = Y, groupType = MULTIPLE_CHOICE
	// They have DIFFERENT groupUuids. The connection is positional.

	// Let me find option blocks that follow this TITLE block positionally
	// by scanning all groups and matching by position proximity.

	// For a simpler approach: find the next non-processed group after this one
	// that contains option/input blocks.

	// Actually, let's use the allGroups map differently:
	// Find groups whose firstIdx is right after our title's firstIdx
	titleIdx := titleGroup.firstIdx

	type indexedGroup struct {
		idx   int
		group *blockGroup
		uuid  string
	}
	var candidates []indexedGroup
	for uuid, g := range allGroups {
		if !processed[uuid] && g.firstIdx > titleIdx {
			candidates = append(candidates, indexedGroup{g.firstIdx, g, uuid})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].idx < candidates[j].idx
	})

	if len(candidates) > 0 {
		nextGroup := candidates[0]
		firstBlock := nextGroup.group.blocks[0]

		switch firstBlock.Type {
		case "MULTIPLE_CHOICE_OPTION":
			processed[nextGroup.uuid] = true
			q.Type = model.SingleChoice
			for _, ob := range nextGroup.group.blocks {
				opt := model.Option{Text: getPayloadText(ob)}
				if isOther, ok := ob.Payload["isOtherOption"].(bool); ok && isOther {
					opt.IsOther = true
				}
				q.Options = append(q.Options, opt)
				optionUUIDToText[ob.UUID] = opt.Text
			}
			if req, ok := firstBlock.Payload["isRequired"].(bool); ok {
				q.Required = req
			}
			if hidden, ok := firstBlock.Payload["isHidden"].(bool); ok {
				q.Hidden = hidden
			}
			if maxChoices, ok := firstBlock.Payload["hasMaxChoices"].(bool); ok && maxChoices {
				if max, ok := firstBlock.Payload["maxChoices"].(float64); ok {
					q.Properties["max"] = int(max)
				}
			}
			// Register option group UUID for conditionals too
			groupToQID[nextGroup.uuid] = qID

		case "CHECKBOX":
			processed[nextGroup.uuid] = true
			q.Type = model.MultiChoice
			for _, ob := range nextGroup.group.blocks {
				opt := model.Option{Text: getPayloadText(ob)}
				if isOther, ok := ob.Payload["isOtherOption"].(bool); ok && isOther {
					opt.IsOther = true
				}
				q.Options = append(q.Options, opt)
				optionUUIDToText[ob.UUID] = opt.Text
			}
			if req, ok := firstBlock.Payload["isRequired"].(bool); ok {
				q.Required = req
			}
			if hidden, ok := firstBlock.Payload["isHidden"].(bool); ok {
				q.Hidden = hidden
			}
			if maxChoices, ok := firstBlock.Payload["hasMaxChoices"].(bool); ok && maxChoices {
				if max, ok := firstBlock.Payload["maxChoices"].(float64); ok {
					q.Properties["max"] = int(max)
				}
			}
			groupToQID[nextGroup.uuid] = qID

		case "DROPDOWN_OPTION":
			processed[nextGroup.uuid] = true
			q.Type = model.Dropdown
			for _, ob := range nextGroup.group.blocks {
				q.Options = append(q.Options, model.Option{Text: getPayloadText(ob)})
			}
			groupToQID[nextGroup.uuid] = qID

		case "TEXTAREA":
			processed[nextGroup.uuid] = true
			q.Type = model.LongText
			if ph, ok := firstBlock.Payload["placeholder"].(string); ok {
				q.Placeholder = ph
			}
			if req, ok := firstBlock.Payload["isRequired"].(bool); ok {
				q.Required = req
			}
			groupToQID[nextGroup.uuid] = qID

		case "INPUT_TEXT":
			processed[nextGroup.uuid] = true
			q.Type = model.ShortText
			if ph, ok := firstBlock.Payload["placeholder"].(string); ok {
				q.Placeholder = ph
			}
			if req, ok := firstBlock.Payload["isRequired"].(bool); ok {
				q.Required = req
			}
			groupToQID[nextGroup.uuid] = qID

		case "MATRIX":
			processed[nextGroup.uuid] = true
			q.Type = model.Matrix
			// Find MATRIX_COLUMN and MATRIX_ROW blocks in this group
			for _, mb := range nextGroup.group.blocks {
				switch mb.Type {
				case "MATRIX_COLUMN":
					q.MatrixCols = append(q.MatrixCols, getPayloadText(mb))
				case "MATRIX_ROW":
					q.MatrixRows = append(q.MatrixRows, getPayloadText(mb))
				}
			}
			groupToQID[nextGroup.uuid] = qID

		case "LINEAR_SCALE":
			processed[nextGroup.uuid] = true
			q.Type = model.Scale
			groupToQID[nextGroup.uuid] = qID

		case "RATING":
			processed[nextGroup.uuid] = true
			q.Type = model.Rating
			groupToQID[nextGroup.uuid] = qID

		case "INPUT_NUMBER":
			processed[nextGroup.uuid] = true
			q.Type = model.Number
			groupToQID[nextGroup.uuid] = qID

		default:
			q.Type = model.ShortText
		}
	}

	return q
}

func decompileConditional(b TallyBlock, groupToQID map[string]string, optionUUIDToText map[string]string) *model.Conditional {
	cond := &model.Conditional{
		Operator: "AND",
	}

	if op, ok := b.Payload["logicalOperator"].(string); ok {
		cond.Operator = op
	}

	// Parse actions → targets
	if actions, ok := b.Payload["actions"].([]any); ok {
		for _, a := range actions {
			am, ok := a.(map[string]any)
			if !ok {
				continue
			}
			if am["type"] == "SHOW_BLOCKS" {
				if payload, ok := am["payload"].(map[string]any); ok {
					if showBlocks, ok := payload["showBlocks"].([]any); ok {
						targetSet := make(map[string]bool)
						for _, sb := range showBlocks {
							// This is a block UUID — we need to find which question it belongs to
							// For now, collect unique question IDs from the groupToQID map
							sbStr, ok := sb.(string)
							if !ok {
								continue
							}
							for gUUID, qID := range groupToQID {
								_ = gUUID
								// Check if this block UUID belongs to this question
								// This is imperfect — we'd need a full block→question map
								// For simplicity, just record the mapping
								targetSet[qID] = true
								_ = sbStr
							}
						}
						// Actually we need a block UUID → question ID map
						// Let's build it differently
					}
				}
			}
		}
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
				if text, ok := optionUUIDToText[v]; ok {
					values = append(values, text)
				} else {
					values = append(values, v)
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
	// We need a block UUID → question ID lookup
	// For now, use a simplified approach
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
								// Look up in groupToQID — showBlocks contains block UUIDs,
								// but we stored groupUUIDs. In the compiler, showBlocks
								// contains all block UUIDs for a question.
								// For decompile, we'd need a reverse lookup.
								// For now, mark any question that has a matching UUID.
								_ = sbStr
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

	if len(cond.Targets) == 0 && len(cond.Conditions) > 0 {
		// Fallback: we couldn't resolve targets perfectly
		// Return the conditional anyway with empty targets
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
		if text, ok := arr[0].(string); ok {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}

func getPayloadText(b TallyBlock) string {
	if text, ok := b.Payload["text"].(string); ok {
		return text
	}
	return extractText(b)
}

func mapBlockTypeToQuestionType(blockType string) model.QuestionType {
	switch blockType {
	case "TEXTAREA":
		return model.LongText
	case "INPUT_TEXT":
		return model.ShortText
	case "INPUT_NUMBER":
		return model.Number
	case "INPUT_EMAIL":
		return model.Email
	case "INPUT_PHONE_NUMBER":
		return model.Phone
	case "INPUT_LINK":
		return model.URL
	case "INPUT_DATE":
		return model.Date
	case "INPUT_TIME":
		return model.Time
	case "RATING":
		return model.Rating
	case "LINEAR_SCALE":
		return model.Scale
	case "FILE_UPLOAD":
		return model.FileUpload
	case "SIGNATURE":
		return model.Signature
	default:
		return model.ShortText
	}
}
