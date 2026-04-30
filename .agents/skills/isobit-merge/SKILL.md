---
name: isobit-merge
description: Use when the user asks to merge upstream, pull the latest release, update from origin/main, or resolve merge conflicts on the isobit-main branch.
---

# Isobit Upstream Merge Procedure

This skill merges the latest upstream release tag into `isobit-main` while
preserving all isobit-specific customizations.

## Before You Start

1. Read `.agents/ISOBIT.md` — it lists every isobit customization, the files
   involved, and known gotchas.
2. Confirm you are on `isobit-main`: `git branch --show-current`
3. Confirm the working tree is clean: `git status --short`

## Step 1 — Fetch and Identify the Latest Release

```bash
git fetch origin
git tag --sort=-v:refname | head -5
```

Pick the newest `vX.Y.Z` tag. Confirm it is on `origin/main`:

```bash
git branch -r --contains <tag>
```

## Step 2 — Merge

```bash
git merge <tag>
```

If there are no conflicts, skip to Step 5.

## Step 3 — Resolve Conflicts

For each conflicted file (`git diff --name-only --diff-filter=U`):

1. **Read `.agents/ISOBIT.md`** to check whether the file has isobit
   customizations.
2. Choose a strategy:
   - **File has NO isobit changes**: take upstream entirely
     (`git show <tag>:<file> > <file>`).
   - **File has isobit changes**: take upstream as the base, then re-apply
     isobit customizations on top. Verify by diffing against the pre-merge
     isobit version (`git show HEAD:<file>`).
   - **Styles/themes**: if upstream refactored the `Styles` struct or
     `quickStyle`, update `IsobitStyles()` in `isobit.go` to match the
     new API. Ensure `ThemeForProvider()` still returns `IsobitStyles()`.
3. After resolving each file: `git add <file>`

### Common Pitfalls

- **Renamed struct fields**: upstream frequently restructures nested
  style fields (e.g. `s.Muted` → gone, `s.EditorPromptX` →
  `s.Editor.PromptX`, `s.ResourceGroupTitle` → `s.Resource.Heading`).
  Search for compile errors after resolving styles.
- **New function signatures**: upstream may add/remove parameters
  (e.g. `NewPermissionService` gained a `*db.Queries` arg on isobit).
  Check test files too.
- **Lost method wiring**: when taking upstream's `ui.go`, isobit-only
  methods (`deleteMessage`, `openPermissionRulesDialog`, vi state,
  `sessionPermissions` field) must be re-added. Check the pre-merge
  version for all `func (m *UI)` methods and struct fields that are
  isobit-specific.

## Step 4 — Verify

```bash
go build ./...
go test ./...
```

Fix any compilation errors. Common causes:
- Missing struct fields on `UI` (add them back).
- Old style field names (update to new upstream names).
- Test files calling functions with wrong arity.

## Step 5 — Update ISOBIT.md

If any customization needed adaptation (new field names, moved code,
changed APIs), update `.agents/ISOBIT.md`:
- Adjust file paths if files were renamed/moved.
- Update the "Notes" section with new gotchas.
- Remove entries for features that were adopted upstream.

## Step 6 — Commit

The merge commit is created by `git merge`. If you made additional
fixup changes after resolving conflicts, commit them separately with
the `[isobit]` prefix:

```
[isobit] fix: resolve v0.65.0 merge — adapt styles to new quickStyle API
```

## Quick Reference — Isobit-Specific Files

These files almost always need manual attention during merges:

| File | Customization |
|------|--------------|
| `internal/ui/styles/isobit.go` | Isobit theme definition |
| `internal/ui/styles/themes.go` | `ThemeForProvider` → always isobit |
| `internal/ui/model/ui.go` | Sidebar width, scroll indicator, delete message, vi state, session permissions, permission rules dialog |
| `internal/ui/model/sidebar.go` | Labeled cwd/dataDir display |
| `internal/ui/model/landing.go` | Labeled cwd/dataDir display |
| `internal/ui/common/common.go` | `DefaultCommon` → `IsobitStyles()` |
| `internal/ui/common/elements.go` | `LabeledPath()` helper |
| `internal/app/app.go` | Isobit spinner theme, update nag suppression |
| `internal/cmd/run.go` | Isobit spinner theme |
| `internal/permission/permission.go` | Extra `*db.Queries` param |
| `internal/ui/dialog/permission_rules*.go` | Rules management UI |
| `internal/ui/model/vi.go` | Vi keybindings |
| `internal/agent/tools/hashline_edit.go` | Hashline edit tool |
