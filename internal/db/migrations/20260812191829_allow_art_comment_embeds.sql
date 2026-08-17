-- +goose Up
ALTER TABLE embeds DROP CONSTRAINT embeds_owner_type_check;
ALTER TABLE embeds ADD CONSTRAINT embeds_owner_type_check CHECK (owner_type IN ('post', 'comment', 'art_comment'));

-- +goose Down
DELETE FROM embeds WHERE owner_type = 'art_comment';
ALTER TABLE embeds DROP CONSTRAINT embeds_owner_type_check;
ALTER TABLE embeds ADD CONSTRAINT embeds_owner_type_check CHECK (owner_type IN ('post', 'comment'));
