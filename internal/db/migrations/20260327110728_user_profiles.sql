-- +goose Up

ALTER TABLE users ADD COLUMN bio TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN avatar_url TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN favourite_character TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN social_twitter TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN social_discord TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN social_waifulist TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN social_tumblr TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN social_github TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN website TEXT DEFAULT '';

-- +goose Down

ALTER TABLE users DROP COLUMN website;
ALTER TABLE users DROP COLUMN social_github;
ALTER TABLE users DROP COLUMN social_tumblr;
ALTER TABLE users DROP COLUMN social_waifulist;
ALTER TABLE users DROP COLUMN social_discord;
ALTER TABLE users DROP COLUMN social_twitter;
ALTER TABLE users DROP COLUMN favourite_character;
ALTER TABLE users DROP COLUMN avatar_url;
ALTER TABLE users DROP COLUMN bio;
