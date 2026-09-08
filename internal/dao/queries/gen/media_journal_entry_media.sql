-- name: AddJournalEntryMedia :one
INSERT INTO journal_entry_media (entry_id, media_url, media_type, thumbnail_url, filename, sort_order, is_spoiler)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(m.sort_order) + 1 FROM journal_entry_media m WHERE m.entry_id = $1), $6::int), $7)
RETURNING id;

-- name: GetJournalEntryMediaURL :one
SELECT media_url FROM journal_entry_media WHERE id = $1 AND entry_id = $2;

-- name: DeleteJournalEntryMediaRow :exec
DELETE FROM journal_entry_media WHERE id = $1 AND entry_id = $2;

-- name: UpdateJournalEntryMediaURL :exec
UPDATE journal_entry_media SET media_url = $1 WHERE id = $2;

-- name: UpdateJournalEntryMediaThumbnail :exec
UPDATE journal_entry_media SET thumbnail_url = $1 WHERE id = $2;

-- name: GetJournalEntryMedia :many
SELECT id, entry_id AS target_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM journal_entry_media
WHERE entry_id = $1
ORDER BY sort_order, id;

-- name: GetJournalEntryMediaBatch :many
SELECT id, entry_id AS target_id, media_url, media_type, thumbnail_url, COALESCE(filename, '')::text AS filename, sort_order, is_spoiler
FROM journal_entry_media
WHERE entry_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY sort_order, id;

-- name: CollectJournalEntryMediaPaths :many
SELECT media_url, thumbnail_url FROM journal_entry_media WHERE entry_id = $1;
