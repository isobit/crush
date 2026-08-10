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
  default sidebar width of 32 columns.

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
- `CRUSH_<UPPER_SNAKE>` environment variables override Options fields.
- Now supports nested fields via `SetFieldByPath` (e.g.
  `CRUSH_TUI_COMPACT_MODE=true` sets `Options.TUI.CompactMode`).

### CLI Config Overrides (`--set`)

- **Files**: `internal/config/cli_overrides.go`,
  `internal/config/cli_overrides_test.go`, `internal/cmd/root.go`,
  `internal/proto/proto.go`, `internal/backend/backend.go`
- `--set key=value` (short: `-o`) persistent flag on the root command
  overrides any Options field for the current session.
- Keys use dotted json-tag paths matching the Options struct (e.g.
  `debug=true`, `tui.compact_mode=true`).
- Uses `SetFieldByPath` — a shared reflection-based mechanism that walks
  nested structs by json tag. The same function powers env var overrides.
- Works in both local and client/server modes (passed through
  `proto.Workspace.SetOverrides`).

### Shell Enhancements

- **Files**: `internal/shell/`
- Shebang/binary/in-process dispatch handler.
- Context-aware `jq` builtin.
- Hook commands run via `shell.Run` with `CRUSH/AGENT` env vars
  propagated.

### Tool Elapsed Time Display

- **Files**: `internal/ui/chat/tools.go`
- Tool calls keep their active spinner and elapsed wall-clock time updating until a result arrives.

### Suppress Update Nag

- **Files**: `internal/app/app.go`
- The update-available notification is suppressed on `isobit-main`
  because git-describe versions look like pre-releases to the checker.

### Compact Logo

- **Files**: `internal/ui/logo/`
- Smaller compact logo variant for the sidebar.

### Glob Tool Timeout

- **Files**: `internal/agent/tools/glob.go`, `internal/config/config.go`
- The glob tool now has a configurable timeout (default 5s) matching the
  existing grep timeout pattern. Prevents runaway CPU on large file trees.
- Configured via `crush.json` under `tools.glob.timeout`.

### Config File Override (`--config`)

- **Files**: `internal/cmd/root.go`, `internal/config/load.go`,
  `internal/config/init.go`, `internal/proto/proto.go`,
  `internal/backend/backend.go`
- `--config /path/to/file.json` persistent flag overrides the entire
  default config lookup chain (global, data, directory-walk configs).
- Can be passed multiple times; files are merged in order (later wins).
- Works in both local and client/server modes (passed through
  `proto.Workspace.ConfigFiles`).
- Uses `config.WithConfigFiles` load option internally.

### Config Profiles (`--profile`/`-p`)

- **Files**: `internal/cmd/root.go`, `internal/config/load.go`,
  `internal/config/store.go`, `internal/config/profile_test.go`,
  `internal/proto/proto.go`, `internal/backend/backend.go`
- `--profile <name>` (short: `-p`) layers `crush.<profile>.json` on top of
  the base global and data configs instead of replacing the chain like
  `--config` does. This keeps directory-level state (the `providers.json`
  cache, shared hooks in the base `crush.json`) while isolating
  profile-specific settings.
- Crucially, all `ScopeGlobal` writes (OAuth tokens, preferred models) are
  redirected to the profile data file
  (`~/.local/share/crush/crush.<profile>.json`) so per-profile MCP
  credentials persist across runs instead of leaking into or reading from
  the shared base config.
- Chain order (low->high): system, base global config, profile config,
  base data config, profile data config, project configs, workspace.
- Profile names are validated (`validateProfileName`) to reject path
  separators and `..`. Mutually exclusive with `--config`.
- The active profile is stored on `ConfigStore.profile` so
  `ReloadFromDisk` keeps the profile layer. Helpers `GlobalConfigProfile`
  and `GlobalConfigDataProfile` derive the paths (honoring
  `CRUSH_GLOBAL_CONFIG`/`CRUSH_GLOBAL_DATA`/XDG overrides).
- Works in both local and client/server modes (passed through
  `proto.Workspace.Profile`). Uses `config.WithProfile` internally.

### MCP Large Output File Spillover

- **Files**: `internal/agent/tools/mcp-tools.go`
- When an MCP tool returns text content larger than `LargeContentThreshold`
  (50 KB), the output is written to a temp file (`mcp-*`) in the
  data directory (`.crush/`) instead of being returned inline.
- The agent can also explicitly request file output by passing
  `__output_file: true` in the tool call parameters. This flag is
  stripped before forwarding to the MCP server.
- The agent receives a short message with the file path and is instructed
  to use `view`/`grep` to process it, keeping the LLM context small.
- On file-write failure, a warning is logged via `slog.Warn` and the
  content falls back to inline delivery.

### Bash Sandbox (bubblewrap)

- **Files**: `internal/shell/sandbox.go`, `internal/shell/sandbox_linux.go`,
  `internal/shell/sandbox_other.go`, `internal/shell/sandbox_test.go`,
  `internal/shell/sandbox_linux_test.go`, `internal/agent/sandbox.go`,
  `internal/agent/sandbox_test.go`, `internal/agent/tools/bash.go`,
  `internal/agent/tools/bash.tpl`, `internal/shell/shell.go`,
  `internal/shell/background.go`, `internal/config/config.go`,
  `internal/ui/dialog/permissions.go`
- On Linux, bash commands can run inside a bubblewrap (`bwrap`) sandbox
  providing filesystem and network isolation via kernel namespaces.
- Configured via `crush.json` `options.sandbox` struct:
  ```json
  { "sandbox": { "mode": "off", "persist": true, "network": false } }
  ```
- `mode`: `"off"` (default) disables sandboxing; `"auto"` enables when
  `bwrap` is on `$PATH` and the platform is Linux; `"on"` fails loudly if
  unavailable.
- `persist` (default true): uses persistent overlayfs — writes outside
  CWD accumulate in `.crush/sandbox/` across commands within a session.
  When false uses `--tmp-overlay` (writes discarded each command).
- When the kernel doesn't support unprivileged overlayfs (detected at
  startup via probe), falls back to `--ro-bind / /` (read-only root).
- The sandbox handler is an `interp.ExecHandler` in the mvdan/sh chain,
  sitting after block checks but before OS exec. Every external command
  invocation is wrapped in `bwrap`.
- Shell I/O is contained too: when the sandbox is active, `newRunner`
  installs `sandboxOpenHandler` as an `interp.OpenHandler` so files the
  shell opens itself (redirections `>`/`>>`/`<`, heredocs) obey the same
  writable-path policy as external commands. mvdan/sh performs those opens
  in-process, so without this a redirection like `echo x > /etc/foo` would
  bypass `bwrap` and hit the real filesystem. Reads are allowed anywhere
  (root is mounted readable); writes are permitted only within CWD, `/tmp`,
  `/dev`, `/proc`, and configured `WritablePaths`, and are otherwise
  rejected with a permission error (stricter than the overlay, which would
  silently discard the write).
- The model can request per-command escape hatches via tool params:
  `sandbox_writable_paths` (real-disk bind mounts that punch through
  the overlay) and `sandbox_network` (allow network). These are
  validated (protected paths rejected) and surfaced in the permission
  prompt (`internal/ui/dialog/permissions.go`).
- Non-Linux platforms get a no-op handler stub.

### Kagi Search Integration

- **Files**: `internal/agent/tools/search_kagi.go`,
  `internal/agent/tools/web_search.go`,
  `internal/agent/tools/web_search_kagi.md.tpl`,
  `internal/config/config.go`, `internal/config/load.go`,
  `internal/agent/coordinator.go`
- `tools.web_search.provider` selects the backend: `"kagi"` or `"duckduckgo"`
  (default). When `"kagi"`, `kagi_api_key` is required.
- `tools.web_search.kagi_api_key` is resolved via shell interpolation and
  validated during config load (fails early if provider is kagi but key is
  missing or resolves to empty).
- `tools.web_search.enable_direct_use` (bool, default false) exposes
  `web_search` directly to the top-level coder agent in addition to
  sub-agents.
- Uses the official `github.com/kagisearch/kagi-openapi-golang` client.
- Sub-agents (e.g. `agentic_fetch`) automatically inherit the backend
  selection since they receive the tool from the same constructor.
- Separate description template (`web_search_kagi.md.tpl`) is shown when
  Kagi is active.

### Numbat Tool

- **Files**: `internal/agent/tools/numbat.go`,
  `internal/agent/tools/numbat.md`,
  `internal/agent/tools/numbat_test.go`,
  `internal/ui/chat/numbat.go`
- Wraps the `numbat` CLI for scientific computation with first-class
  physical dimensions and units (dimensional analysis, unit conversion).
- Passes code via stdin with `--no-config --no-init --color never`.
- 30-second execution timeout.
- Gracefully reports an error if `numbat` is not on `$PATH`.
- Custom chat renderer shows the code prominently in the tool call
  display for human review (one-line summary in header, full code block
  in body when multi-line).

---

## Notes

- When upstream refactors the `Styles` struct, `IsobitStyles()` may need
  updating to match new field names or the `quickStyleOpts` palette.
- The `Sidebar.WorkingDir` style is used as a general "muted text" style
  in places where the old `Styles.Muted` field was used (upstream removed
  the top-level `Muted` field).
- The `permission.NewPermissionService` signature diverges from upstream:
  it is `(workingDir, skip, allowedTools []string, queries *db.Queries)`.
  Upstream added the `allowedTools` param in v0.81.0; isobit adds the
  trailing `*db.Queries`. Tests need both trailing args (`..., nil, nil`).
- Permission resolution (`Grant`, `GrantPersistent`, `GrantAlways`, `Deny`)
  returns `bool` since v0.81.0 and routes through the shared `resolve`
  helper. `GrantAlways` (isobit) passes an `onResolve` callback that
  persists the always-allow rule via `db.Queries`; `GrantPersistent`
  tracks session keys in its callback. The `Workspace` interface mirrors
  these bool returns (`PermissionGrantAlways` included).
- The permission dialog has four options (Allow, Allow for Session, Always
  Allow, Deny) cycling with `% 4`; upstream tests assuming three options
  must be adapted.
- Shell exec: `execHandlerOption(cwd, blockFuncs, sandbox...)` merges
  isobit's sandbox handler with upstream's process-group-isolated base
  handler. `newRunner` keeps the variadic `sandbox ...*SandboxConfig` and
  also applies upstream's `withNonInteractiveEnv`.
- The `toolHeader` renderer takes `*ToolRenderOpts` (not a bare `bool`)
  since v0.81.0; isobit renderers (`chat/file.go`, `chat/numbat.go`) pass
  `opts` directly.
- Config gained upstream's `notification_style` field; `disable_notifications`
  is deprecated but retained. The glob-timeout feature was adopted upstream
  (default 30s there) but isobit keeps its 5s default `ToolGlob`.
- Since v0.87.0, permission diff rendering must support both isobit's
  `hashline_edit` and upstream's `lsp_replace_symbol` tool.
- Since v0.87.0, `RuntimeOverrides` retains isobit's `CLIOverrides` alongside
  upstream's `EnabledChannels`; `proto.Workspace` similarly carries
  `ConfigFiles`, `SetOverrides`, and `Channels`.
- Since v0.87.0, local and client workspaces retain isobit's permission-rule
  and message-deletion methods alongside upstream's question APIs.
- Since v0.87.0, sidebar content uses upstream's scrollable 32-column layout
  while preserving isobit's labeled cwd and optional data-directory rows.
