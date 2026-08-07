-- +goose Up
ALTER TABLE chatbots ADD COLUMN verbosity TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE chatbots DROP COLUMN verbosity;
