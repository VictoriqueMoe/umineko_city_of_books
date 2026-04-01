-- +goose Up
ALTER TABLE post_comments ADD COLUMN parent_id TEXT DEFAULT NULL REFERENCES post_comments(id) ON DELETE CASCADE;
ALTER TABLE post_comments ADD COLUMN updated_at DATETIME DEFAULT NULL;

CREATE INDEX idx_post_comments_parent_id ON post_comments(parent_id);

CREATE TABLE post_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES post_comments(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_post_comment_likes_comment_id ON post_comment_likes(comment_id);

-- +goose Down
DROP TABLE post_comment_likes;
DROP INDEX idx_post_comments_parent_id;
