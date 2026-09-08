-- name: CreateOCComment :one
WITH ins AS (
    INSERT INTO oc_comments (oc_id, parent_id, user_id, body)
    VALUES ($1, $2, $3, $4)
    RETURNING id, oc_id, parent_id, user_id, body, created_at, updated_at
)
SELECT c.id, c.oc_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned
FROM ins c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id;

-- name: UpdateOCComment :execrows
UPDATE oc_comments
SET body = $1, updated_at = NOW()
WHERE id = $2 AND ($3::boolean OR user_id = $4);

-- name: DeleteOCComment :execrows
DELETE FROM oc_comments
WHERE id = $1 AND ($2::boolean OR user_id = $3);

-- name: CountOCComments :one
SELECT COUNT(*) FROM oc_comments
WHERE oc_id = $1
  AND ($2::text = '' OR user_id <> ALL(string_to_array($2::text, ',')::uuid[]));

-- name: ListOCComments :many
SELECT c.id, c.oc_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned,
       (SELECT COUNT(*) FROM oc_comment_likes lc WHERE lc.comment_id = c.id) AS like_count,
       EXISTS(SELECT 1 FROM oc_comment_likes lu WHERE lu.comment_id = c.id AND lu.user_id = $1) AS user_liked
FROM oc_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.oc_id = $2
  AND ($3::text = '' OR c.user_id <> ALL(string_to_array($3::text, ',')::uuid[]))
ORDER BY c.created_at ASC
LIMIT $4 OFFSET $5;

-- name: GetOCCommentByID :one
SELECT c.id, c.oc_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned
FROM oc_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.id = $1;

-- name: GetOCCommentEntityID :one
SELECT oc_id FROM oc_comments WHERE id = $1;

-- name: GetOCCommentAuthorID :one
SELECT user_id FROM oc_comments WHERE id = $1;

-- name: LikeOCComment :exec
INSERT INTO oc_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikeOCComment :exec
DELETE FROM oc_comment_likes WHERE user_id = $1 AND comment_id = $2;

-- name: AddOCCommentMedia :one
INSERT INTO oc_comment_media (comment_id, media_url, media_type, thumbnail_url, filename, sort_order, is_spoiler)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(m.sort_order) + 1 FROM oc_comment_media m WHERE m.comment_id = $1), $6::int), $7)
RETURNING id;

-- name: GetOCCommentMedia :many
SELECT id, comment_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM oc_comment_media
WHERE comment_id = $1
ORDER BY sort_order;

-- name: GetOCCommentMediaBatch :many
SELECT id, comment_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM oc_comment_media
WHERE comment_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order;

-- name: UpdateOCCommentMediaURL :exec
UPDATE oc_comment_media SET media_url = $1 WHERE id = $2;

-- name: UpdateOCCommentMediaThumbnail :exec
UPDATE oc_comment_media SET thumbnail_url = $1 WHERE id = $2;

-- name: CollectOCCommentMediaPaths :many
SELECT m.media_url, m.thumbnail_url
FROM oc_comment_media m
JOIN oc_comments cm ON cm.id = m.comment_id
WHERE cm.oc_id = $1;

-- name: CollectSingleOCCommentMediaPaths :many
WITH RECURSIVE tree AS (
    SELECT root.id FROM oc_comments root WHERE root.id = $1
    UNION ALL
    SELECT child.id FROM oc_comments child JOIN tree t ON child.parent_id = t.id
)
SELECT m.media_url, m.thumbnail_url
FROM oc_comment_media m
JOIN tree ON tree.id = m.comment_id;
