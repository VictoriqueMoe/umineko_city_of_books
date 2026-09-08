-- name: CreateGameRoom :one
INSERT INTO game_rooms (game_type, status, state_json, created_by)
VALUES ($1, 'pending', $2, $3)
RETURNING id, game_type, status, state_json, turn_user_id, winner_user_id,
          COALESCE(result, '')::text AS result,
          created_by, created_at, updated_at, finished_at;

-- name: AddGameRoomPlayerJoined :exec
INSERT INTO game_room_players (room_id, user_id, slot, joined, joined_at)
VALUES ($1, $2, $3, TRUE, NOW());

-- name: AddGameRoomPlayerPending :exec
INSERT INTO game_room_players (room_id, user_id, slot, joined)
VALUES ($1, $2, $3, FALSE);

-- name: GetGameRoom :one
SELECT id, game_type, status, state_json, turn_user_id, winner_user_id,
       COALESCE(result, '')::text AS result,
       created_by, created_at, updated_at, finished_at
FROM game_rooms
WHERE id = $1;

-- name: GetGameRoomPlayers :many
SELECT user_id, slot, joined, joined_at, last_seen_at
FROM game_room_players
WHERE room_id = $1
ORDER BY slot;

-- name: CountGameRoomParticipant :one
SELECT COUNT(*) FROM game_room_players WHERE room_id = $1 AND user_id = $2;

-- name: GetGameRoomPlayerSlot :one
SELECT slot FROM game_room_players WHERE room_id = $1 AND user_id = $2;

-- name: SetGameRoomPlayerJoined :exec
UPDATE game_room_players
SET joined = TRUE, joined_at = COALESCE(joined_at, NOW()), last_seen_at = NOW()
WHERE room_id = $1 AND user_id = $2;

-- name: TouchGameRoomPlayerSeen :exec
UPDATE game_room_players SET last_seen_at = NOW() WHERE room_id = $1 AND user_id = $2;

-- name: SetGameRoomStatus :exec
UPDATE game_rooms SET status = $1, updated_at = NOW() WHERE id = $2;

-- name: SetGameRoomState :exec
UPDATE game_rooms
SET state_json = $1, turn_user_id = $2, updated_at = NOW()
WHERE id = $3 AND status = 'active';

-- name: FinishGameRoom :execrows
UPDATE game_rooms
SET status = $1, winner_user_id = $2, result = $3, state_json = $4, finished_at = NOW(), updated_at = NOW()
WHERE id = $5 AND status = 'active';

-- name: AppendGameRoomMove :exec
INSERT INTO game_room_moves (room_id, ply, user_id, action_json) VALUES ($1, $2, $3, $4);

-- name: ListGameRoomMoves :many
SELECT ply, user_id, action_json, created_at
FROM game_room_moves
WHERE room_id = $1
ORDER BY ply;

-- name: GetGameRoomNextPly :one
SELECT COALESCE(MAX(ply) + 1, 0)::int AS next_ply FROM game_room_moves WHERE room_id = $1;

-- name: CountLiveGameRooms :one
SELECT COUNT(*) FROM game_rooms
WHERE status = 'active'
  AND ($1::text = '' OR game_type = $1::text);

-- name: ListLiveGameRooms :many
SELECT id, game_type, status, state_json, turn_user_id, winner_user_id,
       COALESCE(result, '')::text AS result,
       created_by, created_at, updated_at, finished_at
FROM game_rooms
WHERE status = 'active'
  AND ($1::text = '' OR game_type = $1::text)
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFinishedGameRooms :one
SELECT COUNT(*) FROM game_rooms
WHERE status IN ('finished', 'abandoned')
  AND ($1::text = '' OR game_type = $1::text);

-- name: ListFinishedGameRooms :many
SELECT id, game_type, status, state_json, turn_user_id, winner_user_id,
       COALESCE(result, '')::text AS result,
       created_by, created_at, updated_at, finished_at
FROM game_rooms
WHERE status IN ('finished', 'abandoned')
  AND ($1::text = '' OR game_type = $1::text)
ORDER BY finished_at DESC
LIMIT $2 OFFSET $3;

-- name: CountActiveGameRooms :one
SELECT COUNT(*) FROM game_rooms WHERE status = 'active';

-- name: GetGameRoomScoreboard :many
SELECT s.user_id, s.wins, s.losses, s.draws
FROM (
    SELECT p.user_id AS user_id,
           SUM(CASE WHEN r.winner_user_id = p.user_id THEN 1 ELSE 0 END)::bigint AS wins,
           SUM(CASE WHEN r.winner_user_id IS NOT NULL AND r.winner_user_id != p.user_id THEN 1 ELSE 0 END)::bigint AS losses,
           SUM(CASE WHEN r.winner_user_id IS NULL THEN 1 ELSE 0 END)::bigint AS draws
    FROM game_room_players p
    JOIN game_rooms r ON r.id = p.room_id
    WHERE r.game_type = $1 AND r.status IN ('finished', 'abandoned') AND p.joined = TRUE
    GROUP BY p.user_id
) s
ORDER BY s.wins DESC, (s.wins - s.losses) DESC;

-- name: GetGameRoomTopWinnerIDs :many
WITH ranked AS (
    SELECT p.user_id AS user_id,
           SUM(CASE WHEN r.winner_user_id = p.user_id THEN 1 ELSE 0 END)::bigint AS wins,
           SUM(CASE WHEN r.winner_user_id IS NOT NULL AND r.winner_user_id != p.user_id THEN 1 ELSE 0 END)::bigint AS losses
    FROM game_room_players p
    JOIN game_rooms r ON r.id = p.room_id
    WHERE r.game_type = $1 AND r.status IN ('finished', 'abandoned') AND p.joined = TRUE
    GROUP BY p.user_id
    HAVING SUM(CASE WHEN r.winner_user_id = p.user_id THEN 1 ELSE 0 END) > 0
)
SELECT rk.user_id FROM ranked rk
WHERE rk.wins = (SELECT MAX(rw.wins) FROM ranked rw)
  AND (rk.wins - rk.losses) = (
      SELECT MAX(rd.wins - rd.losses) FROM ranked rd
      WHERE rd.wins = (SELECT MAX(rm.wins) FROM ranked rm)
  );

-- name: CancelIdleGameRoom :execrows
UPDATE game_rooms
SET status = 'abandoned', result = 'timeout', finished_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'active' AND updated_at < $2;

-- name: ListIdleActiveGameRooms :many
SELECT id, game_type, status, state_json, turn_user_id, winner_user_id,
       COALESCE(result, '')::text AS result,
       created_by, created_at, updated_at, finished_at
FROM game_rooms
WHERE status = 'active' AND updated_at < $1;

-- name: CountGameRoomsForUser :one
SELECT COUNT(*) FROM game_rooms r
WHERE EXISTS (SELECT 1 FROM game_room_players p WHERE p.room_id = r.id AND p.user_id = $1)
  AND ($2::text = '' OR r.game_type = $2::text)
  AND ($3::text = '' OR r.status = ANY(string_to_array($3::text, ',')));

-- name: ListGameRoomsForUser :many
SELECT r.id, r.game_type, r.status, r.state_json, r.turn_user_id, r.winner_user_id,
       COALESCE(r.result, '')::text AS result,
       r.created_by, r.created_at, r.updated_at, r.finished_at
FROM game_rooms r
WHERE EXISTS (SELECT 1 FROM game_room_players p WHERE p.room_id = r.id AND p.user_id = $1)
  AND ($2::text = '' OR r.game_type = $2::text)
  AND ($3::text = '' OR r.status = ANY(string_to_array($3::text, ',')))
ORDER BY r.updated_at DESC
LIMIT $4 OFFSET $5;
