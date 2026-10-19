-- +goose Up
ALTER TABLE stored_files ADD COLUMN fallback BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_stored_files_fallback ON stored_files(key) WHERE fallback;

-- +goose Down
DROP INDEX IF EXISTS idx_stored_files_fallback;

ALTER TABLE stored_files DROP COLUMN IF EXISTS fallback;
