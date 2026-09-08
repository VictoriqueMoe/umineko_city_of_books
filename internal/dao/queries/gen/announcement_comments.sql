-- name: CreateAnnouncementComment :one
WITH ins AS (
    INSERT INTO announcement_comments (announcement_id, parent_id, user_id, body)
    VALUES ($1, $2, $3, $4)
    RETURNING id, announcement_id, parent_id, user_id, body, created_at, updated_at
)
SELECT c.id, c.announcement_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned
FROM ins c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id;

-- name: UpdateAnnouncementComment :execrows
UPDATE announcement_comments
SET body = $1, updated_at = NOW()
WHERE id = $2 AND ($3::boolean OR user_id = $4);

-- name: DeleteAnnouncementComment :execrows
DELETE FROM announcement_comments
WHERE id = $1 AND ($2::boolean OR user_id = $3);

-- name: CountAnnouncementComments :one
SELECT COUNT(*) FROM announcement_comments
WHERE announcement_id = $1
  AND ($2::text = '' OR user_id <> ALL(string_to_array($2::text, ',')::uuid[]));

-- name: ListAnnouncementComments :many
SELECT c.id, c.announcement_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned,
       (SELECT COUNT(*) FROM announcement_comment_likes lc WHERE lc.comment_id = c.id) AS like_count,
       EXISTS(SELECT 1 FROM announcement_comment_likes lu WHERE lu.comment_id = c.id AND lu.user_id = $1) AS user_liked
FROM announcement_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.announcement_id = $2
  AND ($3::text = '' OR c.user_id <> ALL(string_to_array($3::text, ',')::uuid[]))
ORDER BY c.created_at ASC
LIMIT $4 OFFSET $5;

-- name: GetAnnouncementCommentByID :one
SELECT c.id, c.announcement_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned
FROM announcement_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.id = $1;

-- name: GetAnnouncementCommentEntityID :one
SELECT announcement_id FROM announcement_comments WHERE id = $1;

-- name: GetAnnouncementCommentAuthorID :one
SELECT user_id FROM announcement_comments WHERE id = $1;

-- name: LikeAnnouncementComment :exec
INSERT INTO announcement_comment_likes (user_id, comment_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikeAnnouncementComment :exec
DELETE FROM announcement_comment_likes WHERE user_id = $1 AND comment_id = $2;

-- name: AddAnnouncementCommentMedia :one
INSERT INTO announcement_comment_media (comment_id, media_url, media_type, thumbnail_url, filename, sort_order, is_spoiler)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(m.sort_order) + 1 FROM announcement_comment_media m WHERE m.comment_id = $1), $6::int), $7)
RETURNING id;

-- name: GetAnnouncementCommentMedia :many
SELECT id, comment_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM announcement_comment_media
WHERE comment_id = $1
ORDER BY sort_order;

-- name: GetAnnouncementCommentMediaBatch :many
SELECT id, comment_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM announcement_comment_media
WHERE comment_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order;

-- name: UpdateAnnouncementCommentMediaURL :exec
UPDATE announcement_comment_media SET media_url = $1 WHERE id = $2;

-- name: UpdateAnnouncementCommentMediaThumbnail :exec
UPDATE announcement_comment_media SET thumbnail_url = $1 WHERE id = $2;

-- name: CollectAnnouncementCommentMediaPaths :many
SELECT m.media_url, m.thumbnail_url
FROM announcement_comment_media m
JOIN announcement_comments cm ON cm.id = m.comment_id
WHERE cm.announcement_id = $1;

-- name: CollectSingleAnnouncementCommentMediaPaths :many
WITH RECURSIVE tree AS (
    SELECT root.id FROM announcement_comments root WHERE root.id = $1
    UNION ALL
    SELECT child.id FROM announcement_comments child JOIN tree t ON child.parent_id = t.id
)
SELECT m.media_url, m.thumbnail_url
FROM announcement_comment_media m
JOIN tree ON tree.id = m.comment_id;
