# tally-form-cli

Bidirectional conversion between Markdown and [Tally.so](https://tally.so) forms. Write your questionnaire in Markdown, push it to Tally with a single command. Export existing forms back to Markdown.

## Installation

```bash
brew install andinger/tap/tally-form-cli
```

## Configuration

Create `~/.config/tally-form-cli/config.yaml`:

```yaml
api:
  token: "tly_your_token_here"
  base_url: "https://api.tally.so"

workspace: "your_workspace_id"
```

## Usage

```bash
# Push (upsert) — creates or updates based on form_id in frontmatter
tally-form-cli push questionnaire.md --config tally-config.yaml

# Dry-run — show JSON payload without calling API
tally-form-cli push questionnaire.md --dry-run

# Create — always creates a new form
tally-form-cli create questionnaire.md

# Update — update existing form
tally-form-cli update <form-id> questionnaire.md

# Export — download form as Markdown
tally-form-cli export <form-id> > questionnaire.md

# Submissions — download responses as CSV
tally-form-cli submissions <form-id> --format csv > responses.csv
```

## Markdown Format

```markdown
---
name: "KI-Hebel-Check — Mustermann GmbH"
password: "muster-check"
---

Intro text here.

## Section Title

F1: In which role are you?
> type: single-choice
> required: true
- Management
- Team Lead
- Other {other}

F2: Which tools do you use?
> type: multi-choice
> max: 3
- Excel
- Slack

> show F3 when F1 is "Management"

F3: Strategic priorities?
> type: long-text
> hidden: true

---
> button: "Next page"

## Page 2

F4: Rate document complexity
> type: matrix
> columns: Low, Moderate, High, N/A
- Reports
- Protocols
```

## License

MIT
