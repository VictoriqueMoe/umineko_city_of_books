-- +goose Up
ALTER TABLE art ADD COLUMN art_type TEXT NOT NULL DEFAULT 'drawing';

-- +goose Down
ALTER TABLE art DROP COLUMN art_type;
