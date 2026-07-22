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

type streamCredentialsRepository struct {
	dao StreamCredentialsRepository
}

func NewStreamCredentialsRepo(dao StreamCredentialsRepository) StreamCredentialsRepository {
	return &streamCredentialsRepository{dao: dao}
}

func (r *streamCredentialsRepository) Get(ctx context.Context, userID uuid.UUID) (*StreamCredentialsRow, error) {
	return r.dao.Get(ctx, userID)
}

func (r *streamCredentialsRepository) Upsert(ctx context.Context, userID uuid.UUID, ingressID, whipURL, streamKey, room string) error {
	return r.dao.Upsert(ctx, userID, ingressID, whipURL, streamKey, room)
}

func (r *streamCredentialsRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	return r.dao.Delete(ctx, userID)
}
