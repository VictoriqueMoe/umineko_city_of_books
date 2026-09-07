package spec

import (
	"github.com/google/uuid"
)

type (
	VanityRoleMove struct {
		UserID     uuid.UUID
		FromRoleID string
		ToRoleID   string
	}

	NewVanityRole struct {
		ID        string
		Label     string
		Color     string
		SortOrder int
	}

	VanityRoleUpdate struct {
		ID        string
		Label     string
		Color     string
		SortOrder int
	}

	VanityRoleAssignment struct {
		UserID uuid.UUID
		RoleID string
	}

	VanityRoleUserQuery struct {
		RoleID string
		Search string
		Limit  int
		Offset int
	}
)
