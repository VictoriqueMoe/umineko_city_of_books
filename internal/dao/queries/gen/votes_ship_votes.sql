-- name: DeleteShipVote :exec
DELETE FROM ship_votes WHERE user_id = $1 AND ship_id = $2;

-- name: UpsertShipVote :exec
INSERT INTO ship_votes (user_id, ship_id, value) VALUES ($1, $2, $3)
ON CONFLICT (user_id, ship_id) DO UPDATE SET value = EXCLUDED.value;
