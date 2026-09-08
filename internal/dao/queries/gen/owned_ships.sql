-- name: DeleteOwnedShip :execrows
DELETE FROM ships WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedShipAsAdmin :exec
DELETE FROM ships WHERE id = $1;

-- name: GetOwnedShipAuthorID :one
SELECT user_id FROM ships WHERE id = $1;

-- name: IncrementShipViewCount :exec
UPDATE ships SET view_count = view_count + 1 WHERE id = $1;
