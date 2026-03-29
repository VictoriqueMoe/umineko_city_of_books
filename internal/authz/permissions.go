package authz

import "umineko_city_of_books/internal/role"

type Permission string

const (
	PermAll               Permission = "*"
	PermDeleteAnyTheory   Permission = "delete_any_theory"
	PermDeleteAnyResponse Permission = "delete_any_response"
	PermBanUser           Permission = "ban_user"
	PermManageRoles       Permission = "manage_roles"
)

var rolePermissions = map[role.Role][]Permission{
	RoleAdmin: {
		PermAll,
	},
	RoleModerator: {
		PermDeleteAnyTheory,
		PermDeleteAnyResponse,
	},
}
