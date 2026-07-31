-- +goose Up

ALTER TABLE chat_message_media ADD COLUMN width INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chat_message_media ADD COLUMN height INTEGER NOT NULL DEFAULT 0;

ALTER TABLE chat_message_media ADD CONSTRAINT chat_message_media_dimensions_check
    CHECK (width >= 0 AND height >= 0);

-- +goose Down

ALTER TABLE chat_message_media DROP CONSTRAINT chat_message_media_dimensions_check;
ALTER TABLE chat_message_media DROP COLUMN height;
ALTER TABLE chat_message_media DROP COLUMN width;
