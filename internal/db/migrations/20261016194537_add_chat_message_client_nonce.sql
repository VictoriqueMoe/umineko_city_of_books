-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS client_nonce TEXT;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_chat_messages_sender_client_nonce
    ON chat_messages (sender_id, client_nonce)
    WHERE client_nonce IS NOT NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_chat_messages_sender_client_nonce;

ALTER TABLE chat_messages DROP COLUMN IF EXISTS client_nonce;
