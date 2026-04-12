-- +goose Up
ALTER TABLE mysteries ADD COLUMN paused_at TIMESTAMP NULL;
ALTER TABLE mysteries ADD COLUMN paused_duration_seconds INTEGER NOT NULL DEFAULT 0;
UPDATE mysteries SET paused_at = CURRENT_TIMESTAMP WHERE paused = 1;

-- +goose Down
ALTER TABLE mysteries DROP COLUMN paused_duration_seconds;
ALTER TABLE mysteries DROP COLUMN paused_at;
