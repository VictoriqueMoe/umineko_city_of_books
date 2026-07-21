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
