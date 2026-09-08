-- name: CreateFollow :exec
INSERT INTO follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteFollow :exec
DELETE FROM follows WHERE follower_id = $1 AND following_id = $2;

-- name: CountFollowRelation :one
SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND following_id = $2;

-- name: CountFollowers :one
SELECT COUNT(*) FROM follows WHERE following_id = $1;

-- name: CountFollowing :one
SELECT COUNT(*) FROM follows WHERE follower_id = $1;

-- name: ListFollowers :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS role
FROM follows f
JOIN users u ON f.follower_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE f.following_id = $1
ORDER BY f.created_at DESC, f.follower_id DESC
LIMIT $2 OFFSET $3;

-- name: ListFollowing :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS role
FROM follows f
JOIN users u ON f.following_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE f.follower_id = $1
ORDER BY f.created_at DESC, f.following_id DESC
LIMIT $2 OFFSET $3;

-- name: ListMutualFollowers :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS role
FROM follows f1
JOIN follows f2 ON f1.following_id = f2.follower_id AND f2.following_id = f1.follower_id
JOIN users u ON f1.following_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE f1.follower_id = $1
ORDER BY LOWER(u.display_name);

-- name: ListFollowerIDsToNotify :many
SELECT f.follower_id
FROM follows f
JOIN users u ON u.id = f.follower_id
WHERE f.following_id = $1 AND u.follow_activity_notifications = TRUE AND u.is_bot = FALSE;
