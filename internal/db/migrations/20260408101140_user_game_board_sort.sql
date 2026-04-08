-- +goose Up
ALTER TABLE users ADD COLUMN game_board_sort TEXT DEFAULT 'relevance';

-- +goose Down
ALTER TABLE users DROP COLUMN game_board_sort;
