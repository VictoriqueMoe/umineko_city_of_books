package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	DeviceTokenRepository interface {
		Upsert(ctx context.Context, userID uuid.UUID, token, platform string) error
		TokensForUser(ctx context.Context, userID uuid.UUID) ([]string, error)
		Delete(ctx context.Context, token string) error
		DeleteMany(ctx context.Context, tokens []string) error
	}
)

type deviceTokenRepository struct {
	dao DeviceTokenRepository
}

func NewDeviceTokenRepo(dao DeviceTokenRepository) DeviceTokenRepository {
	return &deviceTokenRepository{dao: dao}
}

func (r *deviceTokenRepository) Upsert(ctx context.Context, userID uuid.UUID, token, platform string) error {
	return r.dao.Upsert(ctx, userID, token, platform)
}

func (r *deviceTokenRepository) TokensForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return r.dao.TokensForUser(ctx, userID)
}

func (r *deviceTokenRepository) Delete(ctx context.Context, token string) error {
	return r.dao.Delete(ctx, token)
}

func (r *deviceTokenRepository) DeleteMany(ctx context.Context, tokens []string) error {
	return r.dao.DeleteMany(ctx, tokens)
}
