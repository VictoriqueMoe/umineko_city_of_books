-- +goose Up
ALTER TABLE chat_room_members ADD COLUMN left_at DATETIME;
CREATE INDEX idx_chat_room_members_active ON chat_room_members(room_id, user_id) WHERE left_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_chat_room_members_active;
ALTER TABLE chat_room_members DROP COLUMN left_at;
