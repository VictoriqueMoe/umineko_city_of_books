-- +goose Up
CREATE TABLE mystery_comments (
    id TEXT PRIMARY KEY,
    mystery_id TEXT NOT NULL REFERENCES mysteries(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES mystery_comments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);

CREATE INDEX idx_mystery_comments_mystery_id ON mystery_comments(mystery_id);
CREATE INDEX idx_mystery_comments_parent_id ON mystery_comments(parent_id);

CREATE TABLE mystery_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES mystery_comments(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_mystery_comment_likes_comment_id ON mystery_comment_likes(comment_id);

CREATE TABLE mystery_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES mystery_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    thumbnail_url TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_mystery_comment_media_comment_id ON mystery_comment_media(comment_id);

-- +goose Down
DROP TABLE IF EXISTS mystery_comment_media;
DROP TABLE IF EXISTS mystery_comment_likes;
DROP TABLE IF EXISTS mystery_comments;
