-- name: CreateAuditLogEntry :exec
INSERT INTO audit_log (actor_id, action, target_type, target_id, details, subject_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CountAuditLogForUser :one
SELECT COUNT(*)
FROM audit_log a
WHERE ((a.target_type = 'user' AND a.target_id = $1::text) OR a.subject_id = $1::text::uuid);

-- name: ListAuditLogForUser :many
SELECT a.id, a.actor_id,
       COALESCE(u.display_name, '')::text AS actor_name,
       a.action, a.target_type, a.target_id, a.details, a.created_at, a.subject_id,
       COALESCE(s.display_name, '')::text AS subject_name,
       COALESCE(s.username, '')::text AS subject_username
FROM audit_log a
LEFT JOIN users u ON a.actor_id = u.id
LEFT JOIN users s ON a.subject_id = s.id
WHERE ((a.target_type = 'user' AND a.target_id = $1::text) OR a.subject_id = $1::text::uuid)
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditLog :one
SELECT COUNT(*)
FROM audit_log a
WHERE ($1::text = '' OR a.action = $1::text);

-- name: ListAuditLog :many
SELECT a.id, a.actor_id,
       COALESCE(u.display_name, '')::text AS actor_name,
       a.action, a.target_type, a.target_id, a.details, a.created_at, a.subject_id,
       COALESCE(s.display_name, '')::text AS subject_name,
       COALESCE(s.username, '')::text AS subject_username
FROM audit_log a
LEFT JOIN users u ON a.actor_id = u.id
LEFT JOIN users s ON a.subject_id = s.id
WHERE ($1::text = '' OR a.action = $1::text)
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;
