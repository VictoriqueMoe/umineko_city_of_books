package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	EmailVerificationDAO interface {
		Create(ctx context.Context, s spec.NewEmailVerification, tx ...*sql.Tx) error
		GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*model.EmailVerificationToken, error)
		MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error
		DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	emailVerificationDAO struct {
		db *sql.DB
	}
)

func (r *emailVerificationDAO) Create(ctx context.Context, s spec.NewEmailVerification, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreateEmailVerificationToken(ctx, sqlcgen.CreateEmailVerificationTokenParams{
		TokenHash: s.TokenHash,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}

	return nil
}

func (r *emailVerificationDAO) GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*model.EmailVerificationToken, error) {
	row, err := genQueries(r.db, tx).GetEmailVerificationByTokenHash(ctx, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get email verification token: %w", err)
	}

	t := model.EmailVerificationToken{
		TokenHash: row.TokenHash,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}

	if row.UsedAt.Valid {
		t.UsedAt = new(row.UsedAt.Time)
	}

	return &t, nil
}

func (r *emailVerificationDAO) MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkEmailVerificationUsed(ctx, tokenHash); err != nil {
		return fmt.Errorf("mark email verification token used: %w", err)
	}

	return nil
}

func (r *emailVerificationDAO) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteUnusedEmailVerificationsForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete unused email verification tokens: %w", err)
	}

	return nil
}
