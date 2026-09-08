-- name: CreateShip :one
WITH s AS (
    INSERT INTO ships (user_id, title, description, image_url, thumbnail_url)
    VALUES ($1, $2, $3, '', '')
    RETURNING id, user_id, title, description, image_url, thumbnail_url, created_at, updated_at
)
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS vote_score,
       0::int AS user_vote,
       0::bigint AS comment_count
FROM s
JOIN users u ON u.id = s.user_id
LEFT JOIN user_roles r ON r.user_id = s.user_id;

-- name: GetShipByID :one
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE s.id = $2;

-- name: CountShipsByUser :one
SELECT COUNT(*) FROM ships WHERE user_id = $1;

-- name: ListShipsByUser :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE s.user_id = $2
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountShips :one
SELECT COUNT(*) FROM ships s
WHERE ($1::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $1::text))
  AND ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $2::text))
  AND (NOT $3::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $4::int)
  AND ($5::text = '' OR s.user_id <> ALL(string_to_array($5::text, ',')::uuid[]));

-- name: ListShipsByNew :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $2::text))
  AND ($3::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $3::text))
  AND (NOT $4::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $5::int)
  AND ($6::text = '' OR s.user_id <> ALL(string_to_array($6::text, ',')::uuid[]))
ORDER BY s.created_at DESC
LIMIT $7 OFFSET $8;

-- name: ListShipsByTop :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $2::text))
  AND ($3::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $3::text))
  AND (NOT $4::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $5::int)
  AND ($6::text = '' OR s.user_id <> ALL(string_to_array($6::text, ',')::uuid[]))
ORDER BY COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) DESC, s.created_at DESC
LIMIT $7 OFFSET $8;

-- name: ListShipsByCrackship :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $2::text))
  AND ($3::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $3::text))
  AND (NOT $4::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $5::int)
  AND ($6::text = '' OR s.user_id <> ALL(string_to_array($6::text, ',')::uuid[]))
ORDER BY COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) ASC, s.created_at DESC
LIMIT $7 OFFSET $8;

-- name: ListShipsByControversial :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $2::text))
  AND ($3::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $3::text))
  AND (NOT $4::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $5::int)
  AND ($6::text = '' OR s.user_id <> ALL(string_to_array($6::text, ',')::uuid[]))
ORDER BY (
    (SELECT COUNT(*) FROM ship_votes v WHERE v.ship_id = s.id AND v.value = 1) *
    (SELECT COUNT(*) FROM ship_votes v WHERE v.ship_id = s.id AND v.value = -1)
) DESC, s.created_at DESC
LIMIT $7 OFFSET $8;

-- name: ListShipsByComments :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $2::text))
  AND ($3::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $3::text))
  AND (NOT $4::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $5::int)
  AND ($6::text = '' OR s.user_id <> ALL(string_to_array($6::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id) DESC, s.created_at DESC
LIMIT $7 OFFSET $8;

-- name: ListShipsByOld :many
SELECT s.id, s.user_id, s.title, s.description, s.image_url, s.thumbnail_url, s.created_at, s.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0)::bigint AS vote_score,
       COALESCE((SELECT v.value FROM ship_votes v WHERE v.ship_id = s.id AND v.user_id = $1), 0)::int AS user_vote,
       (SELECT COUNT(*) FROM ship_comments c WHERE c.ship_id = s.id)::bigint AS comment_count
FROM ships s
JOIN users u ON s.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE ($2::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.series = $2::text))
  AND ($3::text = '' OR EXISTS(SELECT 1 FROM ship_characters sc WHERE sc.ship_id = s.id AND sc.character_id = $3::text))
  AND (NOT $4::boolean OR COALESCE((SELECT SUM(v.value) FROM ship_votes v WHERE v.ship_id = s.id), 0) <= $5::int)
  AND ($6::text = '' OR s.user_id <> ALL(string_to_array($6::text, ',')::uuid[]))
ORDER BY s.created_at ASC
LIMIT $7 OFFSET $8;

-- name: UpdateShipDetailsAsAdmin :execrows
UPDATE ships SET title = $1, description = $2, updated_at = NOW() WHERE id = $3;

-- name: UpdateShipDetails :execrows
UPDATE ships SET title = $1, description = $2, updated_at = NOW() WHERE id = $3 AND user_id = $4;

-- name: UpdateShipImage :exec
UPDATE ships SET image_url = $1, thumbnail_url = $2 WHERE id = $3;

-- name: GetShipImagePaths :one
SELECT image_url, thumbnail_url FROM ships WHERE id = $1;

-- name: InsertShipCharacter :exec
INSERT INTO ship_characters (ship_id, series, character_id, character_name, sort_order)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteShipCharacters :exec
DELETE FROM ship_characters WHERE ship_id = $1;

-- name: GetShipCharacters :many
SELECT id, ship_id, series, character_id, character_name, sort_order
FROM ship_characters
WHERE ship_id = $1
ORDER BY sort_order ASC;

-- name: GetShipCharactersBatch :many
SELECT id, ship_id, series, character_id, character_name, sort_order
FROM ship_characters
WHERE ship_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order ASC;
