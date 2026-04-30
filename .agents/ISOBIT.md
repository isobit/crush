# Isobit Branch Customizations

This document tracks all features and changes made on the `isobit-main` branch
that diverge from upstream `origin/main`. When resolving merge conflicts after
pulling a new upstream release, use this list to ensure nothing is lost.

## Conventions

- **Commit prefix**: commits made on `isobit-main` should be prefixed with
  `[isobit]` in the commit message (e.g. `[isobit] feat(ui): add foo`).
- **Update this file**: whenever a new isobit-specific change is made, add or
  update the relevant entry below.

---

## Active Customizations

### Theme — Isobit Styles

- **Files**: `internal/ui/styles/isobit.go`, `internal/ui/styles/themes.go`
- `IsobitStyles()` builds on `quickStyle()` with pure-black background
  (`#000`), blue accent (`#2475f4`), bar-shaped cursors, and custom text
  selection colors.
- `ThemeForProvider()` always returns `IsobitStyles()` so the theme is never
  reset when switching models or providers.
- `DefaultCommon()` in `internal/ui/common/common.go` uses `IsobitStyles()`.
- The CLI spinner in `internal/app/app.go` and `internal/cmd/run.go` also
  uses `IsobitStyles()`.

### Sidebar — Labeled CWD and Data Directory

- **Files**: `internal/ui/model/sidebar.go`, `internal/ui/model/landing.go`,
  `internal/ui/common/elements.go`
- Sidebar and landing screen show `cwd /path/to/dir` and optionally
  `data /path/to/data` (only when the data directory differs from the
  default `<cwd>/.crush`).
- `LabeledPath()` helper in `elements.go` renders `label path` in the
  sidebar's muted style.

### Sidebar — Configurable Width

- **Files**: `internal/ui/model/ui.go`, `internal/config/`
- `cfg.Options.TUI.SidebarWidth` (set via `crush.json`) overrides the
  default sidebar width of 30 columns.

### Chat — Scroll Indicator

- **Files**: `internal/ui/model/ui.go`
- When the chat viewport is not auto-following, a `↓ more` indicator is
  drawn in the bottom-right of the chat area.

### Chat — Delete Messages

- **Files**: `internal/ui/model/ui.go`, `internal/ui/model/keys.go`
- Pressing the `DeleteMessage` key binding on a selected chat message
  deletes it from the session.

### Vi-Style Editor Keybindings

- **Files**: `internal/ui/model/vi.go`
- The text editor supports vi-style navigation (normal/insert mode toggle,
  `hjkl` movement, etc.).

### Hashline Edit Tool

- **Files**: `internal/agent/tools/hashline_edit.go`,
  `internal/agent/tools/hashline_edit.md`, `internal/hashline/`
- Line-addressed editing with 3-char content hashes for verification.
- Dedicated chat renderer in `internal/ui/chat/file.go`
  (`HashlineEditToolMessageItem`).
- Diff view in the permission dialog
  (`internal/ui/dialog/permission_rules_item.go`).

### Permission Rules Management UI

- **Files**: `internal/ui/dialog/permission_rules.go`,
  `internal/ui/dialog/permission_rules_item.go`,
  `internal/ui/model/permissions.go`, `internal/ui/model/ui.go`
- "Manage Permission Rules" command opens a dialog listing session
  permissions and persistent allow-always rules.
- Wired via `dialog.PermissionRulesID` case in `openDialog()` and
  `openPermissionRulesDialog()` method.
- Persistent rules stored via `db.Queries` (`NewPermissionService` takes
  a `*db.Queries` parameter — upstream does not).

### Export and Sessions Commands

- **Files**: `internal/cmd/export.go`, `internal/cmd/session.go`
- `crush export <session-id>` exports a session to a file.
- `crush sessions` lists sessions.

### Environment Variable Config Overrides

- **Files**: `internal/config/options_env.go`, `internal/config/load.go`
- `CRUSH_<UPPER_SNAKE>` environment variables override `crush.json`
  options fields.

### Shell Enhancements

- **Files**: `internal/shell/`
- Shebang/binary/in-process dispatch handler.
- Context-aware `jq` builtin.
- Hook commands run via `shell.Run` with `CRUSH/AGENT` env vars
  propagated.

### Tool Elapsed Time Display

- **Files**: `internal/ui/chat/tools.go`
- Tool calls show elapsed wall-clock time while running.

### Suppress Update Nag

- **Files**: `internal/app/app.go`
- The update-available notification is suppressed on `isobit-main`
  because git-describe versions look like pre-releases to the checker.

### Compact Logo

- **Files**: `internal/ui/logo/`
- Smaller compact logo variant for the sidebar.

---

## Notes

- When upstream refactors the `Styles` struct, `IsobitStyles()` may need
  updating to match new field names or the `quickStyleOpts` palette.
- The `Sidebar.WorkingDir` style is used as a general "muted text" style
  in places where the old `Styles.Muted` field was used (upstream removed
  the top-level `Muted` field).
- The `permission.NewPermissionService` signature has an extra
  `*db.Queries` parameter compared to upstream — tests that call it need
  the fourth `nil` argument.
