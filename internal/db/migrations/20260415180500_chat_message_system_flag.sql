-- +goose Up
ALTER TABLE chat_messages ADD COLUMN is_system INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE chat_messages DROP COLUMN is_system;
