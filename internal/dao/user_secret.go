package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	UserSecretDAO interface {
		Unlock(ctx context.Context, s spec.SecretUnlock, tx ...*sql.Tx) error
		ListForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		GetUserIDsWithSecret(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string, tx ...*sql.Tx) ([]uuid.UUID, error)
		IsSolvedByAnyone(ctx context.Context, secretID string, tx ...*sql.Tx) (bool, error)
		DeleteSecrets(ctx context.Context, secretIDs []string, tx ...*sql.Tx) error
	}

	userSecretDAO struct {
		db *sql.DB
	}
)

func (r *userSecretDAO) Unlock(ctx context.Context, s spec.SecretUnlock, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UnlockUserSecret(ctx, sqlcgen.UnlockUserSecretParams{
		UserID:   s.UserID,
		SecretID: s.SecretID,
	})
	if err != nil {
		return fmt.Errorf("unlock secret: %w", err)
	}

	return nil
}

func (r *userSecretDAO) ListForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).ListUserSecretIDsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user secrets: %w", err)
	}

	return rows, nil
}

func (r *userSecretDAO) GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string, tx ...*sql.Tx) ([]uuid.UUID, error) {
	if len(pieceIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListUserIDsWithAnyUserSecret(ctx, strings.Join(pieceIDs, ","))
	if err != nil {
		return nil, fmt.Errorf("list piece participants: %w", err)
	}

	return rows, nil
}

func (r *userSecretDAO) IsSolvedByAnyone(ctx context.Context, secretID string, tx ...*sql.Tx) (bool, error) {
	_, err := genQueries(r.db, tx).GetUserSecretSolvedMarker(ctx, secretID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check secret solved: %w", err)
	}

	return true, nil
}

func (r *userSecretDAO) DeleteSecrets(ctx context.Context, secretIDs []string, tx ...*sql.Tx) error {
	if len(secretIDs) == 0 {
		return nil
	}

	if err := genQueries(r.db, tx).DeleteUserSecrets(ctx, strings.Join(secretIDs, ",")); err != nil {
		return fmt.Errorf("delete secrets: %w", err)
	}

	return nil
}

func (r *userSecretDAO) GetUserIDsWithSecret(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error) {
	rows, err := genQueries(r.db, tx).ListUserIDsWithUserSecret(ctx, secretID)
	if err != nil {
		return nil, fmt.Errorf("list secret holders: %w", err)
	}

	return rows, nil
}
