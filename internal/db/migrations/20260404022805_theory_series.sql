-- +goose Up
ALTER TABLE theories ADD COLUMN series TEXT NOT NULL DEFAULT 'umineko';
CREATE INDEX idx_theories_series ON theories(series);

-- +goose Down
DROP INDEX IF EXISTS idx_theories_series;
ALTER TABLE theories DROP COLUMN series;
