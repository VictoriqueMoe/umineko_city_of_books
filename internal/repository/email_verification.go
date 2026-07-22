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

type emailVerificationRepository struct {
	dao EmailVerificationRepository
}

func NewEmailVerificationRepo(dao EmailVerificationRepository) EmailVerificationRepository {
	return &emailVerificationRepository{dao: dao}
}

func (r *emailVerificationRepository) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	return r.dao.Create(ctx, tokenHash, userID, expiresAt)
}

func (r *emailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error) {
	return r.dao.GetByTokenHash(ctx, tokenHash)
}

func (r *emailVerificationRepository) MarkUsed(ctx context.Context, tokenHash string) error {
	return r.dao.MarkUsed(ctx, tokenHash)
}

func (r *emailVerificationRepository) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error {
	return r.dao.DeleteUnusedForUser(ctx, userID)
}
