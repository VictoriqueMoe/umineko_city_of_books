-- +goose Up
CREATE TABLE galleries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    cover_art_id TEXT REFERENCES art(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);
CREATE INDEX idx_galleries_user_id ON galleries(user_id);

ALTER TABLE art ADD COLUMN gallery_id TEXT REFERENCES galleries(id) ON DELETE SET NULL;
CREATE INDEX idx_art_gallery_id ON art(gallery_id);

-- +goose Down
DROP INDEX IF EXISTS idx_art_gallery_id;
ALTER TABLE art DROP COLUMN gallery_id;
DROP TABLE IF EXISTS galleries;
