-- +goose Up
DROP INDEX IF EXISTS idx_chat_rooms_system_kind;
CREATE UNIQUE INDEX idx_chat_rooms_system_kind ON chat_rooms(system_kind) WHERE system_kind IS NOT NULL AND system_kind <> 'live_stream';
CREATE INDEX idx_chat_rooms_live_stream ON chat_rooms(system_kind) WHERE system_kind = 'live_stream';

-- +goose Down
DROP INDEX IF EXISTS idx_chat_rooms_live_stream;
DROP INDEX IF EXISTS idx_chat_rooms_system_kind;
CREATE UNIQUE INDEX idx_chat_rooms_system_kind ON chat_rooms(system_kind) WHERE system_kind IS NOT NULL;
