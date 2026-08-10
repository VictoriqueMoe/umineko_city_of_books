-- +goose Up
CREATE INDEX idx_journals_created_at ON journals(created_at DESC);

-- +goose Down
DROP INDEX idx_journals_created_at;
