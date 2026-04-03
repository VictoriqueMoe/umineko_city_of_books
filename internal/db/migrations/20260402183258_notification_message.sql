-- +goose Up
ALTER TABLE notifications ADD COLUMN message TEXT DEFAULT '';

-- +goose Down
ALTER TABLE notifications DROP COLUMN message;
