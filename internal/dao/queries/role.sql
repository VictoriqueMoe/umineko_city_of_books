-- name: GetUserRole :one
SELECT role FROM user_roles WHERE user_id = $1 LIMIT 1;

-- name: GetUserRolesBatch :many
SELECT user_id, role FROM user_roles WHERE user_id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: CountUserRole :one
SELECT COUNT(*) FROM user_roles WHERE user_id = $1 AND role = $2;

-- name: ClearUserRoles :exec
DELETE FROM user_roles WHERE user_id = $1;

-- name: InsertUserRole :exec
INSERT INTO user_roles (user_id, role) VALUES ($1, $2);

-- name: DeleteUserRole :exec
DELETE FROM user_roles WHERE user_id = $1 AND role = $2;

-- name: GetUsersByRoles :many
SELECT DISTINCT user_id FROM user_roles WHERE role = ANY(string_to_array($1::text, ','));
