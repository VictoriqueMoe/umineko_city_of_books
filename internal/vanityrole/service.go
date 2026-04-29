package vanityrole

import (
	"context"

	"umineko_city_of_books/internal/repository"
)

type (
	Service interface {
		List(ctx context.Context) ([]repository.VanityRoleRow, error)
		GetAllAssignments(ctx context.Context) (map[string][]string, error)
	}

	service struct {
		repo repository.VanityRoleRepository
	}
)

func NewService(repo repository.VanityRoleRepository) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context) ([]repository.VanityRoleRow, error) {
	return s.repo.List(ctx)
}

func (s *service) GetAllAssignments(ctx context.Context) (map[string][]string, error) {
	return s.repo.GetAllAssignments(ctx)
}
