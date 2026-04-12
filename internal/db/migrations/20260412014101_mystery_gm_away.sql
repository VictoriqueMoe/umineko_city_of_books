-- +goose Up
ALTER TABLE mysteries ADD COLUMN gm_away INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE mysteries DROP COLUMN gm_away;
