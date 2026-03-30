-- +goose Up

CREATE TABLE invites (
    code TEXT PRIMARY KEY,
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invites_created_by ON invites(created_by);

-- +goose Down

DROP TABLE IF EXISTS invites;
