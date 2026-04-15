-- +goose Up
ALTER TABLE chat_room_members ADD COLUMN timeout_until DATETIME;
ALTER TABLE chat_room_members ADD COLUMN timeout_set_by_staff INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE chat_room_members DROP COLUMN timeout_set_by_staff;
ALTER TABLE chat_room_members DROP COLUMN timeout_until;
