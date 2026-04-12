-- +goose Up
ALTER TABLE chat_messages ADD COLUMN reply_to_id TEXT REFERENCES chat_messages(id) ON DELETE SET NULL;
CREATE INDEX idx_chat_messages_reply_to ON chat_messages(reply_to_id);

-- +goose Down
DROP INDEX IF EXISTS idx_chat_messages_reply_to;
ALTER TABLE chat_messages DROP COLUMN reply_to_id;
