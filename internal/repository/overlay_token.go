package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	OverlayTokenRepository interface {
		GetByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetUserByToken(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, error)
		Upsert(ctx context.Context, userID uuid.UUID, token string, tx ...*sql.Tx) error
		Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
	}
)

type overlayTokenRepository struct {
	dao OverlayTokenRepository
}

func NewOverlayTokenRepo(dao OverlayTokenRepository) OverlayTokenRepository {
	return &overlayTokenRepository{dao: dao}
}

func (r *overlayTokenRepository) GetByUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error) {
	return r.dao.GetByUser(ctx, userID, tx...)
}

func (r *overlayTokenRepository) GetUserByToken(ctx context.Context, token string, tx ...*sql.Tx) (uuid.UUID, error) {
	return r.dao.GetUserByToken(ctx, token, tx...)
}

func (r *overlayTokenRepository) Upsert(ctx context.Context, userID uuid.UUID, token string, tx ...*sql.Tx) error {
	return r.dao.Upsert(ctx, userID, token, tx...)
}

func (r *overlayTokenRepository) Delete(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, userID, tx...)
}
