-- +goose Up

ALTER TABLE responses ADD COLUMN parent_id INTEGER DEFAULT NULL REFERENCES responses(id) ON DELETE CASCADE;
CREATE INDEX idx_responses_parent_id ON responses(parent_id);

-- +goose Down

DROP INDEX IF EXISTS idx_responses_parent_id;
ALTER TABLE responses DROP COLUMN parent_id;
