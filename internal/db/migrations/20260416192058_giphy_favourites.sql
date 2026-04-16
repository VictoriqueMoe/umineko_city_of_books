-- +goose Up
CREATE TABLE giphy_favourites (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    giphy_id TEXT NOT NULL,
    url TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    preview_url TEXT NOT NULL DEFAULT '',
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, giphy_id)
);
CREATE INDEX idx_giphy_favourites_user_created ON giphy_favourites(user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_giphy_favourites_user_created;
DROP TABLE IF EXISTS giphy_favourites;
