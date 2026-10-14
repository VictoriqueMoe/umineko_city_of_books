-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionUserID :one
SELECT s.user_id, s.expires_at FROM sessions s WHERE s.token = $1;

-- name: GetSessionAuthState :one
SELECT s.user_id, s.expires_at,
       (u.banned_at IS NOT NULL)::boolean AS banned,
       (u.locked_at IS NOT NULL)::boolean AS locked,
       (NOT u.email_verified AND NOW() >= u.verify_grace_until)::boolean AS requires_email_verification
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteSessionsForUserExcept :exec
DELETE FROM sessions WHERE user_id = $1 AND token <> $2;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < $1;
