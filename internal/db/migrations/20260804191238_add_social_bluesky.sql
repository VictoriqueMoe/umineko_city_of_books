-- +goose Up
ALTER TABLE users ADD COLUMN social_bluesky TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN social_bluesky;
