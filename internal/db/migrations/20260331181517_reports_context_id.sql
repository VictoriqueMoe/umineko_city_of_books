-- +goose Up
ALTER TABLE reports ADD COLUMN context_id TEXT DEFAULT '';

-- +goose Down
ALTER TABLE reports DROP COLUMN context_id;
