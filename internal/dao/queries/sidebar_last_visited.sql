-- name: UpsertSidebarLastVisited :exec
INSERT INTO sidebar_last_visited (user_id, key, visited_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id, key) DO UPDATE SET visited_at = EXCLUDED.visited_at;

-- name: ListSidebarLastVisitedForUser :many
SELECT key, visited_at FROM sidebar_last_visited WHERE user_id = $1;
