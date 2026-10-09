-- +goose Up
ALTER TABLE sessions ADD COLUMN total_input_tokens INTEGER DEFAULT 0 NOT NULL;
ALTER TABLE sessions ADD COLUMN total_cached_input_tokens INTEGER DEFAULT 0 NOT NULL;
ALTER TABLE sessions ADD COLUMN total_output_tokens INTEGER DEFAULT 0 NOT NULL;

-- +goose Down
ALTER TABLE sessions DROP COLUMN total_output_tokens;
ALTER TABLE sessions DROP COLUMN total_cached_input_tokens;
ALTER TABLE sessions DROP COLUMN total_input_tokens;
