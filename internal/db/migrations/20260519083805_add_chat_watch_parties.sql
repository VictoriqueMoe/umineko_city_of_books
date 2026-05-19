-- +goose Up

CREATE TABLE chat_watch_party_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    started_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    controller_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    hyperbeam_session_id TEXT NOT NULL,
    hyperbeam_admin_token TEXT NOT NULL,
    embed_url TEXT NOT NULL,
    start_url TEXT,
    region TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','ended','expired')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    ended_reason TEXT
);
CREATE UNIQUE INDEX idx_chat_watch_party_one_active_per_room
    ON chat_watch_party_sessions(room_id) WHERE status = 'active';
CREATE INDEX idx_chat_watch_party_room ON chat_watch_party_sessions(room_id);

CREATE TABLE chat_watch_party_participants (
    session_id UUID NOT NULL REFERENCES chat_watch_party_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    has_control BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    PRIMARY KEY (session_id, user_id)
);
CREATE INDEX idx_chat_watch_party_participants_active
    ON chat_watch_party_participants(session_id) WHERE left_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_chat_watch_party_participants_active;
DROP TABLE IF EXISTS chat_watch_party_participants;
DROP INDEX IF EXISTS idx_chat_watch_party_room;
DROP INDEX IF EXISTS idx_chat_watch_party_one_active_per_room;
DROP TABLE IF EXISTS chat_watch_party_sessions;
