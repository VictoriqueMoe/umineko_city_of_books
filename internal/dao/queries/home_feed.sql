-- name: ListHomeRecentActivity :many
WITH feed AS (
    SELECT 'theory'::text AS kind, t.id AS id, t.title AS title, substr(t.body, 1, 200)::text AS body,
           t.series AS corner, t.created_at AS created_at, t.user_id AS author_id
    FROM theories t
    UNION ALL
    SELECT 'post'::text AS kind, p.id AS id, ''::text AS title, substr(p.body, 1, 200)::text AS body,
           p.corner AS corner, p.created_at AS created_at, p.user_id AS author_id
    FROM posts p
    UNION ALL
    SELECT 'journal'::text AS kind, j.id AS id, j.title AS title,
           substr(COALESCE((SELECT je.body FROM journal_entries je WHERE je.journal_id = j.id AND NOT je.is_draft ORDER BY je.entry_number DESC LIMIT 1), ''), 1, 200)::text AS body,
           j.work AS corner, j.created_at AS created_at, j.user_id AS author_id
    FROM journals j
    WHERE j.archived_at IS NULL
    UNION ALL
    SELECT 'art'::text AS kind, a.id AS id, a.title AS title, substr(a.description, 1, 200)::text AS body,
           a.corner AS corner, a.created_at AS created_at, a.user_id AS author_id
    FROM art a
)
SELECT f.kind, f.id, f.title, f.body, f.corner, f.created_at,
       f.author_id, u.username, u.display_name, u.avatar_url
FROM feed f
JOIN users u ON u.id = f.author_id
WHERE u.banned_at IS NULL
ORDER BY f.created_at DESC
LIMIT $1;

-- name: ListHomeEchoes :many
WITH feed AS (
    SELECT 'theory'::text AS kind, t.id AS id, t.title AS title, substr(t.body, 1, 200)::text AS body,
           t.series AS corner, t.episode AS episode, FALSE::boolean AS is_spoiler, t.created_at AS created_at, t.user_id AS author_id
    FROM theories t
    UNION ALL
    SELECT 'post'::text AS kind, p.id AS id, ''::text AS title, substr(p.body, 1, 200)::text AS body,
           p.corner AS corner, 0::integer AS episode, FALSE::boolean AS is_spoiler, p.created_at AS created_at, p.user_id AS author_id
    FROM posts p
    UNION ALL
    SELECT 'journal'::text AS kind, j.id AS id, j.title AS title,
           substr(COALESCE((SELECT je.body FROM journal_entries je WHERE je.journal_id = j.id AND NOT je.is_draft ORDER BY je.entry_number DESC LIMIT 1), ''), 1, 200)::text AS body,
           j.work AS corner, 0::integer AS episode, FALSE::boolean AS is_spoiler, j.created_at AS created_at, j.user_id AS author_id
    FROM journals j
    WHERE j.archived_at IS NULL
    UNION ALL
    SELECT 'art'::text AS kind, a.id AS id, a.title AS title, substr(a.description, 1, 200)::text AS body,
           a.corner AS corner, 0::integer AS episode, a.is_spoiler AS is_spoiler, a.created_at AS created_at, a.user_id AS author_id
    FROM art a
)
SELECT f.kind, f.id, f.title, f.body, f.corner, f.episode, f.is_spoiler, f.created_at,
       f.author_id, u.username, u.display_name, u.avatar_url
FROM feed f
JOIN users u ON u.id = f.author_id
WHERE u.banned_at IS NULL AND NOT u.is_bot AND u.echoes_enabled
  AND f.created_at >= (CURRENT_DATE - sqlc.arg(ago)::text::interval)
  AND f.created_at < (CURRENT_DATE - sqlc.arg(ago)::text::interval + INTERVAL '1 day')
ORDER BY f.created_at DESC
LIMIT sqlc.arg(row_limit)::int;

-- name: ListHomeRecentMembers :many
SELECT id, username, display_name, avatar_url, created_at
FROM users
WHERE banned_at IS NULL AND NOT is_bot
ORDER BY created_at DESC
LIMIT $1;

-- name: ListHomeCornerActivity24h :many
SELECT p.corner,
       COUNT(*) AS post_count,
       COUNT(DISTINCT p.user_id) AS unique_posters,
       MAX(p.created_at)::timestamptz AS last_post_at
FROM posts p
JOIN users u ON u.id = p.user_id
WHERE p.created_at > NOW() - INTERVAL '1 day' AND u.banned_at IS NULL
GROUP BY p.corner;

-- name: ListHomeSidebarActivity :many
SELECT ('game_board_' || corner)::text AS key, MAX(created_at)::timestamptz AS latest_at FROM posts GROUP BY corner
UNION ALL
SELECT ('gallery_' || corner)::text AS key, MAX(created_at)::timestamptz AS latest_at FROM art GROUP BY corner
UNION ALL
SELECT ('theories_' || series)::text AS key, MAX(created_at)::timestamptz AS latest_at FROM theories GROUP BY series
UNION ALL
SELECT 'mysteries'::text AS key, MAX(created_at)::timestamptz AS latest_at FROM mysteries HAVING COUNT(*) > 0
UNION ALL
SELECT 'secrets'::text AS key, MAX(created_at)::timestamptz AS latest_at FROM secret_comments HAVING COUNT(*) > 0
UNION ALL
SELECT 'ships'::text AS key, MAX(created_at)::timestamptz AS latest_at FROM ships HAVING COUNT(*) > 0
UNION ALL
SELECT 'fanfiction'::text AS key, MAX(created_at)::timestamptz AS latest_at FROM fanfics HAVING COUNT(*) > 0
UNION ALL
SELECT 'journals'::text AS key, MAX(created_at)::timestamptz AS latest_at FROM journals WHERE archived_at IS NULL HAVING COUNT(*) > 0
UNION ALL
SELECT 'rooms'::text AS key, MAX(created_at)::timestamptz AS latest_at FROM chat_rooms WHERE type = 'group' AND is_public = TRUE AND is_system = FALSE HAVING COUNT(*) > 0;

-- name: ListHomePublicRooms :many
SELECT cr.id, cr.name, cr.description,
       (SELECT COUNT(*) FROM chat_room_members m WHERE m.room_id = cr.id) AS member_count,
       cr.last_message_at
FROM chat_rooms cr
WHERE cr.type = 'group' AND cr.is_public = TRUE AND cr.is_system = FALSE AND cr.archived_at IS NULL
ORDER BY COALESCE(cr.last_message_at, cr.created_at) DESC
LIMIT $1;
