-- +goose Up

ALTER TABLE users ADD COLUMN banner_url TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN gender TEXT DEFAULT '';

-- +goose Down

ALTER TABLE users DROP COLUMN gender;
ALTER TABLE users DROP COLUMN banner_url;
