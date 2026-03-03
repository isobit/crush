Edits a file using line-addressed operations with hash verification.

Lines are referenced by `LINE#HASH` where `LINE` is a 1-based line number and `HASH` is a 3-character hex content hash from `view` output (e.g., `23#a4f`).

## Parameters

- `path` (required): File path to edit.
- `edits`: Array of edit operations. Each operation has:
  - `op` (required): `"replace"`, `"append"`, or `"prepend"`.
  - `pos`: Start line reference (e.g., `"10#a4f"`). Required for `replace`, optional anchor for `append`/`prepend`.
  - `end`: End line reference for range replace (e.g., `"15#b2c"`). Only used with `replace`.
  - `lines`: New content lines as `string[]`, a single `string`, or `null` (delete).
- `delete`: If true, deletes the file (no `edits` needed).
- `move`: New path to move/rename the file (no `edits` needed).

## Operations

### replace
Replace one or more lines. Single line: only `pos`. Range: `pos` + `end` (inclusive).
- `lines: null` deletes the line(s).
- `lines: ["new content"]` replaces with new content.

### append
Insert lines after an anchor line (`pos`). Without `pos`, appends to end of file.

### prepend
Insert lines before an anchor line (`pos`). Without `pos`, prepends to beginning of file.

## Examples

Replace a single line:
```json
{"op": "replace", "pos": "10#a4f", "lines": ["    return nil"]}
```

Delete lines 10-15:
```json
{"op": "replace", "pos": "10#a4f", "end": "15#b2c", "lines": null}
```

Insert after line 20:
```json
{"op": "append", "pos": "20#c3d", "lines": ["    // new comment", "    x := 1"]}
```

Append to end of file:
```json
{"op": "append", "lines": ["// EOF"]}
```

## Best Practices

- Always read the file first with `view` to get fresh line hashes.
- Re-read on hash mismatch errors — the file has changed.
- Anchor on structural lines (closing braces, function signatures), not blank lines.
- Batch all edits for a file into one `hashline_edit` call.
- Edits are applied bottom-up; you can reference original line numbers.
- Lines in `lines` content should NOT include the `LINE#HASH|` prefix — provide raw content.

## Error Recovery

- **Hash mismatch**: The file changed since you read it. Re-read with `view` and retry.
- **Out of bounds**: Line number exceeds file length. Re-read to get current line count.
- **Overlapping ranges**: Two edits touch the same lines. Combine them into one operation.
- **No-op**: The edit would not change the file. Re-read to verify current content.
