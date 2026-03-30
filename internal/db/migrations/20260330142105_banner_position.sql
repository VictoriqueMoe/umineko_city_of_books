-- +goose Up

ALTER TABLE users ADD COLUMN banner_position REAL DEFAULT 50;

-- +goose Down

ALTER TABLE users DROP COLUMN banner_position;
