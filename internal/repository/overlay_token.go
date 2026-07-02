package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type (
	OverlayTokenRepository interface {
		GetByUser(ctx context.Context, userID uuid.UUID) (string, error)
		GetUserByToken(ctx context.Context, token string) (uuid.UUID, error)
		Upsert(ctx context.Context, userID uuid.UUID, token string) error
		Delete(ctx context.Context, userID uuid.UUID) error
	}

	overlayTokenRepository struct {
		db *sql.DB
	}
)

func (r *overlayTokenRepository) GetByUser(ctx context.Context, userID uuid.UUID) (string, error) {
	var token string
	err := r.db.QueryRowContext(ctx,
		`SELECT token FROM overlay_tokens WHERE user_id = $1`,
		userID,
	).Scan(&token)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get overlay token: %w", err)
	}

	return token, nil
}

func (r *overlayTokenRepository) GetUserByToken(ctx context.Context, token string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM overlay_tokens WHERE token = $1`,
		token,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get overlay token user: %w", err)
	}

	return userID, nil
}

func (r *overlayTokenRepository) Upsert(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO overlay_tokens (user_id, token)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE
		    SET token = excluded.token,
		        updated_at = NOW()`,
		userID, token,
	)
	if err != nil {
		return fmt.Errorf("upsert overlay token: %w", err)
	}

	return nil
}

func (r *overlayTokenRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM overlay_tokens WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete overlay token: %w", err)
	}

	return nil
}
