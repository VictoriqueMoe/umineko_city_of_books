package authz

import (
	"context"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	Service interface {
		Can(ctx context.Context, userID uuid.UUID, perm Permission) bool
		GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error)
	}

	service struct {
		roleRepo repository.RoleRepository
	}
)

func NewService(roleRepo repository.RoleRepository) Service {
	return &service{roleRepo: roleRepo}
}

func (s *service) Can(ctx context.Context, userID uuid.UUID, perm Permission) bool {
	if userID == uuid.Nil {
		return false
	}

	r, err := s.roleRepo.GetRole(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to get role for permission check")
		return false
	}
	if r == "" {
		return false
	}

	perms, ok := rolePermissions[r]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == PermAll || p == perm {
			return true
		}
	}
	return false
}

func (s *service) GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error) {
	return s.roleRepo.GetRole(ctx, userID)
}
