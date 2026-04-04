-- +goose Up
CREATE TABLE announcements (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    author_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pinned INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_announcements_created_at ON announcements(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_announcements_created_at;
DROP TABLE IF EXISTS announcements;
