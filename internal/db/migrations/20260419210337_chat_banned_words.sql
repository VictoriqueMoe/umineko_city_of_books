-- +goose Up

CREATE TABLE chat_banned_words (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('global','room')),
    room_id TEXT REFERENCES chat_rooms(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    match_mode TEXT NOT NULL CHECK (match_mode IN ('substring','whole_word','regex')),
    case_sensitive INTEGER NOT NULL DEFAULT 0,
    action TEXT NOT NULL CHECK (action IN ('delete','kick')),
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK ((scope = 'global' AND room_id IS NULL) OR (scope = 'room' AND room_id IS NOT NULL))
);

CREATE INDEX idx_chat_banned_words_scope ON chat_banned_words(scope);
CREATE INDEX idx_chat_banned_words_room ON chat_banned_words(room_id);

-- +goose Down

DROP TABLE IF EXISTS chat_banned_words;
