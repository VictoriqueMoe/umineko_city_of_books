package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	EmailVerificationToken struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
		UsedAt    *time.Time
		CreatedAt time.Time
	}

	EmailVerificationRepository interface {
		Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error
		GetByTokenHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)
		MarkUsed(ctx context.Context, tokenHash string) error
		DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error
	}
)
