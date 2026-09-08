-- name: CreateEmailVerificationToken :exec
INSERT INTO email_verification_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetEmailVerificationByTokenHash :one
SELECT token_hash, user_id, expires_at, used_at, created_at FROM email_verification_tokens WHERE token_hash = $1;

-- name: MarkEmailVerificationUsed :exec
UPDATE email_verification_tokens SET used_at = NOW() WHERE token_hash = $1;

-- name: DeleteUnusedEmailVerificationsForUser :exec
DELETE FROM email_verification_tokens WHERE user_id = $1 AND used_at IS NULL;
