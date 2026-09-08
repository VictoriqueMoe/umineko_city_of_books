-- name: AddPostMedia :one
INSERT INTO post_media (post_id, media_url, media_type, thumbnail_url, filename, sort_order, is_spoiler)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(m.sort_order) + 1 FROM post_media m WHERE m.post_id = $1), $6::int), $7)
RETURNING id;

-- name: GetPostMediaURL :one
SELECT media_url FROM post_media WHERE id = $1 AND post_id = $2;

-- name: DeletePostMediaRow :exec
DELETE FROM post_media WHERE id = $1 AND post_id = $2;

-- name: UpdatePostMediaURL :exec
UPDATE post_media SET media_url = $1 WHERE id = $2;

-- name: UpdatePostMediaThumbnail :exec
UPDATE post_media SET thumbnail_url = $1 WHERE id = $2;

-- name: GetPostMedia :many
SELECT id, post_id AS target_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM post_media
WHERE post_id = $1
ORDER BY sort_order, id;

-- name: GetPostMediaBatch :many
SELECT id, post_id AS target_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM post_media
WHERE post_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order, id;

-- name: CollectPostMediaPaths :many
SELECT media_url, thumbnail_url FROM post_media WHERE post_id = $1;
