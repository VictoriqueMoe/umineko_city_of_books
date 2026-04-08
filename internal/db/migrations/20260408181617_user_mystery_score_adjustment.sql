-- +goose Up
ALTER TABLE users ADD COLUMN mystery_score_adjustment INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN mystery_score_adjustment;
