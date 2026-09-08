-- name: DeleteTheoryVote :exec
DELETE FROM theory_votes WHERE user_id = $1 AND theory_id = $2;

-- name: UpsertTheoryVote :exec
INSERT INTO theory_votes (user_id, theory_id, value) VALUES ($1, $2, $3)
ON CONFLICT (user_id, theory_id) DO UPDATE SET value = EXCLUDED.value;
