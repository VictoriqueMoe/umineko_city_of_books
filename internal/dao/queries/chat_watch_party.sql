-- name: CreateWatchPartySession :one
INSERT INTO chat_watch_party_sessions
    (room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url, title, type, start_url, region, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active')
RETURNING id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
          title, type, start_url, region, status, started_at, ended_at, ended_reason;

-- name: ListActiveWatchPartiesByRoom :many
SELECT id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
       title, type, start_url, region, status, started_at, ended_at, ended_reason
FROM chat_watch_party_sessions
WHERE room_id = $1 AND status = 'active'
ORDER BY started_at ASC;

-- name: GetWatchPartySessionByID :one
SELECT id, room_id, started_by, controller_id, hyperbeam_session_id, hyperbeam_admin_token, embed_url, vm_base_url,
       title, type, start_url, region, status, started_at, ended_at, ended_reason
FROM chat_watch_party_sessions
WHERE id = $1;

-- name: ListIdleActiveWatchPartySessions :many
SELECT s.id, s.room_id, s.started_by, s.controller_id, s.hyperbeam_session_id, s.hyperbeam_admin_token,
       s.embed_url, s.vm_base_url, s.title, s.type, s.start_url, s.region, s.status, s.started_at, s.ended_at, s.ended_reason
FROM chat_watch_party_sessions s
WHERE s.status = 'active'
  AND s.started_at < $1::text::timestamptz
  AND NOT EXISTS (
      SELECT 1 FROM chat_watch_party_participants p
       WHERE p.session_id = s.id AND p.left_at IS NULL
  );

-- name: EndWatchPartySession :exec
UPDATE chat_watch_party_sessions
   SET status = 'ended', ended_at = NOW(), ended_reason = $2
 WHERE id = $1 AND status = 'active';

-- name: SetWatchPartyControllerID :exec
UPDATE chat_watch_party_sessions SET controller_id = $2 WHERE id = $1;

-- name: UpsertWatchPartyParticipant :exec
INSERT INTO chat_watch_party_participants (session_id, user_id, has_control, hyperbeam_identifier)
VALUES ($1, $2, $3, $4)
ON CONFLICT (session_id, user_id) DO UPDATE SET
    has_control = EXCLUDED.has_control,
    hyperbeam_identifier = EXCLUDED.hyperbeam_identifier,
    left_at = NULL,
    joined_at = NOW();

-- name: SetWatchPartyParticipantIdentifier :exec
UPDATE chat_watch_party_participants SET hyperbeam_identifier = $3 WHERE session_id = $1 AND user_id = $2;

-- name: MarkWatchPartyParticipantLeft :exec
UPDATE chat_watch_party_participants
   SET left_at = NOW(), has_control = FALSE
 WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: MarkAllWatchPartyParticipantsLeft :exec
UPDATE chat_watch_party_participants
   SET left_at = NOW(), has_control = FALSE
 WHERE session_id = $1 AND left_at IS NULL;

-- name: GetActiveWatchPartyParticipants :many
SELECT p.session_id, p.user_id, u.username, u.display_name, u.avatar_url, p.has_control, p.hyperbeam_identifier, p.joined_at, p.left_at
FROM chat_watch_party_participants p
JOIN users u ON u.id = p.user_id
WHERE p.session_id = $1 AND p.left_at IS NULL
ORDER BY p.joined_at ASC;

-- name: GetWatchPartyParticipant :one
SELECT p.session_id, p.user_id, u.username, u.display_name, u.avatar_url, p.has_control, p.hyperbeam_identifier, p.joined_at, p.left_at
FROM chat_watch_party_participants p
JOIN users u ON u.id = p.user_id
WHERE p.session_id = $1 AND p.user_id = $2
LIMIT 1;

-- name: SetWatchPartyParticipantControl :exec
UPDATE chat_watch_party_participants SET has_control = $3 WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: CountActiveWatchPartyParticipants :one
SELECT COUNT(*) FROM chat_watch_party_participants WHERE session_id = $1 AND left_at IS NULL;
