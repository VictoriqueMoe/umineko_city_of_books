-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN email_public BOOLEAN DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN email;
ALTER TABLE users DROP COLUMN email_public;
