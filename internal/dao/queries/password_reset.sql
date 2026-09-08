-- name: CreatePasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetPasswordResetTokenByHash :one
SELECT token_hash, user_id, expires_at, used_at, created_at
FROM password_reset_tokens
WHERE token_hash = $1;

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1;

-- name: DeleteUnusedPasswordResetTokensForUser :exec
DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL;
