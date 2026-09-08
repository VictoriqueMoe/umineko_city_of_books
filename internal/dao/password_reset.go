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
	PasswordResetDAO interface {
		Create(ctx context.Context, s spec.NewPasswordReset, tx ...*sql.Tx) error
		GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*model.PasswordResetToken, error)
		MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error
		DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	passwordResetDAO struct {
		db *sql.DB
	}
)

func (r *passwordResetDAO) Create(ctx context.Context, s spec.NewPasswordReset, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreatePasswordResetToken(ctx, sqlcgen.CreatePasswordResetTokenParams{
		TokenHash: s.TokenHash,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}

	return nil
}

func (r *passwordResetDAO) GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*model.PasswordResetToken, error) {
	row, err := genQueries(r.db, tx).GetPasswordResetTokenByHash(ctx, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get password reset token: %w", err)
	}

	t := model.PasswordResetToken{
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

func (r *passwordResetDAO) MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkPasswordResetTokenUsed(ctx, tokenHash); err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}

	return nil
}

func (r *passwordResetDAO) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteUnusedPasswordResetTokensForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete unused password reset tokens: %w", err)
	}

	return nil
}
