-- +goose Up
ALTER TABLE posts ADD COLUMN corner TEXT NOT NULL DEFAULT 'general';
CREATE INDEX idx_posts_corner ON posts(corner);

-- +goose Down
DROP INDEX idx_posts_corner;
