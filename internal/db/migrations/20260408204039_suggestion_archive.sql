-- +goose Up
ALTER TABLE suggestion_resolved ADD COLUMN status TEXT NOT NULL DEFAULT 'done';

-- +goose Down
ALTER TABLE suggestion_resolved DROP COLUMN status;
