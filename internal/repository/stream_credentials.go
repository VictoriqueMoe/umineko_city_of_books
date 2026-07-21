package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	StreamCredentialsRow struct {
		UserID    uuid.UUID
		IngressID string
		WhipURL   string
		StreamKey string
		Room      string
	}

	StreamCredentialsRepository interface {
		Get(ctx context.Context, userID uuid.UUID) (*StreamCredentialsRow, error)
		Upsert(ctx context.Context, userID uuid.UUID, ingressID, whipURL, streamKey, room string) error
		Delete(ctx context.Context, userID uuid.UUID) error
	}
)
