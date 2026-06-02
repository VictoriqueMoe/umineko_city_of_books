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

	emailVerificationRepository struct {
		db *sql.DB
	}
)

func (r *emailVerificationRepository) Create(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO email_verification_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}
	return nil
}

func (r *emailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error) {
	var t EmailVerificationToken
	err := r.db.QueryRowContext(ctx,
		`SELECT token_hash, user_id, expires_at, used_at, created_at FROM email_verification_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.TokenHash, &t.UserID, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get email verification token: %w", err)
	}
	return &t, nil
}

func (r *emailVerificationRepository) MarkUsed(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE email_verification_tokens SET used_at = NOW() WHERE token_hash = $1`, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("mark email verification token used: %w", err)
	}
	return nil
}

func (r *emailVerificationRepository) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_verification_tokens WHERE user_id = $1 AND used_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete unused email verification tokens: %w", err)
	}
	return nil
}
