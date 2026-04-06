-- +goose Up
CREATE TABLE post_polls (
    id TEXT PRIMARY KEY,
    post_id TEXT NOT NULL UNIQUE REFERENCES posts(id) ON DELETE CASCADE,
    duration_seconds INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_post_polls_post_id ON post_polls(post_id);

CREATE TABLE post_poll_options (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    poll_id TEXT NOT NULL REFERENCES post_polls(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_post_poll_options_poll_id ON post_poll_options(poll_id);

CREATE TABLE post_poll_votes (
    poll_id TEXT NOT NULL REFERENCES post_polls(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    option_id INTEGER NOT NULL REFERENCES post_poll_options(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (poll_id, user_id)
);

CREATE INDEX idx_post_poll_votes_option_id ON post_poll_votes(option_id);

-- +goose Down
DROP TABLE IF EXISTS post_poll_votes;
DROP TABLE IF EXISTS post_poll_options;
DROP TABLE IF EXISTS post_polls;
