package sidebar

import (
	"context"
	"errors"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

const visitedKeyMaxLen = 100

var (
	ErrEmptyKey   = errors.New("key is required")
	ErrKeyTooLong = errors.New("key too long")
)

type (
	Service interface {
		ListVisited(ctx context.Context, userID uuid.UUID) (*dto.SidebarLastVisitedResponse, error)
		MarkVisited(ctx context.Context, userID uuid.UUID, key string) error
	}

	service struct {
		repo repository.SidebarLastVisitedRepository
	}
)

func NewService(repo repository.SidebarLastVisitedRepository) Service {
	return &service{repo: repo}
}

func (s *service) ListVisited(ctx context.Context, userID uuid.UUID) (*dto.SidebarLastVisitedResponse, error) {
	visited, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.SidebarLastVisitedResponse{Visited: visited}, nil
}

func (s *service) MarkVisited(ctx context.Context, userID uuid.UUID, key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if len(key) > visitedKeyMaxLen {
		return ErrKeyTooLong
	}
	return s.repo.Upsert(ctx, userID, key)
}
