-- +goose Up
ALTER TABLE chat_room_members ADD COLUMN nickname_locked INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE chat_room_members DROP COLUMN nickname_locked;
