-- name: AddGiphyFavourite :exec
INSERT INTO giphy_favourites (user_id, giphy_id, url, title, preview_url, width, height, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, giphy_id) DO UPDATE SET
    url = EXCLUDED.url,
    title = EXCLUDED.title,
    preview_url = EXCLUDED.preview_url,
    width = EXCLUDED.width,
    height = EXCLUDED.height,
    created_at = EXCLUDED.created_at;

-- name: RemoveGiphyFavourite :exec
DELETE FROM giphy_favourites WHERE user_id = $1 AND giphy_id = $2;

-- name: CountGiphyFavourites :one
SELECT COUNT(*) FROM giphy_favourites WHERE user_id = $1;

-- name: ListGiphyFavourites :many
SELECT giphy_id, url, title, preview_url, width, height, created_at
FROM giphy_favourites WHERE user_id = $1
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListGiphyFavouriteIDs :many
SELECT giphy_id FROM giphy_favourites WHERE user_id = $1;
