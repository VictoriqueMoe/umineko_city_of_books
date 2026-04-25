-- +goose Up
CREATE TABLE sidebar_last_visited (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    visited_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, key)
);

-- +goose Down
DROP TABLE IF EXISTS sidebar_last_visited;
