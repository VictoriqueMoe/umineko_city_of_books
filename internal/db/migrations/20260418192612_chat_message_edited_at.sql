-- +goose Up
ALTER TABLE chat_messages ADD COLUMN edited_at DATETIME;

-- +goose Down
ALTER TABLE chat_messages DROP COLUMN edited_at;
