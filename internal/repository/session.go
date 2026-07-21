package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	SessionRepository interface {
		Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error
		GetUserID(ctx context.Context, token string) (uuid.UUID, time.Time, error)
		Delete(ctx context.Context, token string) error
		DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
		CleanExpired(ctx context.Context) error
	}
)
