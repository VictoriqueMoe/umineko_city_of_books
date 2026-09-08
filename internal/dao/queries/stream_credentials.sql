-- name: GetStreamCredentials :one
SELECT user_id, ingress_id, whip_url, stream_key, room
FROM stream_credentials
WHERE user_id = $1;

-- name: UpsertStreamCredentials :exec
INSERT INTO stream_credentials (user_id, ingress_id, whip_url, stream_key, room)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE
   SET ingress_id = excluded.ingress_id,
       whip_url = excluded.whip_url,
       stream_key = excluded.stream_key,
       room = excluded.room,
       updated_at = NOW();

-- name: DeleteStreamCredentials :exec
DELETE FROM stream_credentials WHERE user_id = $1;
