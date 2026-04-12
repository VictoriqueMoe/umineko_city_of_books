-- +goose Up
ALTER TABLE chat_rooms ADD COLUMN is_rp INTEGER NOT NULL DEFAULT 0;

CREATE TABLE chat_room_tags (
    room_id TEXT NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (room_id, tag)
);
CREATE INDEX idx_chat_room_tags_tag ON chat_room_tags(tag);
CREATE INDEX idx_chat_rooms_is_rp ON chat_rooms(is_rp) WHERE type = 'group';

-- +goose Down
DROP INDEX IF EXISTS idx_chat_rooms_is_rp;
DROP INDEX IF EXISTS idx_chat_room_tags_tag;
DROP TABLE IF EXISTS chat_room_tags;
ALTER TABLE chat_rooms DROP COLUMN is_rp;
