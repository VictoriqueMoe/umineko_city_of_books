-- +goose Up
ALTER TABLE fanfics ALTER COLUMN published_at DROP NOT NULL;
UPDATE fanfics SET published_at = NULL WHERE status = 'draft';

DROP INDEX IF EXISTS idx_fanfics_published_at;
CREATE INDEX idx_fanfics_published_at ON fanfics (published_at DESC NULLS LAST);

ALTER TABLE journal_entries ADD COLUMN published_at TIMESTAMPTZ;
UPDATE journal_entries SET published_at = created_at WHERE NOT is_draft;
ALTER TABLE journal_entries ALTER COLUMN published_at SET DEFAULT NOW();

-- +goose Down
ALTER TABLE journal_entries DROP COLUMN IF EXISTS published_at;

DROP INDEX IF EXISTS idx_fanfics_published_at;
UPDATE fanfics SET published_at = created_at WHERE published_at IS NULL;
ALTER TABLE fanfics ALTER COLUMN published_at SET NOT NULL;
CREATE INDEX idx_fanfics_published_at ON fanfics (published_at DESC);
