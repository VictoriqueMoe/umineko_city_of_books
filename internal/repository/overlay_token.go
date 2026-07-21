package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	OverlayTokenRepository interface {
		GetByUser(ctx context.Context, userID uuid.UUID) (string, error)
		GetUserByToken(ctx context.Context, token string) (uuid.UUID, error)
		Upsert(ctx context.Context, userID uuid.UUID, token string) error
		Delete(ctx context.Context, userID uuid.UUID) error
	}
)
