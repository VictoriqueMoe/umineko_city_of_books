-- name: CreateFanfic :one
WITH f AS (
    INSERT INTO fanfics (user_id, title, summary, series, rating, language, status, is_oneshot, contains_lemons)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id, user_id, title, summary, series, rating, language, status,
              is_oneshot, contains_lemons, cover_image_url, cover_thumbnail_url,
              word_count, favourite_count, view_count, comment_count,
              published_at, created_at, updated_at
)
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS chapter_count,
       FALSE::boolean AS user_favourited,
       $10::boolean AS is_pairing
FROM f
JOIN users u ON u.id = f.user_id
LEFT JOIN user_roles r ON r.user_id = u.id;

-- name: GetFanficByID :one
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM fanfic_chapters ch WHERE ch.fanfic_id = f.id)::bigint AS chapter_count,
       (EXISTS(SELECT 1 FROM fanfic_favourites fv WHERE fv.fanfic_id = f.id AND fv.user_id = $1))::boolean AS user_favourited,
       (EXISTS(SELECT 1 FROM fanfic_characters fc WHERE fc.fanfic_id = f.id AND fc.is_pairing = TRUE))::boolean AS is_pairing
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE f.id = $2;

-- name: CountFanficsByUser :one
SELECT COUNT(*) FROM fanfics WHERE user_id = $1 AND (status != 'draft' OR user_id = $2);

-- name: ListFanficsByUser :many
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM fanfic_chapters ch WHERE ch.fanfic_id = f.id)::bigint AS chapter_count,
       (EXISTS(SELECT 1 FROM fanfic_favourites fv WHERE fv.fanfic_id = f.id AND fv.user_id = $1))::boolean AS user_favourited,
       (EXISTS(SELECT 1 FROM fanfic_characters fc WHERE fc.fanfic_id = f.id AND fc.is_pairing = TRUE))::boolean AS is_pairing
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE f.user_id = $2 AND (f.status != 'draft' OR f.user_id = $1)
ORDER BY f.updated_at DESC
LIMIT $3 OFFSET $4;

-- name: CountFanfics :one
SELECT COUNT(*)
FROM fanfics f
WHERE (f.status != 'draft' OR f.user_id = sqlc.arg(viewer_id))
  AND (sqlc.arg(show_lemons)::boolean OR f.contains_lemons = FALSE)
  AND (sqlc.arg(series)::citext = '' OR f.series = sqlc.arg(series)::citext)
  AND (sqlc.arg(rating)::text = '' OR f.rating = sqlc.arg(rating)::text)
  AND (sqlc.arg(language)::citext = '' OR f.language = sqlc.arg(language)::citext)
  AND (sqlc.arg(status)::text = '' OR f.status = sqlc.arg(status)::text)
  AND (sqlc.arg(genre_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres ga WHERE ga.fanfic_id = f.id AND ga.genre = sqlc.arg(genre_a)::text))
  AND (sqlc.arg(genre_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres gb WHERE gb.fanfic_id = f.id AND gb.genre = sqlc.arg(genre_b)::text))
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM fanfic_tags ft WHERE ft.fanfic_id = f.id AND ft.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(character_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters ca WHERE ca.fanfic_id = f.id AND ca.character_name = sqlc.arg(character_a)::text AND (NOT sqlc.arg(is_pairing)::boolean OR ca.is_pairing = TRUE)))
  AND (sqlc.arg(character_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cb WHERE cb.fanfic_id = f.id AND cb.character_name = sqlc.arg(character_b)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cb.is_pairing = TRUE)))
  AND (sqlc.arg(character_c)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cc WHERE cc.fanfic_id = f.id AND cc.character_name = sqlc.arg(character_c)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cc.is_pairing = TRUE)))
  AND (sqlc.arg(character_d)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cd WHERE cd.fanfic_id = f.id AND cd.character_name = sqlc.arg(character_d)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cd.is_pairing = TRUE)))
  AND (sqlc.arg(search)::text = '' OR f.title ILIKE sqlc.arg(search)::text OR f.summary ILIKE sqlc.arg(search)::text)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR f.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListFanficsByUpdated :many
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM fanfic_chapters ch WHERE ch.fanfic_id = f.id)::bigint AS chapter_count,
       (EXISTS(SELECT 1 FROM fanfic_favourites fv WHERE fv.fanfic_id = f.id AND fv.user_id = sqlc.arg(viewer_id)))::boolean AS user_favourited,
       (EXISTS(SELECT 1 FROM fanfic_characters fc WHERE fc.fanfic_id = f.id AND fc.is_pairing = TRUE))::boolean AS is_pairing
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE (f.status != 'draft' OR f.user_id = sqlc.arg(viewer_id))
  AND (sqlc.arg(show_lemons)::boolean OR f.contains_lemons = FALSE)
  AND (sqlc.arg(series)::citext = '' OR f.series = sqlc.arg(series)::citext)
  AND (sqlc.arg(rating)::text = '' OR f.rating = sqlc.arg(rating)::text)
  AND (sqlc.arg(language)::citext = '' OR f.language = sqlc.arg(language)::citext)
  AND (sqlc.arg(status)::text = '' OR f.status = sqlc.arg(status)::text)
  AND (sqlc.arg(genre_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres ga WHERE ga.fanfic_id = f.id AND ga.genre = sqlc.arg(genre_a)::text))
  AND (sqlc.arg(genre_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres gb WHERE gb.fanfic_id = f.id AND gb.genre = sqlc.arg(genre_b)::text))
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM fanfic_tags ft WHERE ft.fanfic_id = f.id AND ft.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(character_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters ca WHERE ca.fanfic_id = f.id AND ca.character_name = sqlc.arg(character_a)::text AND (NOT sqlc.arg(is_pairing)::boolean OR ca.is_pairing = TRUE)))
  AND (sqlc.arg(character_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cb WHERE cb.fanfic_id = f.id AND cb.character_name = sqlc.arg(character_b)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cb.is_pairing = TRUE)))
  AND (sqlc.arg(character_c)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cc WHERE cc.fanfic_id = f.id AND cc.character_name = sqlc.arg(character_c)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cc.is_pairing = TRUE)))
  AND (sqlc.arg(character_d)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cd WHERE cd.fanfic_id = f.id AND cd.character_name = sqlc.arg(character_d)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cd.is_pairing = TRUE)))
  AND (sqlc.arg(search)::text = '' OR f.title ILIKE sqlc.arg(search)::text OR f.summary ILIKE sqlc.arg(search)::text)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR f.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY f.updated_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListFanficsByPublished :many
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM fanfic_chapters ch WHERE ch.fanfic_id = f.id)::bigint AS chapter_count,
       (EXISTS(SELECT 1 FROM fanfic_favourites fv WHERE fv.fanfic_id = f.id AND fv.user_id = sqlc.arg(viewer_id)))::boolean AS user_favourited,
       (EXISTS(SELECT 1 FROM fanfic_characters fc WHERE fc.fanfic_id = f.id AND fc.is_pairing = TRUE))::boolean AS is_pairing
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE (f.status != 'draft' OR f.user_id = sqlc.arg(viewer_id))
  AND (sqlc.arg(show_lemons)::boolean OR f.contains_lemons = FALSE)
  AND (sqlc.arg(series)::citext = '' OR f.series = sqlc.arg(series)::citext)
  AND (sqlc.arg(rating)::text = '' OR f.rating = sqlc.arg(rating)::text)
  AND (sqlc.arg(language)::citext = '' OR f.language = sqlc.arg(language)::citext)
  AND (sqlc.arg(status)::text = '' OR f.status = sqlc.arg(status)::text)
  AND (sqlc.arg(genre_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres ga WHERE ga.fanfic_id = f.id AND ga.genre = sqlc.arg(genre_a)::text))
  AND (sqlc.arg(genre_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres gb WHERE gb.fanfic_id = f.id AND gb.genre = sqlc.arg(genre_b)::text))
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM fanfic_tags ft WHERE ft.fanfic_id = f.id AND ft.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(character_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters ca WHERE ca.fanfic_id = f.id AND ca.character_name = sqlc.arg(character_a)::text AND (NOT sqlc.arg(is_pairing)::boolean OR ca.is_pairing = TRUE)))
  AND (sqlc.arg(character_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cb WHERE cb.fanfic_id = f.id AND cb.character_name = sqlc.arg(character_b)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cb.is_pairing = TRUE)))
  AND (sqlc.arg(character_c)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cc WHERE cc.fanfic_id = f.id AND cc.character_name = sqlc.arg(character_c)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cc.is_pairing = TRUE)))
  AND (sqlc.arg(character_d)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cd WHERE cd.fanfic_id = f.id AND cd.character_name = sqlc.arg(character_d)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cd.is_pairing = TRUE)))
  AND (sqlc.arg(search)::text = '' OR f.title ILIKE sqlc.arg(search)::text OR f.summary ILIKE sqlc.arg(search)::text)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR f.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY f.published_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListFanficsByFavourites :many
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM fanfic_chapters ch WHERE ch.fanfic_id = f.id)::bigint AS chapter_count,
       (EXISTS(SELECT 1 FROM fanfic_favourites fv WHERE fv.fanfic_id = f.id AND fv.user_id = sqlc.arg(viewer_id)))::boolean AS user_favourited,
       (EXISTS(SELECT 1 FROM fanfic_characters fc WHERE fc.fanfic_id = f.id AND fc.is_pairing = TRUE))::boolean AS is_pairing
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE (f.status != 'draft' OR f.user_id = sqlc.arg(viewer_id))
  AND (sqlc.arg(show_lemons)::boolean OR f.contains_lemons = FALSE)
  AND (sqlc.arg(series)::citext = '' OR f.series = sqlc.arg(series)::citext)
  AND (sqlc.arg(rating)::text = '' OR f.rating = sqlc.arg(rating)::text)
  AND (sqlc.arg(language)::citext = '' OR f.language = sqlc.arg(language)::citext)
  AND (sqlc.arg(status)::text = '' OR f.status = sqlc.arg(status)::text)
  AND (sqlc.arg(genre_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres ga WHERE ga.fanfic_id = f.id AND ga.genre = sqlc.arg(genre_a)::text))
  AND (sqlc.arg(genre_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_genres gb WHERE gb.fanfic_id = f.id AND gb.genre = sqlc.arg(genre_b)::text))
  AND (sqlc.arg(tag)::text = '' OR EXISTS(SELECT 1 FROM fanfic_tags ft WHERE ft.fanfic_id = f.id AND ft.tag = sqlc.arg(tag)::text))
  AND (sqlc.arg(character_a)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters ca WHERE ca.fanfic_id = f.id AND ca.character_name = sqlc.arg(character_a)::text AND (NOT sqlc.arg(is_pairing)::boolean OR ca.is_pairing = TRUE)))
  AND (sqlc.arg(character_b)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cb WHERE cb.fanfic_id = f.id AND cb.character_name = sqlc.arg(character_b)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cb.is_pairing = TRUE)))
  AND (sqlc.arg(character_c)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cc WHERE cc.fanfic_id = f.id AND cc.character_name = sqlc.arg(character_c)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cc.is_pairing = TRUE)))
  AND (sqlc.arg(character_d)::text = '' OR EXISTS(SELECT 1 FROM fanfic_characters cd WHERE cd.fanfic_id = f.id AND cd.character_name = sqlc.arg(character_d)::text AND (NOT sqlc.arg(is_pairing)::boolean OR cd.is_pairing = TRUE)))
  AND (sqlc.arg(search)::text = '' OR f.title ILIKE sqlc.arg(search)::text OR f.summary ILIKE sqlc.arg(search)::text)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR f.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY f.favourite_count DESC, f.updated_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: CountFanficFavourites :one
SELECT COUNT(*) FROM fanfic_favourites WHERE user_id = $1;

-- name: ListFanficFavourites :many
SELECT f.id, f.user_id, f.title, f.summary, f.series, f.rating, f.language, f.status,
       f.is_oneshot, f.contains_lemons, f.cover_image_url, f.cover_thumbnail_url,
       f.word_count, f.favourite_count, f.view_count, f.comment_count,
       f.published_at, f.created_at, f.updated_at,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM fanfic_chapters ch WHERE ch.fanfic_id = f.id)::bigint AS chapter_count,
       (EXISTS(SELECT 1 FROM fanfic_favourites fv WHERE fv.fanfic_id = f.id AND fv.user_id = $1))::boolean AS user_favourited,
       (EXISTS(SELECT 1 FROM fanfic_characters fc WHERE fc.fanfic_id = f.id AND fc.is_pairing = TRUE))::boolean AS is_pairing
FROM fanfics f
JOIN users u ON f.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
JOIN fanfic_favourites fav ON fav.fanfic_id = f.id
WHERE fav.user_id = $2 AND (f.status != 'draft' OR f.user_id = $1)
ORDER BY fav.created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateFanficAsAdmin :execrows
UPDATE fanfics
SET title = $1, summary = $2, series = $3, rating = $4, language = $5, status = $6, is_oneshot = $7, contains_lemons = $8, updated_at = NOW()
WHERE id = $9;

-- name: UpdateFanficAsOwner :execrows
UPDATE fanfics
SET title = $1, summary = $2, series = $3, rating = $4, language = $5, status = $6, is_oneshot = $7, contains_lemons = $8, updated_at = NOW()
WHERE id = $9 AND user_id = $10;

-- name: UpdateFanficCoverImage :exec
UPDATE fanfics SET cover_image_url = $1, cover_thumbnail_url = $2 WHERE id = $3;

-- name: UpdateFanficWordCount :exec
UPDATE fanfics
SET word_count = COALESCE((SELECT SUM(ch.word_count) FROM fanfic_chapters ch WHERE ch.fanfic_id = $1), 0), updated_at = NOW()
WHERE id = $1;

-- name: GetFanficCoverPaths :one
SELECT cover_image_url, cover_thumbnail_url FROM fanfics WHERE id = $1;

-- name: CreateFanficChapter :one
INSERT INTO fanfic_chapters (fanfic_id, chapter_number, title, body, word_count)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, fanfic_id, chapter_number, title, body, word_count, created_at, updated_at;

-- name: UpdateFanficChapter :exec
UPDATE fanfic_chapters SET title = $1, body = $2, word_count = $3, updated_at = NOW() WHERE id = $4;

-- name: DeleteFanficChapter :exec
DELETE FROM fanfic_chapters WHERE id = $1;

-- name: GetFanficChapter :one
SELECT id, fanfic_id, chapter_number, title, body, word_count, created_at, updated_at
FROM fanfic_chapters
WHERE fanfic_id = $1 AND chapter_number = $2;

-- name: ListFanficChapters :many
SELECT id, chapter_number, title, word_count
FROM fanfic_chapters
WHERE fanfic_id = $1
ORDER BY chapter_number ASC;

-- name: CountFanficChapters :one
SELECT COUNT(*) FROM fanfic_chapters WHERE fanfic_id = $1;

-- name: GetNextFanficChapterNumber :one
SELECT (COALESCE(MAX(chapter_number), 0) + 1)::int AS next_number FROM fanfic_chapters WHERE fanfic_id = $1;

-- name: GetFanficChapterFanficID :one
SELECT fanfic_id FROM fanfic_chapters WHERE id = $1;

-- name: GetFanficChapterAuthorID :one
SELECT f.user_id FROM fanfic_chapters c JOIN fanfics f ON c.fanfic_id = f.id WHERE c.id = $1;

-- name: AddFanficGenre :exec
INSERT INTO fanfic_genres (fanfic_id, genre) VALUES ($1, $2);

-- name: DeleteFanficGenres :exec
DELETE FROM fanfic_genres WHERE fanfic_id = $1;

-- name: GetFanficGenres :many
SELECT genre FROM fanfic_genres WHERE fanfic_id = $1 ORDER BY genre ASC;

-- name: GetFanficGenresBatch :many
SELECT fanfic_id, genre
FROM fanfic_genres
WHERE fanfic_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY genre ASC;

-- name: AddFanficTag :exec
INSERT INTO fanfic_tags (fanfic_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteFanficTags :exec
DELETE FROM fanfic_tags WHERE fanfic_id = $1;

-- name: GetFanficTags :many
SELECT tag FROM fanfic_tags WHERE fanfic_id = $1 ORDER BY tag ASC;

-- name: GetFanficTagsBatch :many
SELECT fanfic_id, tag
FROM fanfic_tags
WHERE fanfic_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY tag ASC;

-- name: AddFanficCharacter :exec
INSERT INTO fanfic_characters (fanfic_id, series, character_id, character_name, sort_order, is_pairing)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteFanficCharacters :exec
DELETE FROM fanfic_characters WHERE fanfic_id = $1;

-- name: GetFanficCharacters :many
SELECT id, fanfic_id, series, character_id, character_name, sort_order, is_pairing
FROM fanfic_characters
WHERE fanfic_id = $1
ORDER BY sort_order ASC;

-- name: GetFanficCharactersBatch :many
SELECT id, fanfic_id, series, character_id, character_name, sort_order, is_pairing
FROM fanfic_characters
WHERE fanfic_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order ASC;

-- name: RegisterFanficOCCharacter :exec
INSERT INTO fanfic_oc_characters (name, created_by) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: SearchFanficOCCharacters :many
SELECT name FROM fanfic_oc_characters WHERE name LIKE $1 ORDER BY name ASC;

-- name: GetFanficLanguages :many
SELECT name FROM fanfic_languages ORDER BY name ASC;

-- name: RegisterFanficLanguage :exec
INSERT INTO fanfic_languages (name) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: GetFanficSeries :many
SELECT name FROM fanfic_series ORDER BY name ASC;

-- name: RegisterFanficSeries :exec
INSERT INTO fanfic_series (name) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: FavouriteFanfic :exec
INSERT INTO fanfic_favourites (user_id, fanfic_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnfavouriteFanfic :exec
DELETE FROM fanfic_favourites WHERE user_id = $1 AND fanfic_id = $2;

-- name: SyncFanficFavouriteCount :exec
UPDATE fanfics
SET favourite_count = (SELECT COUNT(*) FROM fanfic_favourites ff WHERE ff.fanfic_id = $1)
WHERE id = $1;

-- name: GetFanficReadingProgress :one
SELECT chapter_number FROM fanfic_reading_progress WHERE user_id = $1 AND fanfic_id = $2;

-- name: SetFanficReadingProgress :exec
INSERT INTO fanfic_reading_progress (user_id, fanfic_id, chapter_number, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (user_id, fanfic_id) DO UPDATE SET chapter_number = $3, updated_at = NOW();
