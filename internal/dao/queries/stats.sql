-- name: StatsCountUsers :one
SELECT COUNT(*) FROM users WHERE NOT is_bot;

-- name: StatsCountTheories :one
SELECT COUNT(*) FROM theories;

-- name: StatsCountResponses :one
SELECT COUNT(*) FROM responses;

-- name: StatsCountVotes :one
SELECT ((SELECT COUNT(*) FROM theory_votes) + (SELECT COUNT(*) FROM response_votes))::bigint AS total_votes;

-- name: StatsCountPosts :one
SELECT COUNT(*) FROM posts;

-- name: StatsCountComments :one
SELECT COUNT(*) FROM post_comments;

-- name: StatsRecentUsers :one
SELECT
    (COUNT(*) FILTER (WHERE u.created_at > NOW() - INTERVAL '1 day'))::bigint AS day_count,
    (COUNT(*) FILTER (WHERE u.created_at > NOW() - INTERVAL '7 days'))::bigint AS week_count,
    (COUNT(*) FILTER (WHERE u.created_at > NOW() - INTERVAL '30 days'))::bigint AS month_count
FROM users u
WHERE NOT u.is_bot;

-- name: StatsRecentTheories :one
SELECT
    (COUNT(*) FILTER (WHERE t.created_at > NOW() - INTERVAL '1 day'))::bigint AS day_count,
    (COUNT(*) FILTER (WHERE t.created_at > NOW() - INTERVAL '7 days'))::bigint AS week_count,
    (COUNT(*) FILTER (WHERE t.created_at > NOW() - INTERVAL '30 days'))::bigint AS month_count
FROM theories t;

-- name: StatsRecentResponses :one
SELECT
    (COUNT(*) FILTER (WHERE r.created_at > NOW() - INTERVAL '1 day'))::bigint AS day_count,
    (COUNT(*) FILTER (WHERE r.created_at > NOW() - INTERVAL '7 days'))::bigint AS week_count,
    (COUNT(*) FILTER (WHERE r.created_at > NOW() - INTERVAL '30 days'))::bigint AS month_count
FROM responses r;

-- name: StatsRecentPosts :one
SELECT
    (COUNT(*) FILTER (WHERE p.created_at > NOW() - INTERVAL '1 day'))::bigint AS day_count,
    (COUNT(*) FILTER (WHERE p.created_at > NOW() - INTERVAL '7 days'))::bigint AS week_count,
    (COUNT(*) FILTER (WHERE p.created_at > NOW() - INTERVAL '30 days'))::bigint AS month_count
FROM posts p;

-- name: StatsPostsByCorner :many
SELECT p.corner, COUNT(*)::bigint AS post_count
FROM posts p
GROUP BY p.corner;

-- name: StatsMostActiveUsers :many
SELECT u.id, u.username, u.display_name, u.avatar_url, COUNT(*)::bigint AS action_count
FROM (
    SELECT t.user_id FROM theories t
    UNION ALL
    SELECT r.user_id FROM responses r
    UNION ALL
    SELECT p.user_id FROM posts p
    UNION ALL
    SELECT c.user_id FROM post_comments c
) actions
JOIN users u ON actions.user_id = u.id
GROUP BY u.id
ORDER BY action_count DESC
LIMIT $1;
