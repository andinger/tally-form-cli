# tally Reference

Bidirectional Markdown-to-Tally.so form builder. Single Go binary, installed via Homebrew.

## Installation

```bash
brew install andinger/tap/tally
```

## Commands

| Command | Usage | Description |
|---|---|---|
| `push` | `push <file.md> [--dry-run] [--create]` | Upsert: creates if no `form_id`, updates if present. `--create` forces a new form. |
| `pull` | `pull <form-id>` | Downloads form as Markdown to stdout |
| `diff` | `diff <file.md> [form-id]` | Compares local Markdown with a Tally form. Uses `form_id` from frontmatter if not provided. |
| `submissions` | `submissions <form-id> [--format csv\|json]` | Downloads responses (default: csv) to stdout |
| `prepare` | `prepare <file.md>` | Merges global config (workspace, logo, password, primary_color, domain) into frontmatter |
| `config` | `config` | Shows the global config file path |
| `reference` | `reference` | Prints this reference documentation to stdout |

### Global Flags

| Flag | Description |
|---|---|
| `--token <token>` | Tally API token (overrides config) |

## Configuration

Two layers, merged in order: **Global config** < **Frontmatter**

### Global Config (`~/.config/tally/config.yaml`)

Credentials and defaults per machine. Not versioned.

```yaml
api:
  token: "tly_xxxxxxxxxxxx"
  base_url: "https://api.tally.so"

workspace: "mOJGz8"
logo: "https://storage.tally.so/..."
primary_color: "#A219B1"
password: "optional-password"
domain: "forms.example.com"
```

`primary_color` is used as both the accent color and button background color in Tally.

### Frontmatter

Per-form overrides in the Markdown file:

```yaml
---
name: "Survey — Acme Corp"
form_id: "auto-filled-after-push"
workspace: "override-ws"
password: "form-specific-password"
---
```

`form_id` is written back into the file after the first `push`. This enables upsert semantics.

Use `tally prepare <file.md>` to copy global settings into the frontmatter.

## Markdown Format

### Questions

```markdown
F1: Question text here?
> type: single-choice
> required: true
> hint: "Helper text"
- Option A
- Option B
- Other {other}
```

ID format: `F<n>:` — must be sequential (F1, F2, F3, ...).

### Question Types

| Markdown Type | Tally Block | Has Options |
|---|---|---|
| `single-choice` | MULTIPLE_CHOICE | yes |
| `multi-choice` | CHECKBOXES | yes |
| `dropdown` | DROPDOWN | yes |
| `long-text` | TEXTAREA | no |
| `short-text` | INPUT_TEXT | no |
| `number` | INPUT_NUMBER | no |
| `email` | INPUT_EMAIL | no |
| `phone` | INPUT_PHONE_NUMBER | no |
| `url` | INPUT_LINK | no |
| `date` | INPUT_DATE | no |
| `time` | INPUT_TIME | no |
| `rating` | RATING | no |
| `scale` | LINEAR_SCALE | no |
| `matrix` | MATRIX | rows as list items |
| `file-upload` | FILE_UPLOAD | no |
| `signature` | SIGNATURE | no |

### Question Metadata

| Key | Default | Description |
|---|---|---|
| `type` | — | Required. Question type |
| `required` | `true` | Required field |
| `hint` | — | Helper text below question |
| `placeholder` | — | Placeholder for text fields |
| `hidden` | `false` | Initially hidden (for conditionals) |
| `max` | — | Max selections (multi-choice) |
| `min` | — | Min selections |
| `other` | `false` | Auto-add "Andere" option |
| `columns` | — | Column headers for matrix |
| `stars` | `5` | Stars for rating |
| `start` | `0` | Scale start value |
| `end` | `10` | Scale end value |
| `step` | `1` | Scale step size |
| `left-label` | — | Left label for scale |
| `right-label` | — | Right label for scale |

### Other Option

Two equivalent ways:

```markdown
# Inline: {other} suffix on an option
- Andere {other}

# Metadata: auto-adds "Andere" as last option
F1: Question?
> type: single-choice
> other: true
- Option A
```

### Matrix Questions

```markdown
F12: Rate the complexity of these documents?
> type: matrix
> columns: Low, Moderate, High, N/A
- Reports
- Protocols
- Documentation
```

### Page Breaks

Pages separated by `---` (thematic break). Optional button label:

```markdown
---
> button: "Next page 3 / 5"
```

### Headings, Text and Inline Formatting

```markdown
## Section Title          → HEADING_2 block
Plain paragraph text.     → TEXT block
**Bold text**             → bold in Tally
*Italic text*             → italic in Tally
[Link text](https://url)  → clickable link in Tally
```

### Thank-You Page

The last page (after the final `---`) is used as the thank-you page if it contains only text (no questions). The first text block becomes `closeMessageTitle`, the rest becomes `closeMessageDescription`.

```markdown
---

**Thank you for your time!**

Your answers help us prepare the workshop.
```

### Conditional Logic

```markdown
> show F3, F4 when F2 is_not_any_of "Option A", "Option B"
> show F17 when F16 is_not_empty
> show F20 when F19 is_not_empty and F19 does_not_contain "None"
```

**Syntax:** `> show <targets> when <condition> [and|or <condition>]*`

**Operators:**

| Operator | Description | Supported Field Types |
|---|---|---|
| `is "text"` | Equals | single-choice, dropdown, text |
| `is_not "text"` | Not equals | single-choice, dropdown, text |
| `is_any_of "a", "b"` | Matches any | single-choice, dropdown |
| `is_not_any_of "a", "b"` | Matches none | single-choice, dropdown |
| `is_empty` | Field empty | all types |
| `is_not_empty` | Field not empty | all types |
| `contains "text"` | Contains value | text fields |
| `does_not_contain "text"` | Does not contain | text fields |

**Note:** `is`, `is_not`, `is_any_of`, `is_not_any_of` are not supported for multi-choice (CHECKBOXES) fields. Use `is_empty` / `is_not_empty` instead.

**Rules:**
- Target questions must have `> hidden: true`
- Conditionals reference question IDs (`F1`, `F2`, ...) and option text
- Place conditionals after the referenced questions
- `and` / `or` for compound conditions

## Workflow Examples

```bash
# Create a new form from Markdown
tally push questionnaire.md --dry-run
tally push questionnaire.md

# Force creating a new form (even if form_id exists)
tally push questionnaire.md --create

# Download existing form as Markdown
tally pull 81GYAY > exported.md

# Compare local file with Tally
tally diff questionnaire.md

# Download submissions as CSV
tally submissions 81GYAY --format csv > responses.csv

# Update after editing (form_id in frontmatter enables upsert)
tally push questionnaire.md

# Prepare a new markdown file with global settings
tally prepare new-form.md

# Show config location
tally config

# Generate reference for Claude
tally reference > ~/.claude/references/tally.md
```
