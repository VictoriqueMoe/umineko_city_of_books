-- +goose Up
CREATE TYPE theory_status AS ENUM ('open', 'contested', 'refuted');

ALTER TABLE theories ADD COLUMN status theory_status NOT NULL DEFAULT 'open';
ALTER TABLE theories ADD COLUMN refuted_by_response_id UUID;
ALTER TABLE theories ADD COLUMN refuted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE theories ADD COLUMN refuted_at TIMESTAMPTZ;

ALTER TABLE theories ADD CONSTRAINT theories_refuted_by_same_theory_fkey FOREIGN KEY (id, refuted_by_response_id) REFERENCES responses (theory_id, id) ON DELETE SET NULL (refuted_by_response_id);

ALTER TABLE theories ADD CONSTRAINT theories_unrefuted_state_check CHECK (status = 'refuted' OR (refuted_by_response_id IS NULL AND refuted_by_user_id IS NULL AND refuted_at IS NULL));

UPDATE theories t SET status = 'contested' WHERE EXISTS (SELECT 1 FROM responses r WHERE r.theory_id = t.id AND r.parent_id IS NULL);

CREATE INDEX idx_theories_status ON theories(status);

-- +goose Down
DROP INDEX idx_theories_status;
ALTER TABLE theories DROP CONSTRAINT theories_unrefuted_state_check;
ALTER TABLE theories DROP CONSTRAINT theories_refuted_by_same_theory_fkey;
ALTER TABLE theories DROP COLUMN refuted_at;
ALTER TABLE theories DROP COLUMN refuted_by_user_id;
ALTER TABLE theories DROP COLUMN refuted_by_response_id;
ALTER TABLE theories DROP COLUMN status;
DROP TYPE theory_status;
