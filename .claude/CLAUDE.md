# tally — Markdown-to-Tally Form Builder

## Build & Test

```bash
go build ./cmd/tally                    # build binary
go test ./... -count=1                  # run all tests
go test ./... -count=1 -coverprofile=c.out && go tool cover -func=c.out  # coverage report
go vet ./...                            # static analysis
```

**Coverage targets:**
- **Core logic (markdown, model, tally compiler/decompiler/types): 100%.** Every new feature or bugfix must include tests. No untested code paths in parser, writer, compiler, decompiler, or schema builder.
- **Config: 90%+.** Filesystem-dependent paths (home dir resolution, legacy fallback) are harder to test but should be covered where feasible.
- **CLI command handlers: best-effort.** The `runXxx` functions are thin orchestration (read file → compile → API call → write output). Test the logic they compose, not the glue itself. Add integration tests with `httptest` servers when the handler contains non-trivial logic (e.g. `writeBackFormID`).
- **cmd/tally/main.go: excluded.** Three lines, just calls cobra.

Run `go tool cover -html=c.out` to visually inspect gaps after changes.

## Architecture

```
cmd/tally/              Entry point (cobra root command)
internal/
  cli/                  Cobra commands (push, pull, diff, submissions, prepare, config, reference)
  config/               2-layer config: global (~/.config/tally/) < frontmatter
  markdown/             Parser (markdown → IR) and writer (IR → markdown)
  model/                Intermediate representation (Form, Page, Block, Question, Conditional)
  tally/                API client, compiler (IR → Tally blocks), decompiler (Tally → IR), types
testdata/               Reference form JSON (QKDAbA) for structure comparison
```

**Data flow:** Markdown → `markdown.Parse()` → IR → `tally.Compile()` → API request → Tally API

**Reverse:** Tally API → `tally.Decompile()` → IR → `markdown.Write()` → Markdown

## Release

GoReleaser via GitHub Actions on tag push. Homebrew tap at `andinger/homebrew-tap`.

```bash
git tag -a v0.x.y -m "message"
git push origin main --follow-tags
# → triggers .github/workflows/release.yml → GoReleaser → brew formula update
brew upgrade andinger/tap/tally
```

After release:

```bash
brew upgrade andinger/tap/tally                   # update local binary
tally reference > ~/.claude/references/tally.md   # regenerate Claude reference
```

## Tally API: Critical Structural Rules

These rules were discovered empirically by comparing CLI-generated forms against manually-created forms in the Tally editor. Violating them produces forms that render for end-users but break in the Tally editor (can't change question types, can't select conditional sources, can't delete blocks).

### Block groupUUID separation

Every question produces a **TITLE block** and **content blocks** (options, inputs, matrix columns/rows). These MUST have **different `groupUuid` values**:

```
TITLE              groupUuid=AAA  groupType=QUESTION          ← title group
MULTIPLE_CHOICE    groupUuid=BBB  groupType=MULTIPLE_CHOICE   ← content group (different!)
```

If they share the same groupUuid, the Tally editor can't recognize the question.

### Conditional field references

The `field.uuid` and `field.blockGroupUuid` in a CONDITIONAL_LOGIC block must **both** be the **content group's `groupUuid`** (the shared groupUuid of option/input blocks). NOT a block UUID, NOT the TITLE's groupUuid.

```json
"field": {
  "uuid": "content-group-uuid",
  "blockGroupUuid": "content-group-uuid",
  "type": "INPUT_FIELD",
  "questionType": "MULTIPLE_CHOICE"
}
```

Verified against form dWlGGV — the UUID is a groupUuid that no block has as its `uuid`.

### Matrix questions

Matrix questions have **no MATRIX container block**. Columns and rows follow the TITLE directly, all sharing one content groupUuid with `groupType=MATRIX`. Labels use `safeHTMLSchema` (not `text`), and each column/row needs `isRequired`.

### safeHTMLSchema format

Rich text uses a segment array — each segment is `["text"]` or `["text", [[style]...]]`:

| Element | Schema | Markdown |
|---------|--------|----------|
| Plain | `["text"]` | `text` |
| Bold | `["text", [["tag","span"],["font-weight","bold"]]]` | `**text**` |
| Italic | `["text", [["tag","span"],["font-style","italic"]]]` | `*text*` |
| Link | `["text", [["href","url"]]]` | `[text](url)` |
| Line break | `["", [["tag","br"]]]` | — |

Nested formatting (e.g. link inside italic) is flattened into separate segments.

### Thank-you page

A text-only last page is not emitted as blocks. Instead, its content goes into `closeMessageTitle` and `closeMessageDescription` settings, which uses Tally's built-in thank-you page.

### Dropdown fields

DROPDOWN_OPTION blocks must NOT include `allowMultiple`, `colorCodeOptions`, `hasBadge`, or `badgeType` — these are MULTIPLE_CHOICE_OPTION-specific. The API rejects them for dropdowns.

### Conditional operator restrictions

`is`, `is_not`, `is_any_of`, `is_not_any_of` are NOT supported for multi-choice (CHECKBOXES) fields. Use `is_empty` / `is_not_empty` instead. The compiler validates this and returns a clear error.

## Go module path

The module path is `github.com/andinger/tally-form-cli` (matches the GitHub repo URL). The binary is named `tally`. All import paths use the full module path — do not change it.
