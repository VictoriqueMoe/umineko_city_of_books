-- +goose Up

ALTER TABLE theory_evidence ADD COLUMN quote_index INTEGER DEFAULT NULL;
ALTER TABLE response_evidence ADD COLUMN quote_index INTEGER DEFAULT NULL;

-- +goose Down

ALTER TABLE theory_evidence DROP COLUMN quote_index;
ALTER TABLE response_evidence DROP COLUMN quote_index;
