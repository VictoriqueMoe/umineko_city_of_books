-- +goose Up

CREATE TABLE secret_comments (
    id TEXT PRIMARY KEY,
    secret_id TEXT NOT NULL,
    parent_id TEXT REFERENCES secret_comments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);

CREATE INDEX idx_secret_comments_secret_id ON secret_comments(secret_id);
CREATE INDEX idx_secret_comments_parent_id ON secret_comments(parent_id);

CREATE TABLE secret_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES secret_comments(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_secret_comment_likes_comment_id ON secret_comment_likes(comment_id);

CREATE TABLE secret_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES secret_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    thumbnail_url TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_secret_comment_media_comment_id ON secret_comment_media(comment_id);

-- +goose Down

DROP TABLE IF EXISTS secret_comment_media;
DROP TABLE IF EXISTS secret_comment_likes;
DROP TABLE IF EXISTS secret_comments;
