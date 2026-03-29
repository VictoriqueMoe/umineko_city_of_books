-- +goose Up

ALTER TABLE users ADD COLUMN pronoun_subject TEXT DEFAULT 'they';
ALTER TABLE users ADD COLUMN pronoun_possessive TEXT DEFAULT 'their';

-- +goose Down

ALTER TABLE users DROP COLUMN pronoun_possessive;
ALTER TABLE users DROP COLUMN pronoun_subject;
