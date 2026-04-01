-- +goose Up
CREATE TABLE embeds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id TEXT NOT NULL,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('post', 'comment')),
    url TEXT NOT NULL,
    embed_type TEXT NOT NULL CHECK (embed_type IN ('link', 'youtube')),
    title TEXT DEFAULT '',
    description TEXT DEFAULT '',
    image TEXT DEFAULT '',
    site_name TEXT DEFAULT '',
    video_id TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_embeds_owner ON embeds(owner_id, owner_type);

-- +goose Down
DROP TABLE embeds;
