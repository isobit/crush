-- +goose Up
ALTER TABLE permission_rules ADD COLUMN params TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE permission_rules DROP COLUMN params;
