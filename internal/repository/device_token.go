package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	DeviceRegistration struct {
		Token    string
		Platform string
	}

	DeviceTokenRepository interface {
		Upsert(ctx context.Context, userID uuid.UUID, token, platform string, tx ...*sql.Tx) error
		RegistrationsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]DeviceRegistration, error)
		Delete(ctx context.Context, userID uuid.UUID, token string, tx ...*sql.Tx) error
		DeleteMany(ctx context.Context, userID uuid.UUID, tokens []string, tx ...*sql.Tx) error
	}
)

type deviceTokenRepository struct {
	dao DeviceTokenRepository
}

func NewDeviceTokenRepo(dao DeviceTokenRepository) DeviceTokenRepository {
	return &deviceTokenRepository{dao: dao}
}

func (r *deviceTokenRepository) Upsert(ctx context.Context, userID uuid.UUID, token, platform string, tx ...*sql.Tx) error {
	return r.dao.Upsert(ctx, userID, token, platform, tx...)
}

func (r *deviceTokenRepository) RegistrationsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]DeviceRegistration, error) {
	return r.dao.RegistrationsForUser(ctx, userID, tx...)
}

func (r *deviceTokenRepository) Delete(ctx context.Context, userID uuid.UUID, token string, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, userID, token, tx...)
}

func (r *deviceTokenRepository) DeleteMany(ctx context.Context, userID uuid.UUID, tokens []string, tx ...*sql.Tx) error {
	return r.dao.DeleteMany(ctx, userID, tokens, tx...)
}
