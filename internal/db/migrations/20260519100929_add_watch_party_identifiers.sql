-- +goose Up

ALTER TABLE chat_watch_party_sessions ADD COLUMN vm_base_url TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_watch_party_participants ADD COLUMN hyperbeam_identifier TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE chat_watch_party_participants DROP COLUMN hyperbeam_identifier;
ALTER TABLE chat_watch_party_sessions DROP COLUMN vm_base_url;
