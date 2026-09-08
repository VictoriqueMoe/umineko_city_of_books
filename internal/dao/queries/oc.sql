-- name: CreateOC :one
WITH o AS (
    INSERT INTO ocs (user_id, name, description, series, custom_series_name)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, user_id, name, description, series, custom_series_name, image_url, thumbnail_url, created_at, updated_at
)
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS vote_score,
       0::int AS user_vote,
       0::bigint AS favourite_count,
       FALSE::boolean AS user_favourited,
       0::bigint AS comment_count
FROM o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id;

-- name: GetOCByID :one
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = $1) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE o.id = $2;

-- name: ListOCsByUser :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE o.user_id = sqlc.arg(owner_id)
ORDER BY o.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountOCsByUser :one
SELECT COUNT(*) FROM ocs WHERE user_id = $1;

-- name: ListOCSummariesByUser :many
SELECT id, name, series, custom_series_name, thumbnail_url
FROM ocs
WHERE user_id = $1
ORDER BY lower(name) ASC;

-- name: UpdateOC :execrows
UPDATE ocs
SET name = $1, description = $2, series = $3, custom_series_name = $4, updated_at = NOW()
WHERE id = $5 AND user_id = $6;

-- name: UpdateOCAsAdmin :execrows
UPDATE ocs
SET name = $1, description = $2, series = $3, custom_series_name = $4, updated_at = NOW()
WHERE id = $5;

-- name: UpdateOCImage :exec
UPDATE ocs SET image_url = $1, thumbnail_url = $2 WHERE id = $3;

-- name: GetOCImagePaths :one
SELECT image_url, thumbnail_url FROM ocs WHERE id = $1;

-- name: OCNameExists :one
SELECT EXISTS(SELECT 1 FROM ocs o WHERE o.user_id = $1 AND lower(o.name) = lower($2::text)) AS oc_exists;

-- name: GetOCGalleryPaths :many
SELECT image_url, thumbnail_url FROM oc_images WHERE oc_id = $1 ORDER BY sort_order, id;

-- name: AddOCGalleryImage :one
INSERT INTO oc_images (oc_id, image_url, thumbnail_url, caption, sort_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateOCGalleryImageURL :exec
UPDATE oc_images SET image_url = $1 WHERE id = $2;

-- name: UpdateOCGalleryImageThumbnail :exec
UPDATE oc_images SET thumbnail_url = $1 WHERE id = $2;

-- name: UpdateOCGalleryImageCaption :execrows
UPDATE oc_images SET caption = $1 WHERE id = $2 AND oc_id = $3;

-- name: UpdateOCGalleryImageSortOrder :execrows
UPDATE oc_images SET sort_order = $1 WHERE id = $2 AND oc_id = $3;

-- name: UpdateOCGalleryImageDetails :execrows
UPDATE oc_images SET caption = $1, sort_order = $2 WHERE id = $3 AND oc_id = $4;

-- name: DeleteOCGalleryImage :execrows
DELETE FROM oc_images WHERE id = $1 AND oc_id = $2;

-- name: GetOCGallery :many
SELECT id, oc_id, image_url, thumbnail_url, caption, sort_order
FROM oc_images
WHERE oc_id = $1
ORDER BY sort_order ASC, id ASC;

-- name: GetOCGalleryBatch :many
SELECT id, oc_id, image_url, thumbnail_url, caption, sort_order
FROM oc_images
WHERE oc_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order ASC, id ASC;

-- name: CountOCs :one
SELECT COUNT(*)
FROM ocs o
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListOCsByNew :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY o.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListOCsByTop :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY COALESCE((SELECT SUM(sv.value) FROM oc_votes sv WHERE sv.oc_id = o.id), 0) DESC, o.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListOCsByCrack :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY COALESCE((SELECT SUM(sv.value) FROM oc_votes sv WHERE sv.oc_id = o.id), 0) ASC, o.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListOCsByFavourites :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM oc_favourites sf WHERE sf.oc_id = o.id) DESC, o.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListOCsByComments :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM oc_comments sc WHERE sc.oc_id = o.id) DESC, o.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListOCsByName :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY lower(o.name) ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListOCsByOld :many
SELECT o.id, o.user_id, o.name, o.description, o.series, o.custom_series_name,
       o.image_url, o.thumbnail_url, o.created_at, o.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM oc_votes v WHERE v.oc_id = o.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM oc_votes uv WHERE uv.oc_id = o.id AND uv.user_id = sqlc.arg(viewer_id)), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM oc_favourites f WHERE f.oc_id = o.id) AS favourite_count,
       EXISTS(SELECT 1 FROM oc_favourites uf WHERE uf.oc_id = o.id AND uf.user_id = sqlc.arg(viewer_id)) AS user_favourited,
       (SELECT COUNT(*) FROM oc_comments c WHERE c.oc_id = o.id) AS comment_count
FROM ocs o
JOIN users u ON o.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = o.user_id
WHERE (sqlc.arg(series)::text = '' OR o.series = sqlc.arg(series)::text)
  AND (sqlc.arg(custom_series_name)::text = '' OR lower(o.custom_series_name) = lower(sqlc.arg(custom_series_name)::text))
  AND (sqlc.arg(owner_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR o.user_id = sqlc.arg(owner_id)::uuid)
  AND (NOT sqlc.arg(crack_only)::boolean OR COALESCE((SELECT SUM(cv.value) FROM oc_votes cv WHERE cv.oc_id = o.id), 0) <= -3)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR o.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY o.created_at ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: FavouriteOC :exec
INSERT INTO oc_favourites (user_id, oc_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnfavouriteOC :exec
DELETE FROM oc_favourites WHERE user_id = $1 AND oc_id = $2;
