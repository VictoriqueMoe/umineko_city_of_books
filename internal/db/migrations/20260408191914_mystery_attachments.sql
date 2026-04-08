-- +goose Up
CREATE TABLE mystery_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mystery_id TEXT NOT NULL REFERENCES mysteries(id) ON DELETE CASCADE,
    file_url TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_size INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mystery_attachments_mystery_id ON mystery_attachments(mystery_id);

-- +goose Down
DROP INDEX IF EXISTS idx_mystery_attachments_mystery_id;
DROP TABLE IF EXISTS mystery_attachments;
