---
name: isobit-contrib
description: Use when making any code change, commit, or new feature on the isobit-main branch. Ensures commits follow the [isobit] prefix convention and ISOBIT.md stays up to date.
---

# Isobit Contribution Workflow

This skill ensures that work on the `isobit-main` branch follows the project
conventions documented in `.agents/ISOBIT.md`.

## Before Starting Work

1. Read `.agents/ISOBIT.md` to understand existing isobit customizations.
2. Confirm you are on the `isobit-main` branch (`git branch --show-current`).

## Commit Convention

All commits on `isobit-main` MUST use this prefix format:

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

- [ ] Branch is `isobit-main`
- [ ] Commit message has `[isobit]` prefix
- [ ] `.agents/ISOBIT.md` is updated if the change affects an isobit customization
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
