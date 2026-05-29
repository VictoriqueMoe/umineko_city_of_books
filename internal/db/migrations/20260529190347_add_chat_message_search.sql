-- +goose Up

ALTER TABLE chat_messages ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED;
CREATE INDEX idx_chat_messages_search_vector ON chat_messages USING GIN(search_vector);
CREATE INDEX idx_chat_messages_body_trgm ON chat_messages USING GIN(body gin_trgm_ops);

-- +goose Down

DROP INDEX IF EXISTS idx_chat_messages_body_trgm;
DROP INDEX IF EXISTS idx_chat_messages_search_vector;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS search_vector;