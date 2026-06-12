-- +goose Up
ALTER TABLE live_streams ADD COLUMN thumbnail_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE live_streams DROP COLUMN thumbnail_url;
