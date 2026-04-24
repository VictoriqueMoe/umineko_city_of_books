-- +goose Up
ALTER TABLE users ADD COLUMN locked_at DATETIME;
ALTER TABLE users ADD COLUMN locked_by TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN lock_reason TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_users_locked_at ON users(locked_at);

-- +goose Down
DROP INDEX IF EXISTS idx_users_locked_at;
ALTER TABLE users DROP COLUMN lock_reason;
ALTER TABLE users DROP COLUMN locked_by;
ALTER TABLE users DROP COLUMN locked_at;
