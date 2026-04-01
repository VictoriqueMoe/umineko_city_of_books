-- +goose Up
ALTER TABLE posts ADD COLUMN updated_at DATETIME DEFAULT NULL;

-- +goose Down
