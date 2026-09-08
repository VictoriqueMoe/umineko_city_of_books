-- name: ListRolePermissions :many
SELECT role, permission FROM role_permissions ORDER BY role, permission;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions WHERE role = $1;

-- name: InsertRolePermission :exec
INSERT INTO role_permissions (role, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListVanityRolePermissions :many
SELECT vrp.vanity_role_id, vrp.permission
FROM vanity_role_permissions vrp
JOIN vanity_roles vr ON vr.id = vrp.vanity_role_id
WHERE vr.is_system = FALSE
ORDER BY vrp.vanity_role_id, vrp.permission;

-- name: DeleteVanityRolePermissions :exec
DELETE FROM vanity_role_permissions WHERE vanity_role_id = $1;

-- name: InsertVanityRolePermission :exec
INSERT INTO vanity_role_permissions (vanity_role_id, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListUserVanityRoleIDs :many
SELECT vanity_role_id FROM user_vanity_roles WHERE user_id = $1 ORDER BY vanity_role_id;
