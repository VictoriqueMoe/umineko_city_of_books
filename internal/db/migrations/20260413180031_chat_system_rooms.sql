-- +goose Up
ALTER TABLE chat_rooms ADD COLUMN is_system INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chat_rooms ADD COLUMN system_kind TEXT;
CREATE UNIQUE INDEX idx_chat_rooms_system_kind ON chat_rooms(system_kind) WHERE system_kind IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_chat_rooms_system_kind;
ALTER TABLE chat_rooms DROP COLUMN system_kind;
ALTER TABLE chat_rooms DROP COLUMN is_system;
