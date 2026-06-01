-- +goose Up

ALTER TABLE chat_watch_party_sessions
    ADD COLUMN type TEXT NOT NULL DEFAULT 'hyperbeam'
    CHECK (type IN ('hyperbeam','screenshare'));

-- +goose Down

ALTER TABLE chat_watch_party_sessions DROP COLUMN type;
