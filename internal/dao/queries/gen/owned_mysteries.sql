-- name: DeleteOwnedMystery :execrows
DELETE FROM mysteries WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedMysteryAsAdmin :exec
DELETE FROM mysteries WHERE id = $1;

-- name: GetOwnedMysteryAuthorID :one
SELECT user_id FROM mysteries WHERE id = $1;

-- name: IncrementMysteryViewCount :exec
UPDATE mysteries SET view_count = view_count + 1 WHERE id = $1;
