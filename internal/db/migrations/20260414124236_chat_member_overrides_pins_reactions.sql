-- +goose Up
ALTER TABLE chat_room_members ADD COLUMN nickname TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_room_members ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';

ALTER TABLE chat_messages ADD COLUMN pinned_at DATETIME;
ALTER TABLE chat_messages ADD COLUMN pinned_by TEXT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_chat_messages_pinned ON chat_messages(room_id, pinned_at) WHERE pinned_at IS NOT NULL;

CREATE TABLE chat_message_reactions (
    message_id TEXT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_chat_message_reactions_message ON chat_message_reactions(message_id);

-- +goose Down
DROP INDEX IF EXISTS idx_chat_message_reactions_message;
DROP TABLE IF EXISTS chat_message_reactions;
DROP INDEX IF EXISTS idx_chat_messages_pinned;
ALTER TABLE chat_messages DROP COLUMN pinned_by;
ALTER TABLE chat_messages DROP COLUMN pinned_at;
ALTER TABLE chat_room_members DROP COLUMN avatar_url;
ALTER TABLE chat_room_members DROP COLUMN nickname;
