# tally Reference

Bidirectional Markdown-to-Tally.so form builder. Single Go binary, installed via Homebrew.

## Installation

```bash
brew install andinger/tap/tally
```

## Invocation

```bash
tally <command> [flags]
```

### Commands

| Command | Usage | Description |
|---|---|---|
| `push` | `push <file.md> [--dry-run]` | Upsert: creates if no `form_id` in frontmatter, updates if present |
| `create` | `create <file.md> [--dry-run]` | Always creates a new form (ignores `form_id`) |
| `update` | `update <form-id> <file.md> [--dry-run]` | Updates an existing form |
| `export` | `export <form-id>` | Downloads form as Markdown to stdout |
| `submissions` | `submissions <form-id> [--format csv\|json]` | Downloads responses (default: csv) to stdout |
| `reference` | `reference` | Prints this reference documentation to stdout |

### Global Flags

| Flag | Description |
|---|---|
| `--config <path>` | Path to project config YAML (theme, branding, defaults) |
| `--token <token>` | Tally API token (overrides config) |
| `--dry-run` | Print JSON payload without calling API (on push/create/update) |

## Configuration — Three Layers

Merge order: User-Config < Project-Config < Form-Frontmatter < CLI-Flags

### Layer 1: User Config (`~/.config/tally/config.yaml`)

Credentials and defaults per machine. Not versioned.

```yaml
api:
  token: "tly_xxxxxxxxxxxx"
  base_url: "https://api.tally.so"

workspace: "mOJGz8"
```

### Layer 2: Project Config (`--config tally-config.yaml`)

Theme, branding, form defaults per product/project. Versioned, shared in team.

```yaml
defaults:
  language: "de"
  status: "PUBLISHED"
  hasProgressBar: true
  saveForLater: true
  pageAutoJump: false
  hasPartialSubmissions: false
  closeMessageTitle: "Vielen Dank..."
  closeMessageDescription: "Falls noch nicht geschehen..."

logo: "https://storage.tally.so/..."

styles:
  theme: "CUSTOM"
  direction: "ltr"
  color:
    background: "#ffffff"
    text: "#37352F"
    accent: "#A219B1"
    buttonBackground: "#07E9A4"
    buttonText: "#000000"
  css: ""
```

### Layer 3: Form Frontmatter

Per-form overrides in the Markdown file:

```yaml
---
name: "KI-Hebel-Check — Mustermann GmbH"
password: "muster-check"
form_id: "81d6KA"   # written back after create
workspace: "other"   # overrides user config
---
```

## Markdown Format

### Frontmatter

```yaml
---
name: "Form Title"
password: "optional-password"
form_id: "auto-filled-after-create"
---
```

`form_id` is written back into the file after `create` or `push` (on first push). This enables upsert semantics.

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

Last page break auto-gets `button: "Absenden"` if no label set.

### Headings and Text

```markdown
## Section Title          → HEADING_2 block
Plain paragraph text.     → TEXT block
**Bold text**             → bold in Tally
*Italic text*             → italic in Tally
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
tally push questionnaire.md --config tally-config.yaml --dry-run
tally push questionnaire.md --config tally-config.yaml

# Export existing form to Markdown
tally export 81GYAY > exported.md

# Download submissions as CSV
tally submissions 81GYAY --format csv > responses.csv

# Update after editing
# (form_id already in frontmatter after first push)
tally push questionnaire.md --config tally-config.yaml

# Round-trip: export, compare, iterate
tally export 81GYAY > exported.md
diff questionnaire.md exported.md

# Generate reference documentation for Claude
tally reference > ~/.claude/references/tally.md
```

## Full Example

```markdown
---
name: "Customer Survey — Acme Corp"
password: "acme-2026"
---

Thank you for participating in our survey. It takes about 10 minutes.

## Your Role

F1: What is your role?
> type: single-choice
> required: true
- CEO / Management
- Department Lead
- Team Lead
- Employee
- Other {other}

F2: How long have you been with the company?
> type: dropdown
- Less than 1 year
- 1-3 years
- 3-10 years
- More than 10 years

---

## Your Daily Work

F3: What is the most annoying routine task?
> type: long-text

F4: Which tools do you use daily?
> type: multi-choice
> max: 5
- Excel
- SAP
- Salesforce
- Custom internal tools
- Other {other}

> show F5 when F4 is_not_empty

F5: Rate the complexity of these tasks
> type: matrix
> hidden: true
> columns: Low, Moderate, High, N/A
- Data entry
- Reporting
- Communication
```
