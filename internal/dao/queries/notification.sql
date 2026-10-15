-- name: CreateNotification :one
WITH n AS (
    INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message)
    VALUES ($1, $2, $3, $4, $5, $6)
    ON CONFLICT (user_id, type, reference_id) WHERE read = FALSE AND type IN ('chat_room_message', 'chat_message')
    DO UPDATE SET
        count = notifications.count + 1,
        actor_id = CASE WHEN EXCLUDED.created_at >= notifications.created_at THEN EXCLUDED.actor_id ELSE notifications.actor_id END,
        reference_type = CASE WHEN EXCLUDED.created_at >= notifications.created_at THEN EXCLUDED.reference_type ELSE notifications.reference_type END,
        message = CASE WHEN EXCLUDED.created_at >= notifications.created_at THEN EXCLUDED.message ELSE notifications.message END,
        created_at = GREATEST(notifications.created_at, EXCLUDED.created_at)
    RETURNING id, user_id, type, reference_id, reference_type, actor_id, message, read, created_at, count
)
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
       COALESCE(n.message, '')::text AS message, n.read, n.created_at, n.count,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM n
LEFT JOIN users u ON n.actor_id = u.id
LEFT JOIN user_roles ur ON n.actor_id = ur.user_id;

-- name: CreateNotifications :many
WITH n AS (
    INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message)
    SELECT b.user_id, b.type, b.reference_id, b.reference_type,
           (SELECT a.id FROM users a WHERE a.id = b.actor_id), b.message
    FROM (
        SELECT unnest(sqlc.arg(user_ids)::text[])::uuid AS user_id,
               unnest(sqlc.arg(types)::text[]) AS type,
               unnest(sqlc.arg(reference_ids)::text[]) AS reference_id,
               unnest(sqlc.arg(reference_types)::text[]) AS reference_type,
               unnest(sqlc.arg(actor_ids)::text[])::uuid AS actor_id,
               unnest(sqlc.arg(messages)::text[]) AS message
    ) b
    WHERE EXISTS (SELECT 1 FROM users r WHERE r.id = b.user_id)
    ORDER BY b.user_id, b.type, b.reference_id
    ON CONFLICT (user_id, type, reference_id) WHERE read = FALSE AND type IN ('chat_room_message', 'chat_message')
    DO UPDATE SET
        count = notifications.count + 1,
        actor_id = CASE WHEN EXCLUDED.created_at >= notifications.created_at THEN EXCLUDED.actor_id ELSE notifications.actor_id END,
        reference_type = CASE WHEN EXCLUDED.created_at >= notifications.created_at THEN EXCLUDED.reference_type ELSE notifications.reference_type END,
        message = CASE WHEN EXCLUDED.created_at >= notifications.created_at THEN EXCLUDED.message ELSE notifications.message END,
        created_at = GREATEST(notifications.created_at, EXCLUDED.created_at)
    RETURNING id, user_id, type, reference_id, reference_type, actor_id, message, read, created_at, count
)
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
       COALESCE(n.message, '')::text AS message, n.read, n.created_at, n.count,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM n
LEFT JOIN users u ON n.actor_id = u.id
LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
ORDER BY n.id;

-- name: CountUserNotifications :one
SELECT COUNT(*) FROM notifications WHERE user_id = $1;

-- name: ListUserNotifications :many
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
       COALESCE(n.message, '')::text AS message, n.read, n.created_at, n.count,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM notifications n
LEFT JOIN users u ON n.actor_id = u.id
LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
WHERE n.user_id = $1
ORDER BY n.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetNotificationByID :one
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
       COALESCE(n.message, '')::text AS message, n.read, n.created_at, n.count,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM notifications n
LEFT JOIN users u ON n.actor_id = u.id
LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
WHERE n.id = $1 AND n.user_id = $2;

-- name: MarkNotificationRead :exec
UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE;

-- name: MarkNotificationsReadByReference :exec
UPDATE notifications SET read = TRUE
WHERE user_id = $1 AND reference_id = $2 AND read = FALSE AND type = ANY(string_to_array($3::text, ','));

-- name: DeleteNotificationsOlderThanBatch :execrows
DELETE FROM notifications
WHERE id IN (
    SELECT o.id FROM notifications o
    WHERE o.created_at < $1
    ORDER BY o.id
    LIMIT $2
);

-- name: CountUnreadNotifications :one
SELECT COALESCE(SUM(count), 0)::bigint FROM notifications WHERE user_id = $1 AND read = FALSE;

-- name: CountRecentDuplicateNotifications :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND actor_id = $4
  AND created_at > NOW() - INTERVAL '1 hour';

-- name: CountRecentNotificationsFromActor :one
SELECT COUNT(*) FROM notifications
WHERE type = $1 AND actor_id = $2
  AND created_at > NOW() - make_interval(secs => $3::double precision);
