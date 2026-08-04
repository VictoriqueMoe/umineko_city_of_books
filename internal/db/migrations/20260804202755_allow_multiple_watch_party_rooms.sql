-- +goose Up
DROP INDEX IF EXISTS idx_chat_rooms_system_kind;
CREATE UNIQUE INDEX idx_chat_rooms_system_kind ON chat_rooms(system_kind) WHERE system_kind IS NOT NULL AND system_kind NOT IN ('live_stream', 'watch_party');
CREATE INDEX idx_chat_rooms_watch_party ON chat_rooms(system_kind) WHERE system_kind = 'watch_party';

-- +goose Down
DROP INDEX IF EXISTS idx_chat_rooms_watch_party;
DROP INDEX IF EXISTS idx_chat_rooms_system_kind;
CREATE UNIQUE INDEX idx_chat_rooms_system_kind ON chat_rooms(system_kind) WHERE system_kind IS NOT NULL AND system_kind <> 'live_stream';
