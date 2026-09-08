-- name: DeleteOCVote :exec
DELETE FROM oc_votes WHERE user_id = $1 AND oc_id = $2;

-- name: UpsertOCVote :exec
INSERT INTO oc_votes (user_id, oc_id, value) VALUES ($1, $2, $3)
ON CONFLICT (user_id, oc_id) DO UPDATE SET value = EXCLUDED.value;
