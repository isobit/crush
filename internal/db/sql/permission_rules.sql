-- name: MatchPermissionRule :one
SELECT id FROM permission_rules
WHERE tool_name = ? AND action = ? AND path = ?
LIMIT 1;

-- name: CreatePermissionRule :exec
INSERT OR IGNORE INTO permission_rules (tool_name, action, path, params)
VALUES (?, ?, ?, ?);

-- name: DeletePermissionRule :exec
DELETE FROM permission_rules WHERE id = ?;

-- name: ListPermissionRules :many
SELECT * FROM permission_rules ORDER BY created_at DESC;
