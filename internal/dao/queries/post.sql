-- name: CreatePost :one
WITH p AS (
    INSERT INTO posts (user_id, corner, body, shared_content_id, shared_content_type)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, user_id, corner, body, created_at, updated_at, view_count, shared_content_id, shared_content_type
)
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS like_count,
       0::bigint AS comment_count,
       FALSE::boolean AS user_liked,
       p.view_count,
       ''::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM p
JOIN users u ON u.id = p.user_id
LEFT JOIN user_roles r ON r.user_id = p.user_id;

-- name: UpdatePostAsAdmin :execrows
UPDATE posts SET body = $1, updated_at = NOW() WHERE id = $2;

-- name: UpdatePostAsOwner :execrows
UPDATE posts SET body = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3;

-- name: GetPostByID :one
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = $1) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.id = $2;

-- name: CountPostsByUser :one
SELECT COUNT(*) FROM posts WHERE user_id = $1;

-- name: ListPostsByUser :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = $1) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.user_id = $2
ORDER BY p.created_at DESC
LIMIT $3 OFFSET $4;

-- name: ResolvePostSuggestion :exec
INSERT INTO suggestion_resolved (post_id, resolved_by, status) VALUES ($1, $2, $3)
ON CONFLICT (post_id) DO UPDATE SET status = $3, resolved_by = $2, resolved_at = NOW();

-- name: UnresolvePostSuggestion :exec
DELETE FROM suggestion_resolved WHERE post_id = $1;

-- name: CountUserPostsToday :one
SELECT COUNT(*) FROM posts WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day';

-- name: GetPostCornerCounts :many
SELECT corner, COUNT(*) AS post_count FROM posts GROUP BY corner;

-- name: GetShareCount :one
SELECT COALESCE(share_count, 0)::int AS share_count
FROM share_counts
WHERE content_id = $1 AND content_type = $2;

-- name: ListShareCounts :many
SELECT content_id, share_count
FROM share_counts
WHERE content_id = ANY(string_to_array($1::text, ',')) AND content_type = $2;

-- name: IncrementShareCount :exec
INSERT INTO share_counts (content_id, content_type, share_count) VALUES ($1, $2, 1)
ON CONFLICT (content_id, content_type) DO UPDATE SET share_count = share_counts.share_count + 1;

-- name: DecrementShareCount :exec
UPDATE share_counts SET share_count = GREATEST(share_count - 1, 0) WHERE content_id = $1 AND content_type = $2;

-- name: GetPostSharedContentFields :one
SELECT shared_content_id, shared_content_type FROM posts WHERE id = $1;

-- name: ListSharedPostPreviews :many
SELECT p.id, p.body, p.user_id,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       p.corner
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: ListSharedPostPreviewMedia :many
SELECT post_id, media_url, media_type, thumbnail_url, sort_order, is_spoiler
FROM post_media
WHERE post_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order
LIMIT 4;

-- name: ListSharedArtPreviews :many
SELECT a.id, a.title, a.description, a.image_url, a.thumbnail_url, a.user_id,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       a.corner
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: ListSharedShipPreviews :many
SELECT s.id, s.title, s.description, s.image_url, s.thumbnail_url, s.user_id,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(sv.value) FROM ship_votes sv WHERE sv.ship_id = s.id), 0)::bigint AS vote_score
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE s.id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: ListSharedMysteryPreviews :many
SELECT m.id, m.title, m.body, m.difficulty, m.solved, m.user_id,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM mysteries m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = m.user_id
WHERE m.id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: ListSharedTheoryPreviews :many
SELECT t.id, t.title, t.body, t.series, t.credibility_score, t.user_id,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = t.user_id
WHERE t.id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: ListSharedFanficPreviews :many
SELECT f.id, f.title, f.summary, f.series, f.rating, f.cover_image_url, f.cover_thumbnail_url, f.word_count,
       (SELECT COUNT(*) FROM fanfic_chapters fc WHERE fc.fanfic_id = f.id) AS chapter_count,
       f.user_id, u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = f.user_id
WHERE f.id = ANY(string_to_array($1::text, ',')::uuid[]) AND f.status != 'draft';

-- name: CreatePostPoll :one
INSERT INTO post_polls (post_id, duration_seconds, expires_at)
VALUES ($1, $2, $3::text::timestamptz)
RETURNING id, post_id, duration_seconds, expires_at;

-- name: AddPostPollOption :exec
INSERT INTO post_poll_options (poll_id, label, sort_order) VALUES ($1::text::uuid, $2, $3);

-- name: GetPostPoll :one
SELECT id, post_id, duration_seconds, expires_at FROM post_polls WHERE post_id = $1;

-- name: ListPostPolls :many
SELECT id, post_id, duration_seconds, expires_at
FROM post_polls
WHERE post_id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: ListPostPollOptions :many
SELECT o.id, o.poll_id, o.label, o.sort_order,
       (SELECT COUNT(*) FROM post_poll_votes v WHERE v.option_id = o.id) AS vote_count
FROM post_poll_options o
WHERE o.poll_id = $1
ORDER BY o.sort_order;

-- name: ListPostPollOptionsByPollIDs :many
SELECT o.id, o.poll_id, o.label, o.sort_order,
       (SELECT COUNT(*) FROM post_poll_votes v WHERE v.option_id = o.id) AS vote_count
FROM post_poll_options o
WHERE o.poll_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY o.sort_order;

-- name: GetPostPollVote :one
SELECT option_id FROM post_poll_votes WHERE poll_id = $1 AND user_id = $2;

-- name: ListPostPollVotesByPollIDs :many
SELECT v.poll_id, v.option_id
FROM post_poll_votes v
WHERE v.poll_id = ANY(string_to_array($1::text, ',')::uuid[]) AND v.user_id = $2;

-- name: VotePostPoll :execrows
INSERT INTO post_poll_votes (poll_id, user_id, option_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: GetSharedPostAuthor :one
SELECT user_id FROM posts WHERE id = $1;

-- name: GetSharedArtAuthor :one
SELECT user_id FROM art WHERE id = $1;

-- name: GetSharedShipAuthor :one
SELECT user_id FROM ships WHERE id = $1;

-- name: GetSharedMysteryAuthor :one
SELECT user_id FROM mysteries WHERE id = $1;

-- name: GetSharedTheoryAuthor :one
SELECT user_id FROM theories WHERE id = $1;

-- name: GetSharedFanficAuthor :one
SELECT user_id FROM fanfics WHERE id = $1;

-- name: CountPostsFeed :one
SELECT COUNT(*)
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.corner = sqlc.arg(corner)
  AND (sqlc.arg(search)::text = ''
       OR p.body LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (CASE sqlc.arg(resolved_filter)::text
         WHEN 'open' THEN NOT EXISTS(SELECT 1 FROM suggestion_resolved so WHERE so.post_id = p.id)
         WHEN 'done' THEN EXISTS(SELECT 1 FROM suggestion_resolved sd WHERE sd.post_id = p.id AND sd.status = 'done')
         WHEN 'archived' THEN EXISTS(SELECT 1 FROM suggestion_resolved sa WHERE sa.post_id = p.id AND sa.status = 'archived')
         ELSE TRUE
       END)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListPostsFeedByNew :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (sqlc.arg(search)::text = ''
       OR p.body LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (CASE sqlc.arg(resolved_filter)::text
         WHEN 'open' THEN NOT EXISTS(SELECT 1 FROM suggestion_resolved so WHERE so.post_id = p.id)
         WHEN 'done' THEN EXISTS(SELECT 1 FROM suggestion_resolved sd WHERE sd.post_id = p.id AND sd.status = 'done')
         WHEN 'archived' THEN EXISTS(SELECT 1 FROM suggestion_resolved sa WHERE sa.post_id = p.id AND sa.status = 'archived')
         ELSE TRUE
       END)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListPostsFeedByLikes :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (sqlc.arg(search)::text = ''
       OR p.body LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (CASE sqlc.arg(resolved_filter)::text
         WHEN 'open' THEN NOT EXISTS(SELECT 1 FROM suggestion_resolved so WHERE so.post_id = p.id)
         WHEN 'done' THEN EXISTS(SELECT 1 FROM suggestion_resolved sd WHERE sd.post_id = p.id AND sd.status = 'done')
         WHEN 'archived' THEN EXISTS(SELECT 1 FROM suggestion_resolved sa WHERE sa.post_id = p.id AND sa.status = 'archived')
         ELSE TRUE
       END)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM post_likes ol WHERE ol.post_id = p.id) DESC, p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListPostsFeedByComments :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (sqlc.arg(search)::text = ''
       OR p.body LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (CASE sqlc.arg(resolved_filter)::text
         WHEN 'open' THEN NOT EXISTS(SELECT 1 FROM suggestion_resolved so WHERE so.post_id = p.id)
         WHEN 'done' THEN EXISTS(SELECT 1 FROM suggestion_resolved sd WHERE sd.post_id = p.id AND sd.status = 'done')
         WHEN 'archived' THEN EXISTS(SELECT 1 FROM suggestion_resolved sa WHERE sa.post_id = p.id AND sa.status = 'archived')
         ELSE TRUE
       END)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM post_comments oc WHERE oc.post_id = p.id) DESC, p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListPostsFeedByViews :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (sqlc.arg(search)::text = ''
       OR p.body LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (CASE sqlc.arg(resolved_filter)::text
         WHEN 'open' THEN NOT EXISTS(SELECT 1 FROM suggestion_resolved so WHERE so.post_id = p.id)
         WHEN 'done' THEN EXISTS(SELECT 1 FROM suggestion_resolved sd WHERE sd.post_id = p.id AND sd.status = 'done')
         WHEN 'archived' THEN EXISTS(SELECT 1 FROM suggestion_resolved sa WHERE sa.post_id = p.id AND sa.status = 'archived')
         ELSE TRUE
       END)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY p.view_count DESC, p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListPostsFeedByRelevance :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (sqlc.arg(search)::text = ''
       OR p.body LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (CASE sqlc.arg(resolved_filter)::text
         WHEN 'open' THEN NOT EXISTS(SELECT 1 FROM suggestion_resolved so WHERE so.post_id = p.id)
         WHEN 'done' THEN EXISTS(SELECT 1 FROM suggestion_resolved sd WHERE sd.post_id = p.id AND sd.status = 'done')
         WHEN 'archived' THEN EXISTS(SELECT 1 FROM suggestion_resolved sa WHERE sa.post_id = p.id AND sa.status = 'archived')
         ELSE TRUE
       END)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (
    (1.0
        + LEAST((SELECT COUNT(*) FROM post_likes ol WHERE ol.post_id = p.id), 50) * 0.15
        + LEAST((SELECT COUNT(*) FROM post_comments oc WHERE oc.post_id = p.id), 30) * 0.3
        + CASE WHEN EXISTS(SELECT 1 FROM follows fb WHERE fb.follower_id = sqlc.arg(viewer_id) AND fb.following_id = p.user_id) THEN 3.0 ELSE 0 END
    ) / (1.0 + EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 3600.0 * 0.3)
    + ((ascii(substr(p.id::text, 1, 1)) * 7 + ascii(substr(p.id::text, 5, 1)) * 13 + sqlc.arg(seed)::bigint) % 1000) / 2500.0
) DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: CountFollowingPostsFeed :one
SELECT COUNT(*)
FROM posts p
WHERE p.corner = sqlc.arg(corner)
  AND (p.user_id = sqlc.arg(viewer_id) OR p.user_id IN (SELECT f.following_id FROM follows f WHERE f.follower_id = sqlc.arg(viewer_id)))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListFollowingPostsByNew :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (p.user_id = sqlc.arg(viewer_id) OR p.user_id IN (SELECT f.following_id FROM follows f WHERE f.follower_id = sqlc.arg(viewer_id)))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListFollowingPostsByLikes :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (p.user_id = sqlc.arg(viewer_id) OR p.user_id IN (SELECT f.following_id FROM follows f WHERE f.follower_id = sqlc.arg(viewer_id)))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM post_likes ol WHERE ol.post_id = p.id) DESC, p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListFollowingPostsByComments :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (p.user_id = sqlc.arg(viewer_id) OR p.user_id IN (SELECT f.following_id FROM follows f WHERE f.follower_id = sqlc.arg(viewer_id)))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM post_comments oc WHERE oc.post_id = p.id) DESC, p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListFollowingPostsByViews :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (p.user_id = sqlc.arg(viewer_id) OR p.user_id IN (SELECT f.following_id FROM follows f WHERE f.follower_id = sqlc.arg(viewer_id)))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY p.view_count DESC, p.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListFollowingPostsByRelevance :many
SELECT p.id, p.user_id, p.corner, p.body, p.created_at, p.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id) AS like_count,
       (SELECT COUNT(*) FROM post_comments pc WHERE pc.post_id = p.id) AS comment_count,
       EXISTS(SELECT 1 FROM post_likes lu WHERE lu.post_id = p.id AND lu.user_id = sqlc.arg(viewer_id)) AS user_liked,
       p.view_count,
       COALESCE((SELECT sr.status FROM suggestion_resolved sr WHERE sr.post_id = p.id), '')::text AS resolved_status,
       p.shared_content_id,
       p.shared_content_type
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = p.user_id
WHERE p.corner = sqlc.arg(corner)
  AND (p.user_id = sqlc.arg(viewer_id) OR p.user_id IN (SELECT f.following_id FROM follows f WHERE f.follower_id = sqlc.arg(viewer_id)))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR p.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (
    (1.0
        + LEAST((SELECT COUNT(*) FROM post_likes ol WHERE ol.post_id = p.id), 50) * 0.15
        + LEAST((SELECT COUNT(*) FROM post_comments oc WHERE oc.post_id = p.id), 30) * 0.3
    ) / (1.0 + EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 3600.0 * 0.3)
    + ((ascii(substr(p.id::text, 1, 1)) * 7 + ascii(substr(p.id::text, 5, 1)) * 13 + sqlc.arg(seed)::bigint) % 1000) / 2500.0
) DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;
