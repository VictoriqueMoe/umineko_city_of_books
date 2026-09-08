-- name: CreateChatBannedWord :one
WITH ins AS (
    INSERT INTO chat_banned_words (scope, room_id, pattern, match_mode, case_sensitive, action, created_by)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, scope, room_id, pattern, match_mode, case_sensitive, action, created_by, created_at
)
SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
       w.created_by, COALESCE(u.display_name, u.username, '')::text AS created_by_name, w.created_at
FROM ins w
LEFT JOIN users u ON w.created_by = u.id;

-- name: UpdateChatBannedWord :execrows
UPDATE chat_banned_words SET pattern = $1, match_mode = $2, case_sensitive = $3, action = $4 WHERE id = $5;

-- name: DeleteChatBannedWord :exec
DELETE FROM chat_banned_words WHERE id = $1;

-- name: GetChatBannedWordByID :one
SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
       w.created_by, COALESCE(u.display_name, u.username, '')::text AS created_by_name, w.created_at
FROM chat_banned_words w
LEFT JOIN users u ON w.created_by = u.id
WHERE w.id = $1;

-- name: ListGlobalChatBannedWords :many
SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
       w.created_by, COALESCE(u.display_name, u.username, '')::text AS created_by_name, w.created_at
FROM chat_banned_words w
LEFT JOIN users u ON w.created_by = u.id
WHERE w.scope = 'global'
ORDER BY w.created_at DESC;

-- name: ListChatBannedWordsForRoom :many
SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
       w.created_by, COALESCE(u.display_name, u.username, '')::text AS created_by_name, w.created_at
FROM chat_banned_words w
LEFT JOIN users u ON w.created_by = u.id
WHERE w.scope = 'room' AND w.room_id = $1
ORDER BY w.created_at DESC;

-- name: ListApplicableChatBannedWords :many
SELECT w.id, w.scope, w.room_id, w.pattern, w.match_mode, w.case_sensitive, w.action,
       w.created_by, COALESCE(u.display_name, u.username, '')::text AS created_by_name, w.created_at
FROM chat_banned_words w
LEFT JOIN users u ON w.created_by = u.id
WHERE w.scope = 'global' OR (w.scope = 'room' AND w.room_id = $1);
