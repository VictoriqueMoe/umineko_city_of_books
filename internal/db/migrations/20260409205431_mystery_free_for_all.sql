-- +goose Up
ALTER TABLE mysteries ADD COLUMN free_for_all INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE mysteries DROP COLUMN free_for_all;
