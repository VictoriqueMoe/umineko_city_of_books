-- name: LikeArt :exec
INSERT INTO art_likes (user_id, art_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikeArt :exec
DELETE FROM art_likes WHERE user_id = $1 AND art_id = $2;

-- name: GetArtLikedBy :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS role
FROM art_likes lk
JOIN users u ON lk.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE lk.art_id = $1
  AND ($2::text = '' OR lk.user_id <> ALL(string_to_array($2::text, ',')::uuid[]))
ORDER BY lk.created_at DESC;
