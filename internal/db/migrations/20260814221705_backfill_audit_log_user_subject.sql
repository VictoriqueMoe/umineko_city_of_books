-- +goose Up
UPDATE audit_log a SET subject_id = a.target_id::uuid
WHERE a.subject_id IS NULL
  AND a.target_type = 'user'
  AND a.target_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
  AND EXISTS (SELECT 1 FROM users u WHERE u.id = a.target_id::uuid);

-- +goose Down
UPDATE audit_log a SET subject_id = NULL
WHERE a.target_type = 'user'
  AND a.target_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
  AND a.subject_id = a.target_id::uuid;
