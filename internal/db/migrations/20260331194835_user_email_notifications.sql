-- +goose Up
ALTER TABLE users ADD COLUMN email_notifications BOOLEAN DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN email_notifications;
