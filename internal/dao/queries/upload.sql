-- name: ListUploadTextColumns :many
SELECT table_name::text AS table_name, column_name::text AS column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND data_type IN ('text', 'character varying', 'citext')
  AND table_name NOT LIKE 'goose_%';
