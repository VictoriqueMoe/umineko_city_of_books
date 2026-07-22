package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	OverlayTokenRepository interface {
		GetByUser(ctx context.Context, userID uuid.UUID) (string, error)
		GetUserByToken(ctx context.Context, token string) (uuid.UUID, error)
		Upsert(ctx context.Context, userID uuid.UUID, token string) error
		Delete(ctx context.Context, userID uuid.UUID) error
	}
)

type overlayTokenRepository struct {
	dao OverlayTokenRepository
}

func NewOverlayTokenRepo(dao OverlayTokenRepository) OverlayTokenRepository {
	return &overlayTokenRepository{dao: dao}
}

func (r *overlayTokenRepository) GetByUser(ctx context.Context, userID uuid.UUID) (string, error) {
	return r.dao.GetByUser(ctx, userID)
}

func (r *overlayTokenRepository) GetUserByToken(ctx context.Context, token string) (uuid.UUID, error) {
	return r.dao.GetUserByToken(ctx, token)
}

func (r *overlayTokenRepository) Upsert(ctx context.Context, userID uuid.UUID, token string) error {
	return r.dao.Upsert(ctx, userID, token)
}

func (r *overlayTokenRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	return r.dao.Delete(ctx, userID)
}
