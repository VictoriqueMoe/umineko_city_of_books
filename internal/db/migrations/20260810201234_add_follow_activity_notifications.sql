-- +goose Up
ALTER TABLE users ADD COLUMN follow_activity_notifications BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE users DROP COLUMN follow_activity_notifications;
