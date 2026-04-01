-- +goose Up
CREATE TABLE post_comments (
    id TEXT PRIMARY KEY,
    post_id TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE post_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES post_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('image', 'video')),
    thumbnail_url TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0
);

CREATE INDEX idx_post_comments_post_id ON post_comments(post_id);
CREATE INDEX idx_post_comments_user_id ON post_comments(user_id);
CREATE INDEX idx_post_comment_media_comment_id ON post_comment_media(comment_id);

-- +goose Down
DROP INDEX IF EXISTS idx_post_comment_media_comment_id;
DROP INDEX IF EXISTS idx_post_comments_user_id;
DROP INDEX IF EXISTS idx_post_comments_post_id;
DROP TABLE IF EXISTS post_comment_media;
DROP TABLE IF EXISTS post_comments;
