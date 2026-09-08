-- name: DeleteOwnedJournal :execrows
DELETE FROM journals WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedJournalAsAdmin :exec
DELETE FROM journals WHERE id = $1;

-- name: GetOwnedJournalAuthorID :one
SELECT user_id FROM journals WHERE id = $1;

-- name: IncrementJournalViewCount :exec
UPDATE journals SET view_count = view_count + 1 WHERE id = $1;
