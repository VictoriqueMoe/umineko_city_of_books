-- name: InsertPendingUpload :exec
INSERT INTO pending_uploads (id, user_id, media_type, filename, file_size)
VALUES ($1, $2, $3, $4, $5);

-- name: MarkPendingUploadReady :execrows
UPDATE pending_uploads
SET status = 'ready', media_url = $2, thumbnail_url = $3, width = $4, height = $5
WHERE id = $1 AND status = 'processing';

-- name: MarkPendingUploadFailed :execrows
UPDATE pending_uploads SET status = 'failed' WHERE id = $1 AND status = 'processing';

-- name: FailProcessingPendingUploads :execrows
UPDATE pending_uploads SET status = 'failed' WHERE status = 'processing';

-- name: ListPendingUploads :many
SELECT * FROM pending_uploads
WHERE user_id = $1 AND id = ANY(string_to_array($2::text, ',')::uuid[]);

-- name: ClaimPendingUploads :many
DELETE FROM pending_uploads
WHERE user_id = $1
  AND id = ANY(string_to_array($2::text, ',')::uuid[])
  AND status = 'ready'
  AND created_at > $3
  AND media_type = ANY(string_to_array($4::text, ','))
RETURNING *;

-- name: DeletePendingUpload :many
DELETE FROM pending_uploads WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteExpiredPendingUploads :many
DELETE FROM pending_uploads WHERE created_at <= $1
RETURNING *;
