-- +goose Up
ALTER TABLE users ADD COLUMN home_page TEXT DEFAULT 'theories';

-- +goose Down
ALTER TABLE users DROP COLUMN home_page;
