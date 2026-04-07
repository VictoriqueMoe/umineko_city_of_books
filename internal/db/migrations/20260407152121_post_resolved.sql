-- +goose Up
CREATE TABLE suggestion_resolved (
    post_id TEXT PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    resolved_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resolved_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS suggestion_resolved;
