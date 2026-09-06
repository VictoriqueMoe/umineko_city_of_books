-- +goose Up
ALTER TABLE post_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE mystery_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE journal_entry_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE post_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE art_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE announcement_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE mystery_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE ship_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE oc_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE fanfic_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE journal_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE secret_comment_media ADD COLUMN is_spoiler BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE secret_comment_media DROP COLUMN is_spoiler;
ALTER TABLE journal_comment_media DROP COLUMN is_spoiler;
ALTER TABLE fanfic_comment_media DROP COLUMN is_spoiler;
ALTER TABLE oc_comment_media DROP COLUMN is_spoiler;
ALTER TABLE ship_comment_media DROP COLUMN is_spoiler;
ALTER TABLE mystery_comment_media DROP COLUMN is_spoiler;
ALTER TABLE announcement_comment_media DROP COLUMN is_spoiler;
ALTER TABLE art_comment_media DROP COLUMN is_spoiler;
ALTER TABLE post_comment_media DROP COLUMN is_spoiler;
ALTER TABLE journal_entry_media DROP COLUMN is_spoiler;
ALTER TABLE mystery_media DROP COLUMN is_spoiler;
ALTER TABLE post_media DROP COLUMN is_spoiler;
