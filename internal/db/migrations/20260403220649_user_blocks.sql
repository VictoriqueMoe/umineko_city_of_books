-- +goose Up
CREATE TABLE blocks (
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (blocker_id, blocked_id)
);

CREATE INDEX idx_blocks_blocker_id ON blocks(blocker_id);
CREATE INDEX idx_blocks_blocked_id ON blocks(blocked_id);

-- +goose Down
DROP INDEX IF EXISTS idx_blocks_blocked_id;
DROP INDEX IF EXISTS idx_blocks_blocker_id;
DROP TABLE IF EXISTS blocks;
