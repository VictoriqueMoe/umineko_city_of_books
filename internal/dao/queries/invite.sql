-- name: CreateInvite :exec
INSERT INTO invites (code, created_by) VALUES ($1, $2);

-- name: GetInviteByCode :one
SELECT code, created_by, used_by, used_at, created_at
FROM invites
WHERE code = $1;

-- name: MarkInviteUsed :execrows
UPDATE invites SET used_by = $1, used_at = NOW() WHERE code = $2 AND used_by IS NULL;

-- name: CountInvites :one
SELECT COUNT(*) FROM invites;

-- name: ListInvites :many
SELECT code, created_by, used_by, used_at, created_at
FROM invites
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteInvite :exec
DELETE FROM invites WHERE code = $1;
