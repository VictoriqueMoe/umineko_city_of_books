-- +goose Up
CREATE TABLE follows (
    follower_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, following_id)
);

CREATE TABLE post_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX idx_follows_following_id ON follows(following_id);
CREATE INDEX idx_follows_follower_id ON follows(follower_id);
CREATE INDEX idx_post_likes_post_id ON post_likes(post_id);

-- +goose Down
DROP INDEX IF EXISTS idx_post_likes_post_id;
DROP INDEX IF EXISTS idx_follows_follower_id;
DROP INDEX IF EXISTS idx_follows_following_id;
DROP TABLE IF EXISTS post_likes;
DROP TABLE IF EXISTS follows;
