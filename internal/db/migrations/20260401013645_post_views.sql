-- +goose Up
ALTER TABLE posts ADD COLUMN view_count INTEGER DEFAULT 0;

-- +goose Down
ALTER TABLE posts DROP COLUMN view_count;
