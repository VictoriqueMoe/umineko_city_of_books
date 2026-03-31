package authz

import "umineko_city_of_books/internal/role"

type Permission string

const (
	PermAll               Permission = "*"
	PermViewAdminPanel    Permission = "view_admin_panel"
	PermViewStats         Permission = "view_stats"
	PermViewAuditLog      Permission = "view_audit_log"
	PermManageSettings    Permission = "manage_settings"
	PermManageRoles       Permission = "manage_roles"
	PermDeleteAnyTheory   Permission = "delete_any_theory"
	PermDeleteAnyResponse Permission = "delete_any_response"
	PermDeleteAnyUser     Permission = "delete_any_user"
	PermBanUser           Permission = "ban_user"
	PermViewUsers         Permission = "view_users"
)

var rolePermissions = map[role.Role][]Permission{
	RoleSuperAdmin: {
		PermAll,
	},
	RoleAdmin: {
		PermAll,
	},
	RoleModerator: {
		PermViewAdminPanel,
		PermViewStats,
		PermViewUsers,
		PermDeleteAnyTheory,
		PermDeleteAnyResponse,
		PermBanUser,
	},
}
