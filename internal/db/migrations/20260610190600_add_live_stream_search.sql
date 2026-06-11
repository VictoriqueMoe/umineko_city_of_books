-- +goose Up
ALTER TABLE live_streams
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (setweight(to_tsvector('english', coalesce(title, '')), 'A')) STORED;

CREATE INDEX idx_live_streams_search_vector ON live_streams USING GIN (search_vector);

-- +goose Down
DROP INDEX IF EXISTS idx_live_streams_search_vector;
ALTER TABLE live_streams DROP COLUMN IF EXISTS search_vector;
