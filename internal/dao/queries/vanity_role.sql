-- name: ListVanityRoles :many
SELECT id, label, color, is_system, sort_order FROM vanity_roles ORDER BY sort_order, label;

-- name: GetVanityRoleByID :one
SELECT id, label, color, is_system, sort_order FROM vanity_roles WHERE id = $1;

-- name: CreateVanityRole :exec
INSERT INTO vanity_roles (id, label, color, sort_order) VALUES ($1, $2, $3, $4);

-- name: UpdateVanityRole :exec
UPDATE vanity_roles SET label = $1, color = $2, sort_order = $3 WHERE id = $4;

-- name: DeleteVanityRole :exec
DELETE FROM vanity_roles WHERE id = $1 AND is_system = FALSE;

-- name: AssignVanityRoleToUser :exec
INSERT INTO user_vanity_roles (user_id, vanity_role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnassignVanityRoleFromUser :exec
DELETE FROM user_vanity_roles WHERE user_id = $1 AND vanity_role_id = $2;

-- name: CountVanityRoleUsers :one
SELECT COUNT(*)
FROM user_vanity_roles uvr
JOIN users u ON uvr.user_id = u.id
WHERE uvr.vanity_role_id = $1
  AND ($2::text = ''
       OR u.username LIKE '%' || $2::text || '%'
       OR u.display_name LIKE '%' || $2::text || '%');

-- name: ListVanityRoleUsers :many
SELECT u.id, u.username, u.display_name, u.avatar_url
FROM user_vanity_roles uvr
JOIN users u ON uvr.user_id = u.id
WHERE uvr.vanity_role_id = $1
  AND ($2::text = ''
       OR u.username LIKE '%' || $2::text || '%'
       OR u.display_name LIKE '%' || $2::text || '%')
ORDER BY LOWER(u.display_name)
LIMIT $3 OFFSET $4;

-- name: GetVanityRolesForUser :many
SELECT vr.id, vr.label, vr.color, vr.is_system, vr.sort_order
FROM vanity_roles vr
JOIN user_vanity_roles uvr ON vr.id = uvr.vanity_role_id
WHERE uvr.user_id = $1
ORDER BY vr.sort_order, vr.label;

-- name: GetVanityRolesForUsersBatch :many
SELECT uvr.user_id, vr.id, vr.label, vr.color, vr.is_system, vr.sort_order
FROM user_vanity_roles uvr
JOIN vanity_roles vr ON vr.id = uvr.vanity_role_id
WHERE uvr.user_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY vr.sort_order, vr.label;

-- name: GetAllVanityRoleAssignments :many
SELECT user_id, vanity_role_id FROM user_vanity_roles ORDER BY user_id;
