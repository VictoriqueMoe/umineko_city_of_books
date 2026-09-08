-- name: UpsertDeviceToken :exec
INSERT INTO device_tokens (token, user_id, platform, last_seen)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (token) DO UPDATE
   SET user_id = excluded.user_id,
       platform = excluded.platform,
       last_seen = NOW();

-- name: ListDeviceTokensForUser :many
SELECT token, platform FROM device_tokens WHERE user_id = $1;

-- name: DeleteDeviceToken :exec
DELETE FROM device_tokens WHERE token = $1 AND user_id = $2;
