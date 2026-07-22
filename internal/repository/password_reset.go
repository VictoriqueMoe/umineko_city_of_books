package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	PasswordResetToken struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
		UsedAt    *time.Time
		CreatedAt time.Time
	}

	PasswordResetRepository interface {
		Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error
		GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
		MarkUsed(ctx context.Context, tokenHash string) error
		DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error
	}
)

type passwordResetRepository struct {
	dao PasswordResetRepository
}

func NewPasswordResetRepo(dao PasswordResetRepository) PasswordResetRepository {
	return &passwordResetRepository{dao: dao}
}

func (r *passwordResetRepository) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	return r.dao.Create(ctx, tokenHash, userID, expiresAt)
}

func (r *passwordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	return r.dao.GetByTokenHash(ctx, tokenHash)
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, tokenHash string) error {
	return r.dao.MarkUsed(ctx, tokenHash)
}

func (r *passwordResetRepository) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error {
	return r.dao.DeleteUnusedForUser(ctx, userID)
}
