-- +goose Up
ALTER TABLE users ADD COLUMN wide_layout INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN wide_layout;
