-- name: ListBannedGiphy :many
SELECT kind, value, created_at, created_by, reason FROM banned_giphy ORDER BY created_at DESC;

-- name: AddBannedGiphy :exec
INSERT INTO banned_giphy (kind, value, created_by, reason) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING;

-- name: RemoveBannedGiphy :exec
DELETE FROM banned_giphy WHERE kind = $1 AND value = $2;
