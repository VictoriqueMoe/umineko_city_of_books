-- name: CreateNotification :one
WITH n AS (
    INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, user_id, type, reference_id, reference_type, actor_id, message, read, created_at
)
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
       COALESCE(n.message, '')::text AS message, n.read, n.created_at,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM n
LEFT JOIN users u ON n.actor_id = u.id
LEFT JOIN user_roles ur ON n.actor_id = ur.user_id;

-- name: CountUserNotifications :one
SELECT (
    (SELECT COUNT(DISTINCT (g.type, g.reference_id)) FROM notifications g
       WHERE g.user_id = $1 AND g.type = ANY(string_to_array($2::text, ',')) AND g.read = FALSE)
    +
    (SELECT COUNT(*) FROM notifications p
       WHERE p.user_id = $1 AND NOT (p.type = ANY(string_to_array($2::text, ',')) AND p.read = FALSE))
)::bigint AS total;

-- name: ListUserNotifications :many
WITH chat_grouped AS (
    SELECT
      g.id, g.user_id, g.type, g.reference_id, g.reference_type, g.actor_id, g.message, g.read, g.created_at,
      ROW_NUMBER() OVER (PARTITION BY g.type, g.reference_id ORDER BY g.created_at DESC, g.id DESC) AS rn,
      COUNT(*) OVER (PARTITION BY g.type, g.reference_id) AS grp_count
    FROM notifications g
    WHERE g.user_id = $1 AND g.type = ANY(string_to_array($2::text, ',')) AND g.read = FALSE
),
combined AS (
    SELECT cg.id, cg.user_id, cg.type, cg.reference_id, cg.reference_type, cg.actor_id,
           COALESCE(cg.message, '')::text AS message, cg.read, cg.created_at, cg.grp_count::bigint AS count
    FROM chat_grouped cg
    WHERE cg.rn = 1
    UNION ALL
    SELECT p.id, p.user_id, p.type, p.reference_id, p.reference_type, p.actor_id,
           COALESCE(p.message, '')::text AS message, p.read, p.created_at, 1::bigint AS count
    FROM notifications p
    WHERE p.user_id = $1 AND NOT (p.type = ANY(string_to_array($2::text, ',')) AND p.read = FALSE)
)
SELECT c.id, c.user_id, c.type, c.reference_id, c.reference_type, c.actor_id,
       c.message, c.read, c.created_at, c.count,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM combined c
LEFT JOIN users u ON c.actor_id = u.id
LEFT JOIN user_roles ur ON c.actor_id = ur.user_id
ORDER BY c.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetNotificationByID :one
SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
       COALESCE(n.message, '')::text AS message, n.read, n.created_at,
       COALESCE(u.username, '')::text AS actor_username,
       COALESCE(u.display_name, '')::text AS actor_display_name,
       COALESCE(u.avatar_url, '')::text AS actor_avatar_url,
       COALESCE(ur.role, '')::text AS actor_role
FROM notifications n
LEFT JOIN users u ON n.actor_id = u.id
LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
WHERE n.id = $1 AND n.user_id = $2;

-- name: GetNotificationReadState :one
SELECT type, reference_id, read FROM notifications WHERE id = $1 AND user_id = $2;

-- name: MarkGroupedNotificationsRead :exec
UPDATE notifications SET read = TRUE
WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND read = FALSE;

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
SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE;

-- name: CountRecentDuplicateNotifications :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND actor_id = $4
  AND created_at > NOW() - INTERVAL '1 hour';

-- name: CountRecentNotificationsFromActor :one
SELECT COUNT(*) FROM notifications
WHERE type = $1 AND actor_id = $2
  AND created_at > NOW() - make_interval(secs => $3::double precision);
