-- name: DeleteMysteryAttemptVote :exec
DELETE FROM mystery_attempt_votes WHERE user_id = $1 AND attempt_id = $2;

-- name: UpsertMysteryAttemptVote :exec
INSERT INTO mystery_attempt_votes (user_id, attempt_id, value) VALUES ($1, $2, $3)
ON CONFLICT (user_id, attempt_id) DO UPDATE SET value = EXCLUDED.value;
