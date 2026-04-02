-- +goose Up
CREATE TABLE art (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    corner TEXT NOT NULL DEFAULT 'general',
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    image_url TEXT NOT NULL,
    thumbnail_url TEXT DEFAULT '',
    view_count INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);
CREATE INDEX idx_art_user_id ON art(user_id);
CREATE INDEX idx_art_corner ON art(corner);
CREATE INDEX idx_art_created_at ON art(created_at);

CREATE TABLE art_tags (
    art_id TEXT NOT NULL REFERENCES art(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (art_id, tag)
);
CREATE INDEX idx_art_tags_tag ON art_tags(tag);

CREATE TABLE art_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    art_id TEXT NOT NULL REFERENCES art(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, art_id)
);
CREATE INDEX idx_art_likes_art_id ON art_likes(art_id);

CREATE TABLE art_views (
    art_id TEXT NOT NULL REFERENCES art(id) ON DELETE CASCADE,
    viewer_hash TEXT NOT NULL,
    PRIMARY KEY (art_id, viewer_hash)
);

CREATE TABLE art_comments (
    id TEXT PRIMARY KEY,
    art_id TEXT NOT NULL REFERENCES art(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES art_comments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);
CREATE INDEX idx_art_comments_art_id ON art_comments(art_id);
CREATE INDEX idx_art_comments_parent_id ON art_comments(parent_id);

CREATE TABLE art_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES art_comments(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, comment_id)
);
CREATE INDEX idx_art_comment_likes_comment_id ON art_comment_likes(comment_id);

CREATE TABLE art_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES art_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('image', 'video')),
    thumbnail_url TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0
);
CREATE INDEX idx_art_comment_media_comment_id ON art_comment_media(comment_id);

-- +goose Down
DROP TABLE IF EXISTS art_comment_media;
DROP TABLE IF EXISTS art_comment_likes;
DROP TABLE IF EXISTS art_comments;
DROP TABLE IF EXISTS art_views;
DROP TABLE IF EXISTS art_likes;
DROP TABLE IF EXISTS art_tags;
DROP TABLE IF EXISTS art;
