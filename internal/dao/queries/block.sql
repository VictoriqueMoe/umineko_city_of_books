-- name: CreateBlock :exec
INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteBlock :exec
DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2;

-- name: CountBlock :one
SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2;

-- name: CountBlockEither :one
SELECT COUNT(*) FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $3 AND blocked_id = $4);

-- name: ListBlockedIDs :many
SELECT b1.blocked_id FROM blocks b1 WHERE b1.blocker_id = $1
UNION
SELECT b2.blocker_id FROM blocks b2 WHERE b2.blocked_id = $2;

-- name: ListBlockedUsers :many
SELECT u.id, u.username, u.display_name, u.avatar_url, b.created_at
FROM blocks b
JOIN users u ON b.blocked_id = u.id
WHERE b.blocker_id = $1
ORDER BY b.created_at DESC;
