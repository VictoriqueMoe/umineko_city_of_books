package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	SettingsRepository interface {
		Get(ctx context.Context, key string) (string, error)
		GetAll(ctx context.Context) (map[string]string, error)
		Set(ctx context.Context, key, value string, updatedBy uuid.UUID) error
		SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID) error
		Delete(ctx context.Context, key string) error
	}
)

type settingsRepository struct {
	dao SettingsRepository
}

func NewSettingsRepo(dao SettingsRepository) SettingsRepository {
	return &settingsRepository{dao: dao}
}

func (r *settingsRepository) Get(ctx context.Context, key string) (string, error) {
	return r.dao.Get(ctx, key)
}

func (r *settingsRepository) GetAll(ctx context.Context) (map[string]string, error) {
	return r.dao.GetAll(ctx)
}

func (r *settingsRepository) Set(ctx context.Context, key, value string, updatedBy uuid.UUID) error {
	return r.dao.Set(ctx, key, value, updatedBy)
}

func (r *settingsRepository) SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID) error {
	return r.dao.SetMultiple(ctx, settings, updatedBy)
}

func (r *settingsRepository) Delete(ctx context.Context, key string) error {
	return r.dao.Delete(ctx, key)
}
