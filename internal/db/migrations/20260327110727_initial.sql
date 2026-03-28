-- +goose Up

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE theories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    episode INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE theory_evidence (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theory_id INTEGER NOT NULL REFERENCES theories(id) ON DELETE CASCADE,
    audio_id TEXT NOT NULL,
    note TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0
);

CREATE TABLE responses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theory_id INTEGER NOT NULL REFERENCES theories(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    side TEXT NOT NULL CHECK (side IN ('with_love', 'without_love')),
    body TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE response_evidence (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    response_id INTEGER NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    audio_id TEXT NOT NULL,
    note TEXT DEFAULT '',
    sort_order INTEGER DEFAULT 0
);

CREATE TABLE theory_votes (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    theory_id INTEGER NOT NULL REFERENCES theories(id) ON DELETE CASCADE,
    value INTEGER NOT NULL CHECK (value IN (1, -1)),
    PRIMARY KEY (user_id, theory_id)
);

CREATE TABLE response_votes (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    response_id INTEGER NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    value INTEGER NOT NULL CHECK (value IN (1, -1)),
    PRIMARY KEY (user_id, response_id)
);

CREATE INDEX idx_theories_user_id ON theories(user_id);
CREATE INDEX idx_theories_created_at ON theories(created_at DESC);
CREATE INDEX idx_theories_episode ON theories(episode);
CREATE INDEX idx_responses_theory_id ON responses(theory_id);
CREATE INDEX idx_responses_user_id ON responses(user_id);
CREATE INDEX idx_theory_evidence_theory_id ON theory_evidence(theory_id);
CREATE INDEX idx_response_evidence_response_id ON response_evidence(response_id);

-- +goose Down

DROP TABLE IF EXISTS response_votes;
DROP TABLE IF EXISTS theory_votes;
DROP TABLE IF EXISTS response_evidence;
DROP TABLE IF EXISTS responses;
DROP TABLE IF EXISTS theory_evidence;
DROP TABLE IF EXISTS theories;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
