package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/dao/sqlcgen"
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
	token, err := genQueries(r.db, tx).GetOverlayTokenByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get overlay token: %w", err)
	}

	return token, nil
}

func (r *overlayTokenDAO) GetUserByToken(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, error) {
	userID, err := genQueries(r.db, tx).GetOverlayTokenUser(ctx, token)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get overlay token user: %w", err)
	}

	return userID, nil
}

func (r *overlayTokenDAO) Upsert(ctx context.Context, s spec.OverlayTokenUpsert, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertOverlayToken(ctx, sqlcgen.UpsertOverlayTokenParams{
		UserID: s.UserID,
		Token:  s.Token,
	})
	if err != nil {
		return fmt.Errorf("upsert overlay token: %w", err)
	}

	return nil
}

func (r *overlayTokenDAO) Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteOverlayToken(ctx, userID); err != nil {
		return fmt.Errorf("delete overlay token: %w", err)
	}

	return nil
}
