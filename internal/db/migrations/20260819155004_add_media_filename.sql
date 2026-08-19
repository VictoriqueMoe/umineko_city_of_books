-- +goose Up
ALTER TABLE announcement_comment_media ADD COLUMN filename TEXT;
ALTER TABLE art_comment_media ADD COLUMN filename TEXT;
ALTER TABLE chat_message_media ADD COLUMN filename TEXT;
ALTER TABLE fanfic_comment_media ADD COLUMN filename TEXT;
ALTER TABLE journal_comment_media ADD COLUMN filename TEXT;
ALTER TABLE journal_entry_media ADD COLUMN filename TEXT;
ALTER TABLE mystery_comment_media ADD COLUMN filename TEXT;
ALTER TABLE mystery_media ADD COLUMN filename TEXT;
ALTER TABLE oc_comment_media ADD COLUMN filename TEXT;
ALTER TABLE post_comment_media ADD COLUMN filename TEXT;
ALTER TABLE post_media ADD COLUMN filename TEXT;
ALTER TABLE secret_comment_media ADD COLUMN filename TEXT;
ALTER TABLE ship_comment_media ADD COLUMN filename TEXT;

-- +goose Down
ALTER TABLE announcement_comment_media DROP COLUMN filename;
ALTER TABLE art_comment_media DROP COLUMN filename;
ALTER TABLE chat_message_media DROP COLUMN filename;
ALTER TABLE fanfic_comment_media DROP COLUMN filename;
ALTER TABLE journal_comment_media DROP COLUMN filename;
ALTER TABLE journal_entry_media DROP COLUMN filename;
ALTER TABLE mystery_comment_media DROP COLUMN filename;
ALTER TABLE mystery_media DROP COLUMN filename;
ALTER TABLE oc_comment_media DROP COLUMN filename;
ALTER TABLE post_comment_media DROP COLUMN filename;
ALTER TABLE post_media DROP COLUMN filename;
ALTER TABLE secret_comment_media DROP COLUMN filename;
ALTER TABLE ship_comment_media DROP COLUMN filename;
