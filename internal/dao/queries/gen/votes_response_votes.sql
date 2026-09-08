-- name: DeleteResponseVote :exec
DELETE FROM response_votes WHERE user_id = $1 AND response_id = $2;

-- name: UpsertResponseVote :exec
INSERT INTO response_votes (user_id, response_id, value) VALUES ($1, $2, $3)
ON CONFLICT (user_id, response_id) DO UPDATE SET value = EXCLUDED.value;
