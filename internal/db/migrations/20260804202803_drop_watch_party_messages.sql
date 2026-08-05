-- +goose Up
DROP TABLE IF EXISTS chat_watch_party_messages;
DROP TYPE IF EXISTS chat_watch_party_message_kind;

-- +goose Down
CREATE TYPE chat_watch_party_message_kind AS ENUM ('user', 'system');

CREATE TABLE chat_watch_party_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_watch_party_sessions(id) ON DELETE CASCADE,
    sender_id UUID REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    kind chat_watch_party_message_kind NOT NULL DEFAULT 'user'
);

ALTER TABLE chat_watch_party_messages
    ADD CONSTRAINT chat_watch_party_message_sender_chk
        CHECK (
            (kind = 'user'   AND sender_id IS NOT NULL)
         OR (kind = 'system' AND sender_id IS NULL)
        );

CREATE INDEX idx_chat_watch_party_messages_session ON chat_watch_party_messages(session_id, created_at);
CREATE INDEX idx_chat_watch_party_messages_sender_id ON chat_watch_party_messages (sender_id) WHERE sender_id IS NOT NULL;
