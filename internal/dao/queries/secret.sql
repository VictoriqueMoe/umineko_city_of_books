-- name: GetFirstSecretSolver :one
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       us.unlocked_at
FROM user_secrets us
JOIN users u ON us.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE us.secret_id = $1
ORDER BY us.unlocked_at ASC
LIMIT 1;

-- name: GetSecretProgressLeaderboard :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COUNT(*) AS pieces
FROM user_secrets us
JOIN users u ON us.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE us.secret_id = ANY(string_to_array($1::text, ',')::text[])
GROUP BY u.id, u.username, u.display_name, u.avatar_url, r.role
ORDER BY pieces DESC, u.display_name ASC;

-- name: CountSecretPiecesForUser :one
SELECT COUNT(*)
FROM user_secrets us
WHERE us.user_id = $1
  AND us.secret_id = ANY(string_to_array($2::text, ',')::text[]);

-- name: GetSecretSolversLeaderboard :many
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COUNT(*) AS solved,
       MAX(us.unlocked_at)::timestamptz AS last_solved
FROM user_secrets us
JOIN users u ON us.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE us.secret_id = ANY(string_to_array($1::text, ',')::text[])
GROUP BY u.id, u.username, u.display_name, u.avatar_url, r.role
ORDER BY solved DESC, last_solved ASC;

-- name: GetSecretUserProgressSummary :one
SELECT u.id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM user_secrets us WHERE us.user_id = u.id AND us.secret_id = ANY(string_to_array($1::text, ',')::text[])) AS pieces
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE u.id = $2;

-- name: GetSecretCommentWithLikes :one
SELECT c.id, c.secret_id::text AS entity_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM secret_comment_likes l WHERE l.comment_id = c.id) AS like_count,
       FALSE::boolean AS user_liked
FROM secret_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE c.id = $1;

-- name: GetSecretCommenterIDs :many
SELECT DISTINCT c.user_id
FROM secret_comments c
WHERE c.secret_id = $1;

-- name: CountSecretCommentsBySecretIDs :many
SELECT c.secret_id, COUNT(*) AS comment_count
FROM secret_comments c
WHERE c.secret_id = ANY(string_to_array($1::text, ',')::text[])
GROUP BY c.secret_id;
