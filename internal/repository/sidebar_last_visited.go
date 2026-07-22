package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	SidebarLastVisitedRepository interface {
		Upsert(ctx context.Context, userID uuid.UUID, key string) error
		ListForUser(ctx context.Context, userID uuid.UUID) (map[string]string, error)
	}
)

type sidebarLastVisitedRepository struct {
	dao SidebarLastVisitedRepository
}

func NewSidebarLastVisitedRepo(dao SidebarLastVisitedRepository) SidebarLastVisitedRepository {
	return &sidebarLastVisitedRepository{dao: dao}
}

func (r *sidebarLastVisitedRepository) Upsert(ctx context.Context, userID uuid.UUID, key string) error {
	return r.dao.Upsert(ctx, userID, key)
}

func (r *sidebarLastVisitedRepository) ListForUser(ctx context.Context, userID uuid.UUID) (map[string]string, error) {
	return r.dao.ListForUser(ctx, userID)
}
