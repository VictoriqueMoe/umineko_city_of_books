-- name: CreateReport :one
WITH rep AS (
    INSERT INTO reports (reporter_id, target_type, target_id, context_id, reason)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, reporter_id, target_type, target_id, context_id, reason, status, resolved_by, created_at
)
SELECT rep.id, rep.reporter_id, u.display_name, u.avatar_url,
       rep.target_type, rep.target_id, COALESCE(rep.context_id, '')::text AS context_id, rep.reason, rep.status,
       rep.resolved_by, ''::text AS resolved_by_name, rep.created_at
FROM rep
JOIN users u ON rep.reporter_id = u.id;

-- name: CountReports :one
SELECT COUNT(*) FROM reports r;

-- name: CountReportsByStatus :one
SELECT COUNT(*) FROM reports r WHERE r.status = $1;

-- name: ListReports :many
SELECT r.id, r.reporter_id, u.display_name, u.avatar_url,
       r.target_type, r.target_id, COALESCE(r.context_id, '')::text AS context_id, r.reason, r.status,
       r.resolved_by, COALESCE(ru.display_name, '')::text AS resolved_by_name, r.created_at
FROM reports r
JOIN users u ON r.reporter_id = u.id
LEFT JOIN users ru ON r.resolved_by = ru.id
ORDER BY r.created_at DESC LIMIT $1 OFFSET $2;

-- name: ListReportsByStatus :many
SELECT r.id, r.reporter_id, u.display_name, u.avatar_url,
       r.target_type, r.target_id, COALESCE(r.context_id, '')::text AS context_id, r.reason, r.status,
       r.resolved_by, COALESCE(ru.display_name, '')::text AS resolved_by_name, r.created_at
FROM reports r
JOIN users u ON r.reporter_id = u.id
LEFT JOIN users ru ON r.resolved_by = ru.id
WHERE r.status = $1
ORDER BY r.created_at DESC LIMIT $2 OFFSET $3;

-- name: GetReportByID :one
SELECT r.id, r.reporter_id, u.display_name, u.avatar_url,
       r.target_type, r.target_id, COALESCE(r.context_id, '')::text AS context_id, r.reason, r.status,
       r.resolved_by, COALESCE(ru.display_name, '')::text AS resolved_by_name, r.created_at
FROM reports r
JOIN users u ON r.reporter_id = u.id
LEFT JOIN users ru ON r.resolved_by = ru.id
WHERE r.id = $1;

-- name: ResolveReport :exec
UPDATE reports SET status = 'resolved', resolved_by = $1, resolution_comment = $2 WHERE id = $3;
