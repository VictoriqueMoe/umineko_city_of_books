-- +goose Up

ALTER TABLE journal_entries ADD COLUMN is_draft BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_journal_entries_draft ON journal_entries(journal_id, is_draft);

-- +goose Down

DROP INDEX IF EXISTS idx_journal_entries_draft;
ALTER TABLE journal_entries DROP COLUMN is_draft;
