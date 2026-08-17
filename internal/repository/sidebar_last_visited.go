package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	SidebarLastVisitedRepository interface {
		Upsert(ctx context.Context, userID uuid.UUID, key string, tx ...*sql.Tx) error
		ListForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (map[string]string, error)
	}
)

type sidebarLastVisitedRepository struct {
	dao SidebarLastVisitedRepository
}

func NewSidebarLastVisitedRepo(dao SidebarLastVisitedRepository) SidebarLastVisitedRepository {
	return &sidebarLastVisitedRepository{dao: dao}
}

func (r *sidebarLastVisitedRepository) Upsert(ctx context.Context, userID uuid.UUID, key string, tx ...*sql.Tx) error {
	return r.dao.Upsert(ctx, userID, key, tx...)
}

func (r *sidebarLastVisitedRepository) ListForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (map[string]string, error) {
	return r.dao.ListForUser(ctx, userID, tx...)
}
