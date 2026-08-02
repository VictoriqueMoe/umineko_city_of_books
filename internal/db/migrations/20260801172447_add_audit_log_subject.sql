-- +goose Up
ALTER TABLE audit_log ADD COLUMN subject_id UUID;

UPDATE audit_log SET subject_id = (details::jsonb ->> 'target_user_id')::uuid
WHERE subject_id IS NULL
  AND details IS JSON OBJECT
  AND (details::jsonb ->> 'target_user_id') IS NOT NULL;

UPDATE audit_log SET subject_id = (substring(details from 'user=([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})'))::uuid
WHERE subject_id IS NULL AND details ~ 'user=[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}';

UPDATE audit_log SET subject_id = (substring(details from 'target=([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})'))::uuid
WHERE subject_id IS NULL AND details ~ 'target=[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}';

UPDATE audit_log a SET subject_id = NULL
WHERE a.subject_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users u WHERE u.id = a.subject_id);

ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_id_fkey FOREIGN KEY (subject_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_audit_log_subject ON audit_log (subject_id, created_at DESC) WHERE subject_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_audit_log_subject;
ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_id_fkey;
ALTER TABLE audit_log DROP COLUMN subject_id;
