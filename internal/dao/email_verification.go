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
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO email_verification_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.TokenHash, s.UserID, s.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}
	return nil
}

func (r *emailVerificationDAO) GetByTokenHash(ctx context.Context, tokenHash string, tx ...*sql.Tx) (*model.EmailVerificationToken, error) {
	var t model.EmailVerificationToken
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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

func (r *emailVerificationDAO) MarkUsed(ctx context.Context, tokenHash string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE email_verification_tokens SET used_at = NOW() WHERE token_hash = $1`, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("mark email verification token used: %w", err)
	}
	return nil
}

func (r *emailVerificationDAO) DeleteUnusedForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM email_verification_tokens WHERE user_id = $1 AND used_at IS NULL`, userID,
	)
	if err != nil {
		return fmt.Errorf("delete unused email verification tokens: %w", err)
	}
	return nil
}
