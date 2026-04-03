-- +goose Up
ALTER TABLE reports ADD COLUMN resolution_comment TEXT DEFAULT '';

-- +goose Down
ALTER TABLE reports DROP COLUMN resolution_comment;
