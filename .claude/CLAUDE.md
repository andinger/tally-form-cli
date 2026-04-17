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

Matrix questions **must** include a MATRIX container block between the TITLE and the MATRIX_COLUMN/MATRIX_ROW blocks. All matrix blocks (container + columns + rows) share **one** phantom `groupUuid` and use `groupType: MATRIX` — including the container itself, which has `groupType: MATRIX` (not `QUESTION`, despite what the OpenAPI schema says — the schema is wrong here, verified against editor-produced forms).

```
TITLE              groupUuid=T      groupType=QUESTION
MATRIX             groupUuid=X      groupType=MATRIX        ← container, payload has isFirst/isLast/index
MATRIX_COLUMN      groupUuid=X      groupType=MATRIX
MATRIX_COLUMN      groupUuid=X      groupType=MATRIX
MATRIX_ROW         groupUuid=X      groupType=MATRIX
MATRIX_ROW         groupUuid=X      groupType=MATRIX
```

The container's payload must carry `isFirst=false`, `isLast=true`, and `index=len(columns)` alongside `isRequired`. Without the container block, the Tally editor marks the form as dirty on load and blocks delete operations on other questions until the matrix is removed. Labels use `safeHTMLSchema` (not `text`), and each column/row needs `isRequired`.

### Hint text on questions

Hints for text-based inputs (`TEXTAREA`, `INPUT_TEXT`, `INPUT_NUMBER`, `INPUT_EMAIL`, `INPUT_PHONE_NUMBER`, `INPUT_LINK`, `INPUT_DATE`, `INPUT_TIME`) must be folded into the input's `placeholder` field — **never** emitted as a separate TEXT block between the TITLE and the input. A TEXT orphan between question blocks breaks the question group: deleting a conditional that references the question cascades into deleting the question itself. Choice/matrix/scale/rating questions have no placeholder equivalent, so the compiler warns and drops hints for those.

### Input payload restrictions

`hasMinCharacters` and `hasMaxCharacters` are **only** accepted on `TEXTAREA` and `INPUT_TEXT`. All other input types (`INPUT_EMAIL`, `INPUT_NUMBER`, `INPUT_PHONE_NUMBER`, `INPUT_LINK`, `INPUT_DATE`, `INPUT_TIME`, `FILE_UPLOAD`, `SIGNATURE`) reject them with a 400 VALIDATION error.

### LinearScale payload

Use `start` / `end` (not `startNumber` / `endNumber`). Left/right labels require matching boolean flags: set `hasLeftLabel: true` whenever `leftLabel` is set, same for `hasRightLabel` / `rightLabel`.

### Conditional value shape

Comparisons come in two flavors:
- **ANY_OF** (`IS_ANY_OF`, `IS_NOT_ANY_OF`) always expect a JSON **array** of UUIDs, even with a single value.
- **Scalar** (`IS`, `IS_NOT`, `CONTAINS`, `DOES_NOT_CONTAIN`) expect a **string** value.

Sending a string to an ANY_OF comparison fails with `value type is not allowed for this comparison`.

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
