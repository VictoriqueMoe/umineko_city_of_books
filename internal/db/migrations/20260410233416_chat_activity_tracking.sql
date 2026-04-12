-- +goose Up

ALTER TABLE chat_rooms ADD COLUMN last_message_at DATETIME;
ALTER TABLE chat_room_members ADD COLUMN last_read_at DATETIME;

CREATE INDEX idx_chat_rooms_last_message_at ON chat_rooms(last_message_at DESC);

UPDATE chat_rooms
SET last_message_at = (
    SELECT MAX(created_at) FROM chat_messages WHERE room_id = chat_rooms.id
);

UPDATE chat_room_members SET last_read_at = CURRENT_TIMESTAMP;

-- +goose Down

DROP INDEX IF EXISTS idx_chat_rooms_last_message_at;
ALTER TABLE chat_room_members DROP COLUMN last_read_at;
ALTER TABLE chat_rooms DROP COLUMN last_message_at;
