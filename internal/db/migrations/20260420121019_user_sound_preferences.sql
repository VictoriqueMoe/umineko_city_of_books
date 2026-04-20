-- +goose Up
ALTER TABLE users ADD COLUMN play_message_sound BOOLEAN NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN play_notification_sound BOOLEAN NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE users DROP COLUMN play_message_sound;
ALTER TABLE users DROP COLUMN play_notification_sound;
