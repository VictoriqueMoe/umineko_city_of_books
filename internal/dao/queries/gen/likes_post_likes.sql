-- name: LikePost :exec
INSERT INTO post_likes (user_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikePost :exec
DELETE FROM post_likes WHERE user_id = $1 AND post_id = $2;

-- name: GetPostLikedBy :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS role
FROM post_likes lk
JOIN users u ON lk.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE lk.post_id = $1
  AND ($2::text = '' OR lk.user_id <> ALL(string_to_array($2::text, ',')::uuid[]))
ORDER BY lk.created_at DESC;
