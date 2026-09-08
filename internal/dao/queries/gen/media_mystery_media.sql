-- name: AddMysteryMedia :one
INSERT INTO mystery_media (mystery_id, media_url, media_type, thumbnail_url, filename, sort_order, is_spoiler)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(m.sort_order) + 1 FROM mystery_media m WHERE m.mystery_id = $1), $6::int), $7)
RETURNING id;

-- name: GetMysteryMediaURL :one
SELECT media_url FROM mystery_media WHERE id = $1 AND mystery_id = $2;

-- name: DeleteMysteryMediaRow :exec
DELETE FROM mystery_media WHERE id = $1 AND mystery_id = $2;

-- name: UpdateMysteryMediaURL :exec
UPDATE mystery_media SET media_url = $1 WHERE id = $2;

-- name: UpdateMysteryMediaThumbnail :exec
UPDATE mystery_media SET thumbnail_url = $1 WHERE id = $2;

-- name: GetMysteryMedia :many
SELECT id, mystery_id AS target_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM mystery_media
WHERE mystery_id = $1
ORDER BY sort_order, id;

-- name: GetMysteryMediaBatch :many
SELECT id, mystery_id AS target_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM mystery_media
WHERE mystery_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order, id;

-- name: CollectMysteryMediaPaths :many
SELECT media_url, thumbnail_url FROM mystery_media WHERE mystery_id = $1;
