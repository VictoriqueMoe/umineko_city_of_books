-- +goose Up

CREATE TABLE chat_message_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    thumbnail_url TEXT DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_chat_message_media_message ON chat_message_media(message_id);

-- +goose Down

DROP INDEX IF EXISTS idx_chat_message_media_message;
DROP TABLE IF EXISTS chat_message_media;
