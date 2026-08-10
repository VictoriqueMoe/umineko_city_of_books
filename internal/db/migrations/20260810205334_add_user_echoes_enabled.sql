-- +goose Up
ALTER TABLE users ADD COLUMN echoes_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE users DROP COLUMN echoes_enabled;
