-- +goose Up

CREATE TABLE game_rooms (
    id TEXT PRIMARY KEY,
    game_type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'finished', 'declined', 'abandoned')),
    state_json TEXT NOT NULL DEFAULT '{}',
    turn_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    winner_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    result TEXT,
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME
);

CREATE INDEX idx_game_rooms_status ON game_rooms(status);
CREATE INDEX idx_game_rooms_game_type_status ON game_rooms(game_type, status);

CREATE TABLE game_room_players (
    room_id TEXT NOT NULL REFERENCES game_rooms(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slot INTEGER NOT NULL,
    joined INTEGER NOT NULL DEFAULT 0,
    joined_at DATETIME,
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, user_id),
    UNIQUE (room_id, slot)
);

CREATE INDEX idx_game_room_players_user ON game_room_players(user_id);

CREATE TABLE game_room_moves (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id TEXT NOT NULL REFERENCES game_rooms(id) ON DELETE CASCADE,
    ply INTEGER NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_json TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (room_id, ply)
);

CREATE INDEX idx_game_room_moves_room ON game_room_moves(room_id, ply);

-- +goose Down

DROP TABLE IF EXISTS game_room_moves;
DROP TABLE IF EXISTS game_room_players;
DROP TABLE IF EXISTS game_rooms;
