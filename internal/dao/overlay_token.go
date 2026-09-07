package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	OverlayTokenDAO interface {
		GetByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetUserByToken(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, error)
		Upsert(ctx context.Context, s spec.OverlayTokenUpsert, tx ...*sql.Tx) error
		Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	overlayTokenDAO struct {
		db *sql.DB
	}
)

func (r *overlayTokenDAO) GetByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	var token string

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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

func (r *overlayTokenDAO) GetUserByToken(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, error) {
	var userID uuid.UUID

	err := txOrDB(r.db, tx).QueryRowContext(ctx,
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

func (r *overlayTokenDAO) Upsert(ctx context.Context, s spec.OverlayTokenUpsert, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO overlay_tokens (user_id, token)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE
		    SET token = excluded.token,
		        updated_at = NOW()`,
		s.UserID, s.Token,
	)
	if err != nil {
		return fmt.Errorf("upsert overlay token: %w", err)
	}

	return nil
}

func (r *overlayTokenDAO) Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM overlay_tokens WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete overlay token: %w", err)
	}

	return nil
}
