-- +goose Up

ALTER TABLE users ADD COLUMN episode_progress INTEGER DEFAULT 0;

-- +goose Down

ALTER TABLE users DROP COLUMN episode_progress;
