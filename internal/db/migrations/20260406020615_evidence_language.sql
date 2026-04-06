-- +goose Up
ALTER TABLE theory_evidence ADD COLUMN lang TEXT NOT NULL DEFAULT 'en';
ALTER TABLE response_evidence ADD COLUMN lang TEXT NOT NULL DEFAULT 'en';

-- +goose Down
ALTER TABLE theory_evidence DROP COLUMN lang;
ALTER TABLE response_evidence DROP COLUMN lang;
