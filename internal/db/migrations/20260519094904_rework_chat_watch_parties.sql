-- +goose Up

DROP INDEX IF EXISTS idx_chat_watch_party_one_active_per_room;

ALTER TABLE chat_watch_party_sessions ADD COLUMN title TEXT NOT NULL DEFAULT '';

CREATE TABLE chat_watch_party_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_watch_party_sessions(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chat_watch_party_messages_session ON chat_watch_party_messages(session_id, created_at);

-- +goose Down

DROP INDEX IF EXISTS idx_chat_watch_party_messages_session;
DROP TABLE IF EXISTS chat_watch_party_messages;
ALTER TABLE chat_watch_party_sessions DROP COLUMN title;
CREATE UNIQUE INDEX idx_chat_watch_party_one_active_per_room
    ON chat_watch_party_sessions(room_id) WHERE status = 'active';
