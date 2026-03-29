# tally

Bidirectional conversion between Markdown and [Tally.so](https://tally.so) forms. Write your questionnaire in Markdown, push it to Tally with a single command. Pull existing forms back to Markdown.

## Installation

```bash
brew install andinger/tap/tally
```

## Configuration

Create `~/.config/tally/config.yaml`:

```yaml
api:
  token: "tly_your_token_here"

workspace: "your_workspace_id"
logo: "https://storage.tally.so/..."
primary_color: "#A219B1"
```

## Usage

```bash
# Push (upsert) — creates or updates based on form_id in frontmatter
tally push questionnaire.md

# Force create a new form (ignores existing form_id)
tally push questionnaire.md --create

# Dry-run — show JSON payload without calling API
tally push questionnaire.md --dry-run

# Pull — download form as Markdown
tally pull <form-id> > questionnaire.md

# Diff — compare local file with Tally
tally diff questionnaire.md

# Submissions — download responses as CSV
tally submissions <form-id> --format csv > responses.csv

# Prepare — merge global config into frontmatter
tally prepare questionnaire.md

# Config — show config file location
tally config

# Reference — print CLI reference for Claude
tally reference > ~/.claude/references/tally.md
```

## Markdown Format

See `tally reference` for the full Markdown format specification.

## License

MIT
