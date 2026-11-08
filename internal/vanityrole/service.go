package vanityrole

import (
	"context"

	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/repository"
)

type (
	Service interface {
		List(ctx context.Context) ([]model.VanityRoleRow, error)
		GetAllAssignments(ctx context.Context) (map[string][]string, error)
	}

	service struct {
		repo        repository.VanityRoleRepository
		permissions repository.PermissionRepository
	}
)

func NewService(repo repository.VanityRoleRepository, permissions repository.PermissionRepository) Service {
	return &service{repo: repo, permissions: permissions}
}

func (s *service) List(ctx context.Context) ([]model.VanityRoleRow, error) {
	return s.repo.List(ctx)
}

func (s *service) GetAllAssignments(ctx context.Context) (map[string][]string, error) {
	return s.permissions.GetAllVanityRoleAssignments(ctx)
}
