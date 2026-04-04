-- +goose Up
CREATE TABLE mysteries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    difficulty TEXT NOT NULL DEFAULT 'medium',
    solved INTEGER NOT NULL DEFAULT 0,
    winner_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    solved_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mysteries_created_at ON mysteries(created_at DESC);
CREATE INDEX idx_mysteries_user_id ON mysteries(user_id);

CREATE TABLE mystery_clues (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mystery_id TEXT NOT NULL REFERENCES mysteries(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    truth_type TEXT NOT NULL DEFAULT 'red',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_mystery_clues_mystery_id ON mystery_clues(mystery_id);

CREATE TABLE mystery_attempts (
    id TEXT PRIMARY KEY,
    mystery_id TEXT NOT NULL REFERENCES mysteries(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES mystery_attempts(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    is_winner INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mystery_attempts_mystery_id ON mystery_attempts(mystery_id);

CREATE TABLE mystery_attempt_votes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attempt_id TEXT NOT NULL REFERENCES mystery_attempts(id) ON DELETE CASCADE,
    value INTEGER NOT NULL CHECK (value IN (1, -1)),
    PRIMARY KEY (user_id, attempt_id)
);

-- +goose Down
DROP TABLE IF EXISTS mystery_attempt_votes;
DROP TABLE IF EXISTS mystery_attempts;
DROP TABLE IF EXISTS mystery_clues;
DROP INDEX IF EXISTS idx_mysteries_created_at;
DROP INDEX IF EXISTS idx_mysteries_user_id;
DROP TABLE IF EXISTS mysteries;
