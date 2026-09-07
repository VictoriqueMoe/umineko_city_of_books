package spec

import (
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	UserRoleSpec struct {
		UserID uuid.UUID
		Role   role.Role
	}
)
