-- name: CreateAnnouncement :one
WITH a AS (
    INSERT INTO announcements (author_id, title, body)
    VALUES ($1, $2, $3)
    RETURNING id, title, body, author_id, pinned, created_at, updated_at
)
SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
       COALESCE(u.username, '')::text AS author_username,
       COALESCE(u.display_name, '')::text AS author_display_name,
       COALESCE(u.avatar_url, '')::text AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM a
LEFT JOIN users u ON u.id = a.author_id
LEFT JOIN user_roles r ON r.user_id = u.id;

-- name: UpdateAnnouncement :exec
UPDATE announcements SET title = $1, body = $2, updated_at = NOW() WHERE id = $3;

-- name: DeleteAnnouncement :exec
DELETE FROM announcements WHERE id = $1;

-- name: GetAnnouncementByID :one
SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
       COALESCE(u.username, '')::text AS author_username,
       COALESCE(u.display_name, '')::text AS author_display_name,
       COALESCE(u.avatar_url, '')::text AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM announcements a
LEFT JOIN users u ON a.author_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE a.id = $1;

-- name: CountAnnouncements :one
SELECT COUNT(*) FROM announcements;

-- name: ListAnnouncements :many
SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
       COALESCE(u.username, '')::text AS author_username,
       COALESCE(u.display_name, '')::text AS author_display_name,
       COALESCE(u.avatar_url, '')::text AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM announcements a
LEFT JOIN users u ON a.author_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
ORDER BY a.pinned DESC, a.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetLatestAnnouncement :one
SELECT a.id, a.title, a.body, a.author_id, a.pinned, a.created_at, a.updated_at,
       COALESCE(u.username, '')::text AS author_username,
       COALESCE(u.display_name, '')::text AS author_display_name,
       COALESCE(u.avatar_url, '')::text AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM announcements a
LEFT JOIN users u ON a.author_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
ORDER BY a.pinned DESC, a.created_at DESC
LIMIT 1;

-- name: SetAnnouncementPinned :exec
UPDATE announcements SET pinned = $1 WHERE id = $2;
