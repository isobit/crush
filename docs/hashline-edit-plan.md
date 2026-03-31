# Hashline Edit Tool — Implementation Plan

## Design Summary

Hashline is an **optional mode** toggled by `Options.HashlineEdit`. When enabled:
- `view` emits `LINE#HASH|` prefixed output
- `hashline_edit` replaces `edit`/`multiedit` entirely (those tools are removed from `AllowedTools`)
- `write` remains unchanged (full-file replacement is orthogonal)
- System prompt gains a hashline-specific section

When disabled (default): no changes to current behavior.

---

## 1. Core Library: `internal/hashline/`

### `hashline.go` (~180 lines)

| Function | Description |
|---|---|
| `NormalizeLine(s string) string` | `TrimSpace` → collapse internal whitespace → lowercase |
| `ComputeHash(line string) string` | Normalize → FNV-1a (`hash/fnv` stdlib) → truncate to 3 hex chars |
| `FormatLine(lineNum int, hash, content string) string` | `"LINE#HASH\| CONTENT"` |
| `FormatLines(lines []string, startLine int) string` | Batch format for view output |
| `ParseRef(ref string) (line int, hash string, err error)` | Parse `"23#a4f"` → `(23, "a4f", nil)` |
| `HashFile(lines []string) []LineRef` | Compute `LineRef` for every line |
| `ValidateRefs(lines []string, refs ...string) []Mismatch` | Batch-validate refs, return **all** mismatches |
| `type LineRef struct` | `Num int`, `Hash string`, `Content string` |
| `type Mismatch struct` | `Line int`, `ExpectedHash string`, `ActualHash string`, `CurrentContent string`, `Reference string` |

**Hash params:** FNV-1a, 3 hex chars (12 bits, 4096 buckets). No new dependencies.

### `hashline_test.go` (~250 lines)

- Determinism, normalization edge cases (Unicode, BOM, tabs, CRLF)
- `ParseRef` valid/invalid/malformed inputs
- `ValidateRefs` match/mismatch/out-of-bounds
- Empty file, single-line, very long lines
- Collision rate sanity check on realistic Go source

---

## 2. Config: Feature Toggle

### `internal/config/config.go` — `Options`

```go
HashlineEdit *bool `json:"hashline_edit,omitempty" jsonschema:"..."`
```

`*bool`, defaults to `false` via `ptrValOr(c.Options.HashlineEdit, false)`.

### `allToolNames()`

Add `"hashline_edit"` to the full tool list.

### `SetupAgents()`

When `HashlineEdit` is true:
- Add `"hashline_edit"` to coder agent's `AllowedTools`
- **Remove** `"edit"` and `"multiedit"` from `AllowedTools`

When false (default):
- `"hashline_edit"` excluded, `"edit"`/`"multiedit"` included (current behavior)

---

## 3. View Tool: Conditional Hash Output

### `internal/agent/tools/view.go`

`NewViewTool` gains a `hashlineMode bool` parameter.

When true: replace `addLineNumbers()` with `hashline.FormatLines()`.
Output changes from `     1|content` to `1#a4f| content`.

When false: no change.

### Wiring in `coordinator.go`

```go
tools.NewViewTool(..., ptrValOr(c.cfg.Options.HashlineEdit, false), c.cfg.Options.SkillsPaths...)
```

---

## 4. Hashline Edit Tool: `internal/agent/tools/`

### `hashline_edit.go` (~400 lines)

**Tool name:** `"hashline_edit"`

**Params:**
```go
type HashlineEditParams struct {
    Path   string        `json:"path"`
    Edits  []HashlineOp  `json:"edits"`
    Delete bool          `json:"delete,omitempty"`
    Move   string        `json:"move,omitempty"`
}

type HashlineOp struct {
    Op    string          `json:"op"`       // "replace", "append", "prepend"
    Pos   string          `json:"pos,omitempty"`
    End   string          `json:"end,omitempty"`
    Lines json.RawMessage `json:"lines"`    // []string | string | null
}
```

`Lines` uses `json.RawMessage` → custom normalizer → `*[]string` (nil = delete).

**Execution flow:**
1. Resolve path via `filepathext.SmartJoin`
2. Handle `Delete`/`Move` file-level ops
3. Read file into `[]string`
4. **Validation phase** (all-or-nothing):
   - Parse all `pos`/`end` refs
   - Bounds-check line numbers
   - Hash-verify via `hashline.ValidateRefs`
   - Reject overlapping ranges
   - Collect all errors → abort with structured error if any
5. Strip hashline prefixes from `Lines` content (spec §8.3)
6. Sort edits descending by anchor line (bottom-up)
7. Apply ops on `[]string` slice
8. Join → diff → no-op check → permission → write → history → record read → LSP notify
9. Return `EditResponseMetadata` + diagnostics

**File creation:** Only anchorless `append`/`prepend` allowed. Any op with `pos`/`end` → error.

**Errors** returned as structured text via `fantasy.NewTextErrorResponse`:
- Hash mismatch → all stale lines with current content
- No-op → re-read advice
- Overlapping ranges → conflicting edit details

### `hashline_edit.md` (~120 lines)

LLM tool description: `LINE#HASH` format, all three operations with examples, anchoring best practices, error recovery guidance.

### `hashline_edit_test.go` (~350 lines)

- Single-line replace, range replace
- Append/prepend with and without anchor
- Delete (null lines), file creation
- Hash mismatch, no-op, overlapping range rejection
- Bottom-up ordering, adjacent-line edits
- CRLF, empty file, malformed refs, out-of-bounds

---

## 5. System Prompt: Conditional Section

### `internal/agent/prompt/prompt.go` — `PromptDat`

Add: `HashlineEdit bool`

Populated from `ptrValOr(cfg.Options.HashlineEdit, false)`.

### `internal/agent/templates/coder.md.tpl`

Conditional block gated on `{{if .HashlineEdit}}`:

```
{{if .HashlineEdit}}
<hashline_editing>
The view tool outputs lines with content hashes: `LINE#HASH| CONTENT`.
Use `hashline_edit` for all file mutations. Reference lines by `LINE#HASH`.
- Read file first to get fresh hashes
- Re-read on hash mismatch errors
- Prefer append/prepend for insertions, replace for modifications
- Anchor on structural boundaries (closing braces), not whitespace lines
- Batch all operations for a file in one hashline_edit call
</hashline_editing>
{{end}}
```

Update `<editing_files>` tool list conditionally:
```
{{if .HashlineEdit}}
- `hashline_edit` - Line-addressed editing with hash verification
{{else}}
- `edit` - Single find/replace in a file
- `multiedit` - Multiple find/replace operations in one file
{{end}}
- `write` - Create/overwrite entire file
```

---

## 6. Coordinator Wiring

### `internal/agent/coordinator.go` — `buildTools()`

```go
hashlineMode := ptrValOr(c.cfg.Options.HashlineEdit, false)

// Conditionally add hashline_edit OR edit/multiedit
if hashlineMode {
    allTools = append(allTools,
        tools.NewHashlineEditTool(c.lspManager, c.permissions, c.history, c.filetracker, c.cfg.WorkingDir()),
    )
} else {
    allTools = append(allTools,
        tools.NewEditTool(c.lspManager, c.permissions, c.history, c.filetracker, c.cfg.WorkingDir()),
        tools.NewMultiEditTool(c.lspManager, c.permissions, c.history, c.filetracker, c.cfg.WorkingDir()),
    )
}

// View always registered, but with hashline flag
allTools = append(allTools,
    tools.NewViewTool(..., hashlineMode, c.cfg.Options.SkillsPaths...),
)
```

The `AllowedTools` filter in `SetupAgents` provides the second gate — `edit`/`multiedit` won't be in `AllowedTools` when hashline is on, and `hashline_edit` won't be when it's off.

---

## 7. File Inventory

| File | Action | Est. Lines |
|---|---|---|
| `internal/hashline/hashline.go` | **Create** | ~180 |
| `internal/hashline/hashline_test.go` | **Create** | ~250 |
| `internal/agent/tools/hashline_edit.go` | **Create** | ~400 |
| `internal/agent/tools/hashline_edit.md` | **Create** | ~120 |
| `internal/agent/tools/hashline_edit_test.go` | **Create** | ~350 |
| `internal/agent/tools/view.go` | **Modify** | ~20 lines |
| `internal/agent/coordinator.go` | **Modify** | ~15 lines |
| `internal/config/config.go` | **Modify** | ~10 lines |
| `internal/agent/prompt/prompt.go` | **Modify** | ~3 lines |
| `internal/agent/templates/coder.md.tpl` | **Modify** | ~20 lines |

---

## 8. Open Question

**Config key naming:** `hashline_edit` (matches tool name, explicit) vs `hashline` (shorter, acknowledges it also affects view). Leaning `hashline_edit` since the primary behavior change is the edit tool swap.
