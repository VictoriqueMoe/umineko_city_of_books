package repository

import (
	"context"
	"database/sql"
	"time"

	"umineko_city_of_books/internal/db"

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

	NewEmailVerification struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
	}

	EmailVerificationDAO interface {
		Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time, tx ...*sql.Tx) error
		GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*EmailVerificationToken, error)
		MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error
		DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	EmailVerificationRepository interface {
		EmailVerificationDAO

		Issue(ctx context.Context, spec NewEmailVerification, tx ...*sql.Tx) error
	}
)

type emailVerificationRepository struct {
	db  *sql.DB
	dao EmailVerificationDAO
}

func NewEmailVerificationRepo(database *sql.DB, dao EmailVerificationDAO) EmailVerificationRepository {
	return &emailVerificationRepository{db: database, dao: dao}
}

func (r *emailVerificationRepository) Issue(ctx context.Context, spec NewEmailVerification, tx ...*sql.Tx) error {
	return db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.DeleteUnusedForUser(ctx, spec.UserID, tx); err != nil {
			return err
		}

		return r.dao.Create(ctx, spec.TokenHash, spec.UserID, spec.ExpiresAt, tx)
	})
}

func (r *emailVerificationRepository) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time, tx ...*sql.Tx) error {
	return r.dao.Create(ctx, tokenHash, userID, expiresAt, tx...)
}

func (r *emailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*EmailVerificationToken, error) {
	return r.dao.GetByTokenHash(ctx, tokenHash, tx...)
}

func (r *emailVerificationRepository) MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error {
	return r.dao.MarkUsed(ctx, tokenHash, tx...)
}

func (r *emailVerificationRepository) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteUnusedForUser(ctx, userID, tx...)
}
