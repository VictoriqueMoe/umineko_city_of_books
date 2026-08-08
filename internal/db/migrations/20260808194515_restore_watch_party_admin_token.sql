-- +goose Up
-- +goose StatementBegin
ALTER TABLE chat_watch_party_sessions ADD COLUMN hyperbeam_admin_token TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE chat_watch_party_sessions DROP COLUMN hyperbeam_admin_token;
-- +goose StatementEnd
