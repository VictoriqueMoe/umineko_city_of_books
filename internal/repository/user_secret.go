package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	UserSecretRepository interface {
		Unlock(ctx context.Context, userID uuid.UUID, secretID string) error
		ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error)
		GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error)
		GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string) ([]uuid.UUID, error)
		IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error)
		DeleteSecrets(ctx context.Context, secretIDs []string) error
	}
)
