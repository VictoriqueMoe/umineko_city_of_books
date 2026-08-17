package repository

import (
	"context"
	"database/sql"
	"time"

	"umineko_city_of_books/internal/db"

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

	NewPasswordReset struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
	}

	PasswordResetDAO interface {
		Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time, tx ...*sql.Tx) error
		GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*PasswordResetToken, error)
		MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error
		DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	PasswordResetRepository interface {
		PasswordResetDAO

		Issue(ctx context.Context, spec NewPasswordReset, tx ...*sql.Tx) error
	}
)

type passwordResetRepository struct {
	db  *sql.DB
	dao PasswordResetDAO
}

func NewPasswordResetRepo(database *sql.DB, dao PasswordResetDAO) PasswordResetRepository {
	return &passwordResetRepository{db: database, dao: dao}
}

func (r *passwordResetRepository) Issue(ctx context.Context, spec NewPasswordReset, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.DeleteUnusedForUser(ctx, spec.UserID, tx); err != nil {
			return err
		}

		return r.dao.Create(ctx, spec.TokenHash, spec.UserID, spec.ExpiresAt, tx)
	})
}

func (r *passwordResetRepository) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time, tx ...*sql.Tx) error {
	return r.dao.Create(ctx, tokenHash, userID, expiresAt, tx...)
}

func (r *passwordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*PasswordResetToken, error) {
	return r.dao.GetByTokenHash(ctx, tokenHash, tx...)
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error {
	return r.dao.MarkUsed(ctx, tokenHash, tx...)
}

func (r *passwordResetRepository) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.DeleteUnusedForUser(ctx, userID, tx...)
}
