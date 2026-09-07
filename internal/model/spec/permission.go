package spec

type (
	RolePermissionsUpdate struct {
		RoleName    string
		Permissions []string
	}

	VanityRolePermissionsUpdate struct {
		VanityRoleID string
		Permissions  []string
	}
)
