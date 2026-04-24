-- +goose Up
ALTER TABLE chat_rooms ADD COLUMN archived_at DATETIME;
CREATE INDEX idx_chat_rooms_archived_at ON chat_rooms(archived_at);

-- +goose Down
DROP INDEX IF EXISTS idx_chat_rooms_archived_at;
ALTER TABLE chat_rooms DROP COLUMN archived_at;
