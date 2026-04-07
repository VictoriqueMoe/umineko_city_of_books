-- +goose Up
ALTER TABLE users ADD COLUMN ip TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN ip;
