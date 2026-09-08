-- name: CreateMysteryComment :one
WITH ins AS (
    INSERT INTO mystery_comments (mystery_id, parent_id, user_id, body)
    VALUES ($1, $2, $3, $4)
    RETURNING id, mystery_id, parent_id, user_id, body, created_at, updated_at
)
SELECT c.id, c.mystery_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned
FROM ins c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id;

-- name: UpdateMysteryComment :execrows
UPDATE mystery_comments
SET body = $1, updated_at = NOW()
WHERE id = $2 AND ($3::boolean OR user_id = $4);

-- name: DeleteMysteryComment :execrows
DELETE FROM mystery_comments
WHERE id = $1 AND ($2::boolean OR user_id = $3);

-- name: CountMysteryComments :one
SELECT COUNT(*) FROM mystery_comments
WHERE mystery_id = $1
  AND ($2::text = '' OR user_id <> ALL(string_to_array($2::text, ',')::uuid[]));

-- name: ListMysteryComments :many
SELECT c.id, c.mystery_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned,
       (SELECT COUNT(*) FROM mystery_comment_likes lc WHERE lc.comment_id = c.id) AS like_count,
       EXISTS(SELECT 1 FROM mystery_comment_likes lu WHERE lu.comment_id = c.id AND lu.user_id = $1) AS user_liked
FROM mystery_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.mystery_id = $2
  AND ($3::text = '' OR c.user_id <> ALL(string_to_array($3::text, ',')::uuid[]))
ORDER BY c.created_at ASC
LIMIT $4 OFFSET $5;

-- name: GetMysteryCommentByID :one
SELECT c.id, c.mystery_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned
FROM mystery_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.id = $1;

-- name: GetMysteryCommentEntityID :one
SELECT mystery_id FROM mystery_comments WHERE id = $1;

-- name: GetMysteryCommentAuthorID :one
SELECT user_id FROM mystery_comments WHERE id = $1;

-- name: LikeMysteryComment :exec
INSERT INTO mystery_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikeMysteryComment :exec
DELETE FROM mystery_comment_likes WHERE user_id = $1 AND comment_id = $2;

-- name: AddMysteryCommentMedia :one
INSERT INTO mystery_comment_media (comment_id, media_url, media_type, thumbnail_url, filename, sort_order, is_spoiler)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(m.sort_order) + 1 FROM mystery_comment_media m WHERE m.comment_id = $1), $6::int), $7)
RETURNING id;

-- name: GetMysteryCommentMedia :many
SELECT id, comment_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM mystery_comment_media
WHERE comment_id = $1
ORDER BY sort_order;

-- name: GetMysteryCommentMediaBatch :many
SELECT id, comment_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM mystery_comment_media
WHERE comment_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order;

-- name: UpdateMysteryCommentMediaURL :exec
UPDATE mystery_comment_media SET media_url = $1 WHERE id = $2;

-- name: UpdateMysteryCommentMediaThumbnail :exec
UPDATE mystery_comment_media SET thumbnail_url = $1 WHERE id = $2;

-- name: CollectMysteryCommentMediaPaths :many
SELECT m.media_url, m.thumbnail_url
FROM mystery_comment_media m
JOIN mystery_comments cm ON cm.id = m.comment_id
WHERE cm.mystery_id = $1;

-- name: CollectSingleMysteryCommentMediaPaths :many
WITH RECURSIVE tree AS (
    SELECT root.id FROM mystery_comments root WHERE root.id = $1
    UNION ALL
    SELECT child.id FROM mystery_comments child JOIN tree t ON child.parent_id = t.id
)
SELECT m.media_url, m.thumbnail_url
FROM mystery_comment_media m
JOIN tree ON tree.id = m.comment_id;
