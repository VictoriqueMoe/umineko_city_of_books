-- name: DeleteOwnedPost :execrows
DELETE FROM posts WHERE id = $1 AND user_id = $2;

-- name: DeleteOwnedPostAsAdmin :exec
DELETE FROM posts WHERE id = $1;

-- name: GetOwnedPostAuthorID :one
SELECT user_id FROM posts WHERE id = $1;

-- name: IncrementPostViewCount :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = $1;
