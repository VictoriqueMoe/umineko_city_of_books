-- +goose Up
CREATE TABLE fanfic_reading_progress (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fanfic_id TEXT NOT NULL REFERENCES fanfics(id) ON DELETE CASCADE,
    chapter_number INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, fanfic_id)
);

-- +goose Down
DROP TABLE IF EXISTS fanfic_reading_progress;
