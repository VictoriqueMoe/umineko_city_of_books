-- +goose Up
ALTER TABLE mystery_clues ADD COLUMN player_id TEXT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX idx_mystery_clues_player_id ON mystery_clues(player_id);

-- +goose Down
DROP INDEX IF EXISTS idx_mystery_clues_player_id;
ALTER TABLE mystery_clues DROP COLUMN player_id;
