package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model/spec"
)

type (
	SidebarLastVisitedDAO interface {
		Upsert(ctx context.Context, s spec.NewSidebarVisit, tx ...*sql.Tx) error
		ListForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (map[string]string, error)
	}

	sidebarLastVisitedDAO struct {
		db *sql.DB
	}
)

func (r *sidebarLastVisitedDAO) Upsert(ctx context.Context, s spec.NewSidebarVisit, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertSidebarLastVisited(ctx, sqlcgen.UpsertSidebarLastVisitedParams{
		UserID: s.UserID,
		Key:    s.Key,
	})
	if err != nil {
		return fmt.Errorf("upsert sidebar last visited: %w", err)
	}

	return nil
}

func (r *sidebarLastVisitedDAO) ListForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (map[string]string, error) {
	rows, err := genQueries(r.db, tx).ListSidebarLastVisitedForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sidebar last visited: %w", err)
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Key] = row.VisitedAt.UTC().Format(time.RFC3339)
	}

	return out, nil
}
