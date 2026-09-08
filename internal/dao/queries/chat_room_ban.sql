-- name: BanFromChatRoom :exec
INSERT INTO chat_room_bans (room_id, user_id, banned_by, reason)
VALUES ($1, $2, $3, $4)
ON CONFLICT (room_id, user_id) DO UPDATE SET
    banned_by = EXCLUDED.banned_by,
    reason    = EXCLUDED.reason,
    created_at = NOW();

-- name: UnbanFromChatRoom :exec
DELETE FROM chat_room_bans WHERE room_id = $1 AND user_id = $2;

-- name: IsBannedFromChatRoom :one
SELECT 1::int AS banned FROM chat_room_bans WHERE room_id = $1 AND user_id = $2 LIMIT 1;

-- name: ListChatRoomBans :many
SELECT b.room_id, b.user_id,
       u.username, u.display_name, u.avatar_url,
       COALESCE(ur.role, '')::text AS role,
       b.banned_by,
       COALESCE(bu.username, '')::text AS banned_by_username,
       COALESCE(bu.display_name, '')::text AS banned_by_display_name,
       COALESCE(bu.avatar_url, '')::text AS banned_by_avatar_url,
       b.reason, b.created_at
FROM chat_room_bans b
JOIN users u ON b.user_id = u.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN users bu ON b.banned_by = bu.id
WHERE b.room_id = $1
ORDER BY b.created_at DESC;

-- name: ListBannedRoomIDsForUser :many
SELECT room_id FROM chat_room_bans WHERE user_id = $1;
