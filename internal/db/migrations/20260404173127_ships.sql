-- +goose Up
CREATE TABLE ships (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    thumbnail_url TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ships_created_at ON ships(created_at DESC);
CREATE INDEX idx_ships_user_id ON ships(user_id);

CREATE TABLE ship_characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ship_id TEXT NOT NULL REFERENCES ships(id) ON DELETE CASCADE,
    series TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    character_name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_ship_characters_ship_id ON ship_characters(ship_id);
CREATE INDEX idx_ship_characters_lookup ON ship_characters(series, character_id);

CREATE TABLE ship_votes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ship_id TEXT NOT NULL REFERENCES ships(id) ON DELETE CASCADE,
    value INTEGER NOT NULL CHECK (value IN (1, -1)),
    PRIMARY KEY (user_id, ship_id)
);

CREATE INDEX idx_ship_votes_ship_id ON ship_votes(ship_id);

CREATE TABLE ship_comments (
    id TEXT PRIMARY KEY,
    ship_id TEXT NOT NULL REFERENCES ships(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES ship_comments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);

CREATE INDEX idx_ship_comments_ship_id ON ship_comments(ship_id);
CREATE INDEX idx_ship_comments_parent_id ON ship_comments(parent_id);

CREATE TABLE ship_comment_likes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL REFERENCES ship_comments(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_ship_comment_likes_comment_id ON ship_comment_likes(comment_id);

CREATE TABLE ship_comment_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id TEXT NOT NULL REFERENCES ship_comments(id) ON DELETE CASCADE,
    media_url TEXT NOT NULL,
    media_type TEXT NOT NULL,
    thumbnail_url TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_ship_comment_media_comment_id ON ship_comment_media(comment_id);

-- +goose Down
DROP TABLE IF EXISTS ship_comment_media;
DROP TABLE IF EXISTS ship_comment_likes;
DROP TABLE IF EXISTS ship_comments;
DROP TABLE IF EXISTS ship_votes;
DROP TABLE IF EXISTS ship_characters;
DROP TABLE IF EXISTS ships;
