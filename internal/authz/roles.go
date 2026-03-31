package authz

import "umineko_city_of_books/internal/role"

const (
	RoleSuperAdmin role.Role = "super_admin"
	RoleAdmin      role.Role = "admin"
	RoleModerator  role.Role = "moderator"
)
