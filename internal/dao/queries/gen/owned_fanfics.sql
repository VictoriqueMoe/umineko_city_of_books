-- name: DeleteOwnedFanfic :execrows
DELETE FROM fanfics WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedFanficAsAdmin :exec
DELETE FROM fanfics WHERE id = $1;

-- name: GetOwnedFanficAuthorID :one
SELECT user_id FROM fanfics WHERE id = $1;

-- name: IncrementFanficViewCount :exec
UPDATE fanfics SET view_count = view_count + 1 WHERE id = $1;
