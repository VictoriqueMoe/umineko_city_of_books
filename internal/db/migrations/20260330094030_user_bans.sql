-- +goose Up

ALTER TABLE users ADD COLUMN banned_at DATETIME;
ALTER TABLE users ADD COLUMN banned_by TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN ban_reason TEXT DEFAULT '';

-- +goose Down

ALTER TABLE users DROP COLUMN ban_reason;
ALTER TABLE users DROP COLUMN banned_by;
ALTER TABLE users DROP COLUMN banned_at;
