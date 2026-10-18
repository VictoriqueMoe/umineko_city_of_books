-- name: GetStoredFile :one
SELECT key, backend, location, size, created_at, fallback FROM stored_files WHERE key = $1;

-- name: UpsertStoredFile :exec
INSERT INTO stored_files (key, backend, location, size, fallback) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (key) DO UPDATE SET backend = EXCLUDED.backend, location = EXCLUDED.location, size = EXCLUDED.size, fallback = EXCLUDED.fallback, created_at = now();

-- name: BackfillStoredFiles :execrows
INSERT INTO stored_files (key, backend, location, size)
SELECT unnest(sqlc.arg(keys)::text[]), sqlc.arg(backend)::storage_backend, unnest(sqlc.arg(locations)::text[]), unnest(sqlc.arg(sizes)::bigint[])
ON CONFLICT (key) DO NOTHING;

-- name: DeleteStoredFilesAt :many
DELETE FROM stored_files
WHERE backend = sqlc.arg(backend)::storage_backend
  AND (key, location) IN (SELECT unnest(sqlc.arg(keys)::text[]), unnest(sqlc.arg(locations)::text[]))
RETURNING key;

-- name: ListStoredFilesByKeys :many
SELECT key, backend, location, size, created_at, fallback FROM stored_files WHERE key = ANY(sqlc.arg(keys)::text[]);

-- name: ListStoredFilesByPrefix :many
SELECT key, backend, location, size, created_at, fallback FROM stored_files
WHERE starts_with(key, sqlc.arg(prefix)::text)
ORDER BY key;

-- name: CountStoredFilesByBackend :one
SELECT count(*) FROM stored_files WHERE backend = $1;

-- name: ListFallbackStoredFiles :many
SELECT key, backend, location, size, created_at, fallback FROM stored_files
WHERE fallback AND backend = 'local'
ORDER BY key;

-- name: MoveStoredFile :execrows
UPDATE stored_files SET backend = sqlc.arg(backend), location = sqlc.arg(location), size = sqlc.arg(size), fallback = false
WHERE key = sqlc.arg(key) AND backend = sqlc.arg(from_backend) AND location = sqlc.arg(from_location);
