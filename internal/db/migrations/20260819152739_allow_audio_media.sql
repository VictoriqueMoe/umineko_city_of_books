-- +goose Up
ALTER TABLE post_media DROP CONSTRAINT post_media_media_type_check;
ALTER TABLE post_media ADD CONSTRAINT post_media_media_type_check CHECK (media_type IN ('image', 'video', 'audio'));
ALTER TABLE post_comment_media DROP CONSTRAINT post_comment_media_media_type_check;
ALTER TABLE post_comment_media ADD CONSTRAINT post_comment_media_media_type_check CHECK (media_type IN ('image', 'video', 'audio'));
ALTER TABLE art_comment_media DROP CONSTRAINT art_comment_media_media_type_check;
ALTER TABLE art_comment_media ADD CONSTRAINT art_comment_media_media_type_check CHECK (media_type IN ('image', 'video', 'audio'));

-- +goose Down
DELETE FROM post_media WHERE media_type = 'audio';
DELETE FROM post_comment_media WHERE media_type = 'audio';
DELETE FROM art_comment_media WHERE media_type = 'audio';
ALTER TABLE post_media DROP CONSTRAINT post_media_media_type_check;
ALTER TABLE post_media ADD CONSTRAINT post_media_media_type_check CHECK (media_type IN ('image', 'video'));
ALTER TABLE post_comment_media DROP CONSTRAINT post_comment_media_media_type_check;
ALTER TABLE post_comment_media ADD CONSTRAINT post_comment_media_media_type_check CHECK (media_type IN ('image', 'video'));
ALTER TABLE art_comment_media DROP CONSTRAINT art_comment_media_media_type_check;
ALTER TABLE art_comment_media ADD CONSTRAINT art_comment_media_media_type_check CHECK (media_type IN ('image', 'video'));
