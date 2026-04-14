package role

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleModerator  Role = "moderator"
)

func (r Role) IsSiteStaff() bool {
	return r == RoleSuperAdmin || r == RoleAdmin || r == RoleModerator
}
