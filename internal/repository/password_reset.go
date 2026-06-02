package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

	passwordResetRepository struct {
		db *sql.DB
	}
)

func (r *passwordResetRepository) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *passwordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	var t PasswordResetToken
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

func (r *passwordResetRepository) MarkUsed(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1`, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *passwordResetRepository) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete unused password reset tokens: %w", err)
	}
	return nil
}
