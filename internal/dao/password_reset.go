package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	passwordResetDAO struct {
		db *sql.DB
	}

	passwordResetRepository struct {
		repository.PasswordResetRepository
	}
)

func (r *passwordResetDAO) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *passwordResetDAO) GetByTokenHash(ctx context.Context, tokenHash string) (*repository.PasswordResetToken, error) {
	var t repository.PasswordResetToken
	err := r.db.QueryRowContext(ctx,
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

func (r *passwordResetDAO) MarkUsed(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1`, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *passwordResetDAO) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete unused password reset tokens: %w", err)
	}
	return nil
}
