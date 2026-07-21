package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	DeviceTokenRepository interface {
		Upsert(ctx context.Context, userID uuid.UUID, token, platform string) error
		TokensForUser(ctx context.Context, userID uuid.UUID) ([]string, error)
		Delete(ctx context.Context, token string) error
		DeleteMany(ctx context.Context, tokens []string) error
	}
)
