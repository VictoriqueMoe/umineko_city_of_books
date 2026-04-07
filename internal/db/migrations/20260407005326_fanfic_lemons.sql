-- +goose Up
ALTER TABLE fanfics ADD COLUMN contains_lemons INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_fanfics_contains_lemons ON fanfics(contains_lemons);

-- +goose Down
DROP INDEX IF EXISTS idx_fanfics_contains_lemons;
ALTER TABLE fanfics DROP COLUMN contains_lemons;
