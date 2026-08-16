---
name: isobit-contrib
description: Use when making any code change, commit, or new feature in the Isobit fork. Ensures commits follow the [isobit] prefix convention and ISOBIT.md stays up to date.
---

# Isobit Contribution Workflow

This skill ensures that work in an Isobit fork follows the project conventions
documented in `.agents/ISOBIT.md`.

## Before Starting Work

1. Read `.agents/ISOBIT.md` to understand existing Isobit customizations.
2. Confirm the repository has an Isobit fork remote and a Charmbracelet
   upstream remote:

   ```bash
   git remote -v
   ```

   Remote names and the local branch name are not prescribed.

## Commit Convention

All commits in the Isobit fork MUST use this prefix format:

```
[isobit] <type>(<scope>): <description>
```

Examples:
- `[isobit] feat(ui): add dark mode toggle`
- `[isobit] fix(sidebar): restore cwd label after merge`
- `[isobit] refactor(styles): simplify isobit palette`

The `<type>` follows standard semantic commits (`feat`, `fix`, `refactor`,
`chore`, `docs`, `sec`, etc.).

Exception: merge commits (e.g. `Merge tag 'v0.65.0' into isobit-main`) do
NOT get the prefix.

## Updating ISOBIT.md

After making a change that adds, modifies, or removes an isobit-specific
feature, update `.agents/ISOBIT.md`:

- **New feature**: add a new `### Section` under "Active Customizations"
  with the relevant files and a brief description.
- **Modified feature**: update the existing section to reflect the change.
- **Removed feature**: delete the section.
- **Merge fix**: if a merge resolution required adapting an existing
  customization to new upstream APIs (e.g. renamed style fields), update
  the "Notes" section with any new gotchas.

Include the ISOBIT.md update in the same commit as the code change.

## Checklist

- [ ] Repository is the Isobit fork
- [ ] Commit message has `[isobit]` prefix
- [ ] `.agents/ISOBIT.md` is updated if the change affects an isobit customization
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
