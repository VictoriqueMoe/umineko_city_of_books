-- +goose Up
ALTER TABLE chat_message_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE chat_message_media DROP COLUMN is_spoiler;
