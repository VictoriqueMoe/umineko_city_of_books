-- name: GetStoredFile :one
SELECT key, backend, location, size, created_at FROM stored_files WHERE key = $1;

-- name: UpsertStoredFile :exec
INSERT INTO stored_files (key, backend, location, size) VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE SET backend = EXCLUDED.backend, location = EXCLUDED.location, size = EXCLUDED.size, created_at = now();

-- name: BackfillStoredFiles :execrows
INSERT INTO stored_files (key, backend, location, size)
SELECT unnest(sqlc.arg(keys)::text[]), sqlc.arg(backend)::storage_backend, unnest(sqlc.arg(locations)::text[]), unnest(sqlc.arg(sizes)::bigint[])
ON CONFLICT (key) DO NOTHING;

-- name: DeleteStoredFiles :exec
DELETE FROM stored_files WHERE key = ANY(sqlc.arg(keys)::text[]);

-- name: ListStoredFilesByKeys :many
SELECT key, backend, location, size, created_at FROM stored_files WHERE key = ANY(sqlc.arg(keys)::text[]);

-- name: ListStoredFilesByPrefix :many
SELECT key, backend, location, size, created_at FROM stored_files
WHERE starts_with(key, sqlc.arg(prefix)::text)
ORDER BY key;

-- name: CountStoredFilesByBackend :one
SELECT count(*) FROM stored_files WHERE backend = $1;
