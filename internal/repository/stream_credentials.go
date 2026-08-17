package repository

import (
	"context"
	"database/sql"

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
		Get(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*StreamCredentialsRow, error)
		Upsert(ctx context.Context, spec NewStreamCredentials, tx ...*sql.Tx) error
		Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}

	NewStreamCredentials struct {
		UserID    uuid.UUID
		IngressID string
		WhipURL   string
		StreamKey string
		Room      string
	}
)

type streamCredentialsRepository struct {
	dao StreamCredentialsRepository
}

func NewStreamCredentialsRepo(dao StreamCredentialsRepository) StreamCredentialsRepository {
	return &streamCredentialsRepository{dao: dao}
}

func (r *streamCredentialsRepository) Get(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (*StreamCredentialsRow, error) {
	return r.dao.Get(ctx, userID, tx...)
}

func (r *streamCredentialsRepository) Upsert(ctx context.Context, spec NewStreamCredentials, tx ...*sql.Tx) error {
	return r.dao.Upsert(ctx, spec, tx...)
}

func (r *streamCredentialsRepository) Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, userID, tx...)
}
