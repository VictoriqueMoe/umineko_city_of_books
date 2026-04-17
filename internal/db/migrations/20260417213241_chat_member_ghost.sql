-- +goose Up
-- +goose StatementBegin
ALTER TABLE chat_room_members ADD COLUMN ghost INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE chat_room_members DROP COLUMN ghost;
-- +goose StatementEnd
