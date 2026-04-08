-- +goose Up
ALTER TABLE posts ADD COLUMN shared_content_id TEXT DEFAULT NULL;
ALTER TABLE posts ADD COLUMN shared_content_type TEXT DEFAULT NULL;

CREATE TABLE share_counts (
    content_id TEXT NOT NULL,
    content_type TEXT NOT NULL,
    share_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (content_id, content_type)
);

CREATE INDEX idx_posts_shared_content ON posts(shared_content_id, shared_content_type);

-- +goose Down
DROP INDEX IF EXISTS idx_posts_shared_content;
DROP TABLE IF EXISTS share_counts;
