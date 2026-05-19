-- +goose Up
CREATE TYPE chat_watch_party_message_kind AS ENUM ('user', 'system');

ALTER TABLE chat_watch_party_messages
    ADD COLUMN kind chat_watch_party_message_kind NOT NULL DEFAULT 'user',
    ALTER COLUMN sender_id DROP NOT NULL;

ALTER TABLE chat_watch_party_messages
    ADD CONSTRAINT chat_watch_party_message_sender_chk
        CHECK (
            (kind = 'user'   AND sender_id IS NOT NULL)
         OR (kind = 'system' AND sender_id IS NULL)
        );

-- +goose Down
ALTER TABLE chat_watch_party_messages
    DROP CONSTRAINT IF EXISTS chat_watch_party_message_sender_chk;

DELETE FROM chat_watch_party_messages WHERE kind = 'system';

ALTER TABLE chat_watch_party_messages
    DROP COLUMN kind,
    ALTER COLUMN sender_id SET NOT NULL;

DROP TYPE IF EXISTS chat_watch_party_message_kind;
