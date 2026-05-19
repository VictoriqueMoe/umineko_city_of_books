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

func (r Role) Rank() int {
	switch r {
	case RoleSuperAdmin:
		return 4
	case RoleAdmin:
		return 3
	case RoleModerator:
		return 2
	default:
		return 0
	}
}
