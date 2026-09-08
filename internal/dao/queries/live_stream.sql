-- name: CreateLiveStream :one
WITH ins AS (
    INSERT INTO live_streams (user_id, title, status)
    SELECT sqlc.arg(user_id)::uuid, sqlc.arg(title)::text, 'starting'
    WHERE (SELECT COUNT(*) FROM live_streams ls WHERE ls.status <> 'offline') < sqlc.arg(max_concurrent)::int
    RETURNING id, user_id, title, status, livekit_room, ingress_id, whip_url, stream_key,
              viewer_count, started_at, ended_at, created_at, thumbnail_url,
              egress_id, hls_playlist_url, default_mode
)
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM ins s
JOIN users u ON u.id = s.user_id;

-- name: GetLiveStreamByID :one
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM live_streams s
JOIN users u ON u.id = s.user_id
WHERE s.id = $1;

-- name: GetLiveStreamByRoom :one
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM live_streams s
JOIN users u ON u.id = s.user_id
WHERE s.livekit_room = $1;

-- name: GetActiveLiveStreamByUser :one
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM live_streams s
JOIN users u ON u.id = s.user_id
WHERE s.user_id = $1 AND s.status <> 'offline'
LIMIT 1;

-- name: GetActiveLiveStreamByUsername :one
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM live_streams s
JOIN users u ON u.id = s.user_id
WHERE LOWER(u.username) = LOWER(sqlc.arg(username)::text) AND s.status <> 'offline'
LIMIT 1;

-- name: ListLiveStreamsLive :many
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM live_streams s
JOIN users u ON u.id = s.user_id
WHERE s.status = 'live'
ORDER BY s.started_at DESC;

-- name: ListLiveStreamsStartingBefore :many
SELECT s.id, s.user_id, s.title, s.status, s.livekit_room, s.ingress_id,
       s.whip_url, s.stream_key, s.viewer_count, s.started_at, s.ended_at, s.created_at, s.thumbnail_url,
       s.egress_id, s.hls_playlist_url, s.default_mode,
       u.username, u.display_name, u.avatar_url
FROM live_streams s
JOIN users u ON u.id = s.user_id
WHERE s.status = 'starting' AND s.created_at < sqlc.arg(cutoff)::text::timestamptz;

-- name: CountActiveLiveStreams :one
SELECT COUNT(*) FROM live_streams WHERE status <> 'offline';

-- name: SetLiveStreamIngress :exec
UPDATE live_streams
SET ingress_id = $2, livekit_room = $3, whip_url = $4, stream_key = $5
WHERE id = $1;

-- name: MarkLiveStreamLive :exec
UPDATE live_streams
SET status = 'live', started_at = COALESCE(started_at, NOW())
WHERE id = $1 AND status <> 'offline';

-- name: MarkLiveStreamOffline :execrows
UPDATE live_streams
SET status = 'offline', ended_at = NOW(), viewer_count = 0
WHERE id = $1 AND status <> 'offline';

-- name: AdjustLiveStreamViewerCount :one
UPDATE live_streams
SET viewer_count = GREATEST(0, viewer_count + sqlc.arg(delta)::int)
WHERE id = sqlc.arg(id) AND status = 'live'
RETURNING viewer_count;

-- name: SetLiveStreamThumbnail :exec
UPDATE live_streams SET thumbnail_url = $2 WHERE id = $1;

-- name: SetLiveStreamEgress :exec
UPDATE live_streams SET egress_id = $2, hls_playlist_url = $3 WHERE id = $1;

-- name: SetLiveStreamDefaultMode :exec
UPDATE live_streams SET default_mode = $2 WHERE id = $1;

-- name: SetLiveStreamTitle :exec
UPDATE live_streams SET title = $2 WHERE id = $1;
