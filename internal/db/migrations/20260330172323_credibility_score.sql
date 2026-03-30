-- +goose Up

ALTER TABLE theories ADD COLUMN credibility_score REAL DEFAULT 50.0;
ALTER TABLE response_evidence ADD COLUMN truth_weight REAL DEFAULT 1.0;
CREATE INDEX idx_theories_credibility ON theories(credibility_score DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_theories_credibility;
