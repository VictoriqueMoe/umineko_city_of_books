-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionUserID :one
SELECT s.user_id, s.expires_at FROM sessions s WHERE s.token = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteSessionsForUserExcept :exec
DELETE FROM sessions WHERE user_id = $1 AND token <> $2;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < $1;
