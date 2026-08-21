-- +goose Up
ALTER TABLE journals ADD COLUMN paused_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE journals DROP COLUMN paused_at;
