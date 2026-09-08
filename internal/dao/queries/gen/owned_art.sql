-- name: DeleteOwnedArt :execrows
DELETE FROM art WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedArtAsAdmin :exec
DELETE FROM art WHERE id = $1;

-- name: GetOwnedArtAuthorID :one
SELECT user_id FROM art WHERE id = $1;

-- name: IncrementArtViewCount :exec
UPDATE art SET view_count = view_count + 1 WHERE id = $1;
