package repository

import (
	"context"

	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	RoleRepository interface {
		GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error)
		GetRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]role.Role, error)
		HasRole(ctx context.Context, userID uuid.UUID, r role.Role) (bool, error)
		SetRole(ctx context.Context, userID uuid.UUID, r role.Role) error
		RemoveRole(ctx context.Context, userID uuid.UUID, r role.Role) error
		GetUsersByRoles(ctx context.Context, roles []role.Role) ([]uuid.UUID, error)
	}
)
