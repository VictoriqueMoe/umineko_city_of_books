-- name: DeleteOwnedOC :execrows
DELETE FROM ocs WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedOCAsAdmin :exec
DELETE FROM ocs WHERE id = $1;

-- name: GetOwnedOCAuthorID :one
SELECT user_id FROM ocs WHERE id = $1;

-- name: IncrementOCViewCount :exec
UPDATE ocs SET view_count = view_count + 1 WHERE id = $1;
