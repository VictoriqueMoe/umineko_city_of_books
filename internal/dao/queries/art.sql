-- name: CreateArt :one
WITH a AS (
    INSERT INTO art (user_id, corner, art_type, title, description, image_url, thumbnail_url, is_spoiler)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, user_id, corner, art_type, title, description, image_url, thumbnail_url,
              gallery_id, view_count, is_spoiler, created_at, updated_at
)
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS like_count,
       0::bigint AS comment_count,
       a.view_count,
       FALSE::boolean AS user_liked,
       a.is_spoiler
FROM a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id;

-- name: GetArtByID :one
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id) AS like_count,
       (SELECT COUNT(*) FROM art_comments ac WHERE ac.art_id = a.id) AS comment_count,
       a.view_count,
       EXISTS(SELECT 1 FROM art_likes ul WHERE ul.art_id = a.id AND ul.user_id = $1) AS user_liked,
       a.is_spoiler
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.id = $2;

-- name: CountArtByUser :one
SELECT COUNT(*) FROM art WHERE user_id = $1;

-- name: ListArtByUser :many
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id) AS like_count,
       (SELECT COUNT(*) FROM art_comments ac WHERE ac.art_id = a.id) AS comment_count,
       a.view_count,
       EXISTS(SELECT 1 FROM art_likes ul WHERE ul.art_id = a.id AND ul.user_id = sqlc.arg(viewer_id)) AS user_liked,
       a.is_spoiler
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.user_id = sqlc.arg(user_id)
ORDER BY a.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: CountArtInGallery :one
SELECT COUNT(*) FROM art WHERE gallery_id = sqlc.arg(gallery_id)::uuid;

-- name: ListArtInGallery :many
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id) AS like_count,
       (SELECT COUNT(*) FROM art_comments ac WHERE ac.art_id = a.id) AS comment_count,
       a.view_count,
       EXISTS(SELECT 1 FROM art_likes ul WHERE ul.art_id = a.id AND ul.user_id = sqlc.arg(viewer_id)) AS user_liked,
       a.is_spoiler
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.gallery_id = sqlc.arg(gallery_id)::uuid
ORDER BY a.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: UpdateArt :execrows
UPDATE art SET title = $1, description = $2, is_spoiler = $3, updated_at = NOW() WHERE id = $4 AND user_id = $5;

-- name: UpdateArtAsAdmin :execrows
UPDATE art SET title = $1, description = $2, is_spoiler = $3, updated_at = NOW() WHERE id = $4;

-- name: GetArtImageURL :one
SELECT image_url FROM art WHERE id = $1;

-- name: GetArtImagePaths :one
SELECT image_url, thumbnail_url FROM art WHERE id = $1;

-- name: ListGalleryArtImages :many
SELECT id, image_url, thumbnail_url
FROM art
WHERE gallery_id = sqlc.arg(gallery_id)::uuid AND user_id = sqlc.arg(user_id);

-- name: CountUserArtToday :one
SELECT COUNT(*) FROM art WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day';

-- name: GetArtCornerCounts :many
SELECT corner, COUNT(*) AS count FROM art GROUP BY corner;

-- name: InsertArtTag :exec
INSERT INTO art_tags (art_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteArtTags :exec
DELETE FROM art_tags WHERE art_id = $1;

-- name: GetArtTags :many
SELECT tag FROM art_tags WHERE art_id = $1 ORDER BY tag;

-- name: GetArtTagsBatch :many
SELECT art_id, tag
FROM art_tags
WHERE art_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY tag;

-- name: GetPopularArtTags :many
SELECT t.tag, COUNT(*) AS cnt
FROM art_tags t
JOIN art a ON t.art_id = a.id
WHERE ($1::text = '' OR a.corner = $1::text)
GROUP BY t.tag
ORDER BY cnt DESC
LIMIT $2;

-- name: SetArtGallery :execrows
UPDATE art
SET gallery_id = sqlc.narg(gallery_id)::uuid
WHERE art.id = sqlc.arg(art_id)
  AND art.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(gallery_id)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM galleries g WHERE g.id = sqlc.narg(gallery_id)::uuid AND g.user_id = sqlc.arg(user_id)));

-- name: CreateGallery :one
WITH g AS (
    INSERT INTO galleries (user_id, name, description)
    VALUES ($1, $2, $3)
    RETURNING id, user_id, name, description, cover_art_id, created_at, updated_at
)
SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
       ''::text AS cover_image_url,
       ''::text AS cover_thumbnail_url,
       0::bigint AS art_count,
       g.created_at, g.updated_at,
       u.username, u.display_name, u.avatar_url
FROM g
JOIN users u ON g.user_id = u.id;

-- name: UpdateGallery :execrows
UPDATE galleries SET name = $1, description = $2, updated_at = NOW() WHERE id = $3 AND user_id = $4;

-- name: SetGalleryCover :execrows
UPDATE galleries
SET cover_art_id = sqlc.narg(cover_art_id)::uuid, updated_at = NOW()
WHERE galleries.id = sqlc.arg(gallery_id)
  AND galleries.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(cover_art_id)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM art a WHERE a.id = sqlc.narg(cover_art_id)::uuid AND a.user_id = sqlc.arg(user_id)));

-- name: DeleteArtInGallery :exec
DELETE FROM art WHERE gallery_id = sqlc.arg(gallery_id)::uuid AND user_id = sqlc.arg(user_id);

-- name: DeleteGalleryRow :execrows
DELETE FROM galleries WHERE id = $1 AND user_id = $2;

-- name: GetGalleryByID :one
SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
       COALESCE(a.image_url, '')::text AS cover_image_url,
       COALESCE(a.thumbnail_url, '')::text AS cover_thumbnail_url,
       (SELECT COUNT(*) FROM art ar WHERE ar.gallery_id = g.id) AS art_count,
       g.created_at, g.updated_at,
       u.username, u.display_name, u.avatar_url
FROM galleries g
JOIN users u ON g.user_id = u.id
LEFT JOIN art a ON g.cover_art_id = a.id
WHERE g.id = $1;

-- name: ListGalleriesByUser :many
SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
       COALESCE(a.image_url, '')::text AS cover_image_url,
       COALESCE(a.thumbnail_url, '')::text AS cover_thumbnail_url,
       (SELECT COUNT(*) FROM art ar WHERE ar.gallery_id = g.id) AS art_count,
       g.created_at, g.updated_at,
       u.username, u.display_name, u.avatar_url
FROM galleries g
JOIN users u ON g.user_id = u.id
LEFT JOIN art a ON g.cover_art_id = a.id
WHERE g.user_id = $1
ORDER BY g.created_at DESC;

-- name: ListAllGalleries :many
SELECT g.id, g.user_id, g.name, g.description, g.cover_art_id,
       COALESCE(a.image_url, '')::text AS cover_image_url,
       COALESCE(a.thumbnail_url, '')::text AS cover_thumbnail_url,
       (SELECT COUNT(*) FROM art ar WHERE ar.gallery_id = g.id) AS art_count,
       g.created_at, g.updated_at,
       u.username, u.display_name, u.avatar_url
FROM galleries g
JOIN users u ON g.user_id = u.id
LEFT JOIN art a ON g.cover_art_id = a.id
WHERE ($1::text = '' OR EXISTS(SELECT 1 FROM art ac WHERE ac.gallery_id = g.id AND ac.corner = $1::text))
ORDER BY g.created_at DESC;

-- name: GetGalleryPreviewImages :many
SELECT thumbnail_url, image_url
FROM art
WHERE gallery_id = sqlc.arg(gallery_id)::uuid
ORDER BY created_at DESC
LIMIT sqlc.arg(row_limit)::int;

-- name: CountArtFiltered :one
SELECT COUNT(*)
FROM art a
JOIN users u ON a.user_id = u.id
WHERE a.corner = sqlc.arg(corner)
  AND (sqlc.arg(art_type)::text = '' OR a.art_type = sqlc.arg(art_type)::text)
  AND (sqlc.arg(search)::text = ''
       OR a.title LIKE sqlc.arg(search)::text
       OR a.description LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM art_tags ta WHERE ta.art_id = a.id AND ta.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR a.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListArtByNew :many
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id) AS like_count,
       (SELECT COUNT(*) FROM art_comments ac WHERE ac.art_id = a.id) AS comment_count,
       a.view_count,
       EXISTS(SELECT 1 FROM art_likes ul WHERE ul.art_id = a.id AND ul.user_id = sqlc.arg(viewer_id)) AS user_liked,
       a.is_spoiler
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.corner = sqlc.arg(corner)
  AND (sqlc.arg(art_type)::text = '' OR a.art_type = sqlc.arg(art_type)::text)
  AND (sqlc.arg(search)::text = ''
       OR a.title LIKE sqlc.arg(search)::text
       OR a.description LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM art_tags ta WHERE ta.art_id = a.id AND ta.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR a.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY a.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListArtByPopular :many
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id) AS like_count,
       (SELECT COUNT(*) FROM art_comments ac WHERE ac.art_id = a.id) AS comment_count,
       a.view_count,
       EXISTS(SELECT 1 FROM art_likes ul WHERE ul.art_id = a.id AND ul.user_id = sqlc.arg(viewer_id)) AS user_liked,
       a.is_spoiler
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.corner = sqlc.arg(corner)
  AND (sqlc.arg(art_type)::text = '' OR a.art_type = sqlc.arg(art_type)::text)
  AND (sqlc.arg(search)::text = ''
       OR a.title LIKE sqlc.arg(search)::text
       OR a.description LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM art_tags ta WHERE ta.art_id = a.id AND ta.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR a.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM art_likes pl WHERE pl.art_id = a.id) DESC, a.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListArtByViews :many
SELECT a.id, a.user_id, a.corner, a.art_type, a.title, a.description, a.image_url, a.thumbnail_url,
       a.gallery_id, a.created_at, a.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM art_likes al WHERE al.art_id = a.id) AS like_count,
       (SELECT COUNT(*) FROM art_comments ac WHERE ac.art_id = a.id) AS comment_count,
       a.view_count,
       EXISTS(SELECT 1 FROM art_likes ul WHERE ul.art_id = a.id AND ul.user_id = sqlc.arg(viewer_id)) AS user_liked,
       a.is_spoiler
FROM art a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = a.user_id
WHERE a.corner = sqlc.arg(corner)
  AND (sqlc.arg(art_type)::text = '' OR a.art_type = sqlc.arg(art_type)::text)
  AND (sqlc.arg(search)::text = ''
       OR a.title LIKE sqlc.arg(search)::text
       OR a.description LIKE sqlc.arg(search)::text
       OR u.display_name LIKE sqlc.arg(search)::text
       OR u.username LIKE sqlc.arg(search)::text)
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM art_tags ta WHERE ta.art_id = a.id AND ta.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR a.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY a.view_count DESC, a.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;
