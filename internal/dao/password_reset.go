package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

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
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO password_reset_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.TokenHash, s.UserID, s.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *passwordResetDAO) GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*model.PasswordResetToken, error) {
	var t model.PasswordResetToken
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT token_hash, user_id, expires_at, used_at, created_at FROM password_reset_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.TokenHash, &t.UserID, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get password reset token: %w", err)
	}
	return &t, nil
}

func (r *passwordResetDAO) MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1`, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *passwordResetDAO) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete unused password reset tokens: %w", err)
	}
	return nil
}
