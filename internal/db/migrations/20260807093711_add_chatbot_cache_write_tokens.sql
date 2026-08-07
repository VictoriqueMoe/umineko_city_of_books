-- +goose Up
ALTER TABLE chatbot_invocations ADD COLUMN cache_write_tokens INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE chatbot_invocations DROP COLUMN cache_write_tokens;
