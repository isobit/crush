-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS permission_rules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tool_name  TEXT NOT NULL,
    action     TEXT NOT NULL,
    path       TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    UNIQUE(tool_name, action, path)
);
CREATE INDEX IF NOT EXISTS idx_permission_rules_lookup
    ON permission_rules (tool_name, action, path);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_permission_rules_lookup;
DROP TABLE IF EXISTS permission_rules;
-- +goose StatementEnd
