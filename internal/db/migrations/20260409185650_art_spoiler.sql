-- +goose Up
ALTER TABLE art ADD COLUMN is_spoiler INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE art DROP COLUMN is_spoiler;
