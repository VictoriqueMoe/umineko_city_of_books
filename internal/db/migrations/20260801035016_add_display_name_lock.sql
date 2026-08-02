-- +goose Up
ALTER TABLE users ADD COLUMN display_name_locked BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_users_ip ON users (ip) WHERE ip IS NOT NULL;

-- +goose Down
DROP INDEX idx_users_ip;
ALTER TABLE users DROP COLUMN display_name_locked;
