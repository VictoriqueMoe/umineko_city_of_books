-- +goose Up
ALTER TABLE users ADD COLUMN dob TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN dob_public BOOLEAN DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN dob_public;
ALTER TABLE users DROP COLUMN dob;
