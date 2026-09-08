-- name: CreateJournal :one
WITH j AS (
    INSERT INTO journals (user_id, title, work)
    VALUES ($1, $2, $3)
    RETURNING id, user_id, title, work, created_at, updated_at, last_author_activity_at, archived_at, paused_at
)
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS follower_count,
       0::bigint AS comment_count,
       0::bigint AS entry_count,
       NULL::int AS latest_entry_number,
       NULL::text AS latest_entry_title,
       NULL::text AS latest_entry_body,
       NULL::timestamptz AS latest_entry_at
FROM j
JOIN users u ON u.id = j.user_id
LEFT JOIN user_roles r ON r.user_id = u.id;

-- name: GetJournalByID :one
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id) AS follower_count,
       (SELECT COUNT(*) FROM journal_comments jc WHERE jc.journal_id = j.id) AS comment_count,
       (SELECT COUNT(*) FROM journal_entries je WHERE je.journal_id = j.id) AS entry_count,
       le.entry_number AS latest_entry_number,
       le.title AS latest_entry_title,
       le.body AS latest_entry_body,
       le.created_at AS latest_entry_at
FROM journals j
JOIN users u ON j.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN journal_entries le ON le.journal_id = j.id AND NOT le.is_draft
    AND le.entry_number = (SELECT MAX(e.entry_number) FROM journal_entries e WHERE e.journal_id = j.id AND NOT e.is_draft)
WHERE j.id = $1;

-- name: CountJournals :one
SELECT COUNT(*) FROM journals j
WHERE (sqlc.arg(work)::text = '' OR j.work = sqlc.arg(work)::text)
  AND (sqlc.narg(author_id)::uuid IS NULL OR j.user_id = sqlc.narg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR j.title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(include_archived)::boolean OR j.archived_at IS NULL)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR j.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListJournalsByNew :many
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id) AS follower_count,
       (SELECT COUNT(*) FROM journal_comments jc WHERE jc.journal_id = j.id) AS comment_count,
       (SELECT COUNT(*) FROM journal_entries je WHERE je.journal_id = j.id) AS entry_count,
       le.entry_number AS latest_entry_number,
       le.title AS latest_entry_title,
       le.body AS latest_entry_body,
       le.created_at AS latest_entry_at
FROM journals j
JOIN users u ON j.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN journal_entries le ON le.journal_id = j.id AND NOT le.is_draft
    AND le.entry_number = (SELECT MAX(e.entry_number) FROM journal_entries e WHERE e.journal_id = j.id AND NOT e.is_draft)
WHERE (sqlc.arg(work)::text = '' OR j.work = sqlc.arg(work)::text)
  AND (sqlc.narg(author_id)::uuid IS NULL OR j.user_id = sqlc.narg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR j.title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(include_archived)::boolean OR j.archived_at IS NULL)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR j.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY j.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListJournalsByOld :many
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id) AS follower_count,
       (SELECT COUNT(*) FROM journal_comments jc WHERE jc.journal_id = j.id) AS comment_count,
       (SELECT COUNT(*) FROM journal_entries je WHERE je.journal_id = j.id) AS entry_count,
       le.entry_number AS latest_entry_number,
       le.title AS latest_entry_title,
       le.body AS latest_entry_body,
       le.created_at AS latest_entry_at
FROM journals j
JOIN users u ON j.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN journal_entries le ON le.journal_id = j.id AND NOT le.is_draft
    AND le.entry_number = (SELECT MAX(e.entry_number) FROM journal_entries e WHERE e.journal_id = j.id AND NOT e.is_draft)
WHERE (sqlc.arg(work)::text = '' OR j.work = sqlc.arg(work)::text)
  AND (sqlc.narg(author_id)::uuid IS NULL OR j.user_id = sqlc.narg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR j.title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(include_archived)::boolean OR j.archived_at IS NULL)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR j.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY j.created_at ASC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListJournalsByRecentlyActive :many
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id) AS follower_count,
       (SELECT COUNT(*) FROM journal_comments jc WHERE jc.journal_id = j.id) AS comment_count,
       (SELECT COUNT(*) FROM journal_entries je WHERE je.journal_id = j.id) AS entry_count,
       le.entry_number AS latest_entry_number,
       le.title AS latest_entry_title,
       le.body AS latest_entry_body,
       le.created_at AS latest_entry_at
FROM journals j
JOIN users u ON j.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN journal_entries le ON le.journal_id = j.id AND NOT le.is_draft
    AND le.entry_number = (SELECT MAX(e.entry_number) FROM journal_entries e WHERE e.journal_id = j.id AND NOT e.is_draft)
WHERE (sqlc.arg(work)::text = '' OR j.work = sqlc.arg(work)::text)
  AND (sqlc.narg(author_id)::uuid IS NULL OR j.user_id = sqlc.narg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR j.title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(include_archived)::boolean OR j.archived_at IS NULL)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR j.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY j.last_author_activity_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListJournalsByMostFollowed :many
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id) AS follower_count,
       (SELECT COUNT(*) FROM journal_comments jc WHERE jc.journal_id = j.id) AS comment_count,
       (SELECT COUNT(*) FROM journal_entries je WHERE je.journal_id = j.id) AS entry_count,
       le.entry_number AS latest_entry_number,
       le.title AS latest_entry_title,
       le.body AS latest_entry_body,
       le.created_at AS latest_entry_at
FROM journals j
JOIN users u ON j.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN journal_entries le ON le.journal_id = j.id AND NOT le.is_draft
    AND le.entry_number = (SELECT MAX(e.entry_number) FROM journal_entries e WHERE e.journal_id = j.id AND NOT e.is_draft)
WHERE (sqlc.arg(work)::text = '' OR j.work = sqlc.arg(work)::text)
  AND (sqlc.narg(author_id)::uuid IS NULL OR j.user_id = sqlc.narg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR j.title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(include_archived)::boolean OR j.archived_at IS NULL)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR j.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM journal_follows fo WHERE fo.journal_id = j.id) DESC, j.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: CountFollowedJournals :one
SELECT COUNT(*) FROM journal_follows WHERE user_id = $1;

-- name: ListFollowedJournals :many
SELECT j.id, j.title, j.work, j.created_at, j.updated_at, j.last_author_activity_at, j.archived_at, j.paused_at,
       u.id AS author_id, u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_follows f WHERE f.journal_id = j.id) AS follower_count,
       (SELECT COUNT(*) FROM journal_comments jc WHERE jc.journal_id = j.id) AS comment_count,
       (SELECT COUNT(*) FROM journal_entries je WHERE je.journal_id = j.id) AS entry_count,
       le.entry_number AS latest_entry_number,
       le.title AS latest_entry_title,
       le.body AS latest_entry_body,
       le.created_at AS latest_entry_at
FROM journals j
JOIN users u ON j.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN journal_entries le ON le.journal_id = j.id AND NOT le.is_draft
    AND le.entry_number = (SELECT MAX(e.entry_number) FROM journal_entries e WHERE e.journal_id = j.id AND NOT e.is_draft)
JOIN journal_follows jf ON jf.journal_id = j.id
WHERE jf.user_id = $1
ORDER BY jf.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateJournal :execrows
UPDATE journals
SET title = $1, work = $2, updated_at = NOW(), last_author_activity_at = NOW(), archived_at = NULL
WHERE id = $3 AND user_id = $4;

-- name: UpdateJournalAsAdmin :exec
UPDATE journals SET title = $1, work = $2, updated_at = NOW() WHERE id = $3;

-- name: GetJournalTitle :one
SELECT title FROM journals WHERE id = $1;

-- name: GetJournalArchivedAt :one
SELECT archived_at FROM journals WHERE id = $1;

-- name: CountUserJournalsToday :one
SELECT COUNT(*) FROM journals WHERE user_id = $1 AND created_at >= NOW() - INTERVAL '1 day';

-- name: UpdateJournalLastAuthorActivity :exec
UPDATE journals SET last_author_activity_at = NOW(), archived_at = NULL WHERE id = $1;

-- name: ArchiveStaleJournals :many
UPDATE journals SET archived_at = NOW()
WHERE archived_at IS NULL AND paused_at IS NULL AND last_author_activity_at < $1
RETURNING id;

-- name: SetJournalPaused :execrows
UPDATE journals SET paused_at = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3;

-- name: FollowJournal :exec
INSERT INTO journal_follows (user_id, journal_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnfollowJournal :exec
DELETE FROM journal_follows WHERE user_id = $1 AND journal_id = $2;

-- name: IsJournalFollower :one
SELECT EXISTS(SELECT 1 FROM journal_follows f WHERE f.user_id = $1 AND f.journal_id = $2) AS is_follower;

-- name: IsJournalFollowedByViewer :one
SELECT EXISTS(SELECT 1 FROM journal_follows f WHERE f.journal_id = $1 AND f.user_id = $2) AS is_following;

-- name: GetJournalFollowerIDs :many
SELECT user_id FROM journal_follows WHERE journal_id = $1;

-- name: GetJournalFollowerCount :one
SELECT COUNT(*) FROM journal_follows WHERE journal_id = $1;

-- name: ListJournalEntryIDs :many
SELECT id FROM journal_entries WHERE journal_id = $1 ORDER BY entry_number;

-- name: ListJournalEntryCommentIDs :many
WITH RECURSIVE tree AS (
    SELECT root.id FROM journal_comments root WHERE root.entry_id = sqlc.arg(entry_id)::uuid
    UNION
    SELECT child.id FROM journal_comments child JOIN tree t ON child.parent_id = t.id
)
SELECT t.id FROM tree t;

-- name: CreateJournalEntry :one
INSERT INTO journal_entries (journal_id, entry_number, title, body, word_count, is_draft)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, journal_id, entry_number, title, body, word_count, is_draft, created_at, updated_at;

-- name: UpdateJournalEntry :execrows
UPDATE journal_entries
SET title = $1, body = $2, word_count = $3, is_draft = $4, updated_at = NOW()
WHERE id = $5;

-- name: DeleteJournalEntry :execrows
DELETE FROM journal_entries WHERE id = $1;

-- name: GetJournalEntry :one
SELECT e.id, e.journal_id, e.entry_number, e.title, e.body, e.word_count, e.is_draft, e.created_at, e.updated_at,
       EXISTS(SELECT 1 FROM journal_entries p WHERE p.journal_id = $1 AND p.entry_number < $2 AND NOT p.is_draft) AS has_prev,
       EXISTS(SELECT 1 FROM journal_entries n WHERE n.journal_id = $1 AND n.entry_number > $2 AND NOT n.is_draft) AS has_next
FROM journal_entries e
WHERE e.journal_id = $1 AND e.entry_number = $2;

-- name: GetJournalEntryByID :one
SELECT id, journal_id, entry_number, title, body, word_count, is_draft, created_at, updated_at
FROM journal_entries
WHERE id = $1;

-- name: ListJournalEntries :many
SELECT id, entry_number, title, word_count, is_draft, created_at
FROM journal_entries
WHERE journal_id = $1
ORDER BY entry_number DESC;

-- name: GetNextJournalEntryNumber :one
SELECT (COALESCE(MAX(entry_number), 0) + 1)::int AS next_entry_number
FROM journal_entries
WHERE journal_id = $1;

-- name: GetJournalEntryJournalID :one
SELECT journal_id FROM journal_entries WHERE id = $1;

-- name: GetJournalEntryAuthorID :one
SELECT j.user_id FROM journal_entries e JOIN journals j ON j.id = e.journal_id WHERE e.id = $1;

-- name: CreateJournalCommentWithEntry :one
WITH c AS (
    INSERT INTO journal_comments (journal_id, entry_id, parent_id, user_id, body)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, journal_id, entry_id, parent_id, user_id, body, created_at, updated_at
)
SELECT c.id, c.journal_id::text AS entity_id, c.entry_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (u.banned_at IS NOT NULL)::boolean AS author_banned,
       0::bigint AS like_count,
       FALSE::boolean AS user_liked
FROM c
JOIN users u ON u.id = c.user_id
LEFT JOIN user_roles r ON r.user_id = c.user_id;

-- name: CountJournalRootComments :one
SELECT COUNT(*) FROM journal_comments
WHERE journal_id = $1 AND entry_id IS NULL
  AND ($2::text = '' OR user_id <> ALL(string_to_array($2::text, ',')::uuid[]));

-- name: ListJournalRootComments :many
SELECT c.id, c.journal_id::text AS entity_id, c.entry_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_comment_likes lc WHERE lc.comment_id = c.id) AS like_count,
       EXISTS(SELECT 1 FROM journal_comment_likes lu WHERE lu.comment_id = c.id AND lu.user_id = $1) AS user_liked
FROM journal_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.journal_id = $2 AND c.entry_id IS NULL
  AND ($3::text = '' OR c.user_id <> ALL(string_to_array($3::text, ',')::uuid[]))
ORDER BY c.created_at ASC
LIMIT $4 OFFSET $5;

-- name: CountJournalEntryComments :one
SELECT COUNT(*) FROM journal_comments
WHERE entry_id = $1::uuid
  AND ($2::text = '' OR user_id <> ALL(string_to_array($2::text, ',')::uuid[]));

-- name: ListJournalEntryComments :many
SELECT c.id, c.journal_id::text AS entity_id, c.entry_id, c.parent_id, c.user_id, c.body, c.created_at, c.updated_at,
       u.username, u.display_name, u.avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       (SELECT COUNT(*) FROM journal_comment_likes lc WHERE lc.comment_id = c.id) AS like_count,
       EXISTS(SELECT 1 FROM journal_comment_likes lu WHERE lu.comment_id = c.id AND lu.user_id = $1) AS user_liked
FROM journal_comments c
JOIN users u ON c.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = c.user_id
WHERE c.entry_id = $2::uuid
  AND ($3::text = '' OR c.user_id <> ALL(string_to_array($3::text, ',')::uuid[]))
ORDER BY c.created_at ASC
LIMIT $4 OFFSET $5;

-- name: GetJournalCommentEntryNumber :one
SELECT e.entry_number
FROM journal_comments c
LEFT JOIN journal_entries e ON e.id = c.entry_id
WHERE c.id = $1;
