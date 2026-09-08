-- name: UnlockUserSecret :exec
INSERT INTO user_secrets (user_id, secret_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListUserSecretIDsForUser :many
SELECT us.secret_id FROM user_secrets us WHERE us.user_id = $1 ORDER BY us.secret_id;

-- name: ListUserIDsWithAnyUserSecret :many
SELECT DISTINCT us.user_id
FROM user_secrets us
WHERE us.secret_id = ANY(string_to_array($1::text, ',')::text[]);

-- name: GetUserSecretSolvedMarker :one
SELECT 1::int AS solved FROM user_secrets us WHERE us.secret_id = $1 LIMIT 1;

-- name: DeleteUserSecrets :exec
DELETE FROM user_secrets WHERE secret_id = ANY(string_to_array($1::text, ',')::text[]);

-- name: ListUserIDsWithUserSecret :many
SELECT us.user_id FROM user_secrets us WHERE us.secret_id = $1;
