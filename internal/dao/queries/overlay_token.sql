-- name: GetOverlayTokenByUser :one
SELECT token FROM overlay_tokens WHERE user_id = $1;

-- name: GetOverlayTokenUser :one
SELECT user_id FROM overlay_tokens WHERE token = $1;

-- name: UpsertOverlayToken :exec
INSERT INTO overlay_tokens (user_id, token)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
   SET token = excluded.token,
       updated_at = NOW();

-- name: DeleteOverlayToken :exec
DELETE FROM overlay_tokens WHERE user_id = $1;
