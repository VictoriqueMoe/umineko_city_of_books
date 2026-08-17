package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"

	"github.com/google/uuid"
)

type (
	SettingsDAO interface {
		Get(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) (string, error)
		GetAll(ctx context.Context, tx ...*sql.Tx) (map[config.SiteSettingKey]string, error)
		Set(ctx context.Context, key config.SiteSettingKey, value string, updatedBy uuid.UUID, tx ...*sql.Tx) error
		SetMultiple(ctx context.Context, settings map[config.SiteSettingKey]string, updatedBy uuid.UUID, tx ...*sql.Tx) error
		Delete(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) error
	}

	SettingsRepository interface {
		SettingsDAO

		Reconcile(ctx context.Context, spec SettingsReconcile, tx ...*sql.Tx) error
	}

	SettingsReconcile struct {
		Missing   map[config.SiteSettingKey]string
		Stale     []config.SiteSettingKey
		UpdatedBy uuid.UUID
	}
)

type settingsRepository struct {
	db    *sql.DB
	dao   SettingsDAO
	cache *cache.Manager
}

func NewSettingsRepo(database *sql.DB, dao SettingsDAO, c *cache.Manager) SettingsRepository {
	return &settingsRepository{db: database, dao: dao, cache: c}
}

func (r *settingsRepository) Reconcile(ctx context.Context, spec SettingsReconcile, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if len(spec.Missing) > 0 {
			if err := r.dao.SetMultiple(ctx, spec.Missing, spec.UpdatedBy, tx); err != nil {
				return err
			}
		}

		for _, key := range spec.Stale {
			if err := r.dao.Delete(ctx, key, tx); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	touched := make([]config.SiteSettingKey, 0, len(spec.Missing)+len(spec.Stale))
	for key := range spec.Missing {
		touched = append(touched, key)
	}
	touched = append(touched, spec.Stale...)

	r.invalidate(ctx, touched...)

	return nil
}

func (r *settingsRepository) Get(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) (string, error) {
	cacheKey := cache.Setting.Key(string(key))

	if cached, err := cache.Get[string](ctx, r.cache, cacheKey); err == nil {
		return cached, nil
	}

	value, err := r.dao.Get(ctx, key, tx...)
	if err != nil {
		return "", err
	}

	_ = cache.Set(ctx, r.cache, cacheKey, value, cache.Setting.TTL)

	return value, nil
}

func (r *settingsRepository) GetAll(ctx context.Context, tx ...*sql.Tx) (map[config.SiteSettingKey]string, error) {
	return r.dao.GetAll(ctx, tx...)
}

func (r *settingsRepository) Set(ctx context.Context, key config.SiteSettingKey, value string, updatedBy uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.Set(ctx, key, value, updatedBy, tx...); err != nil {
		return err
	}

	r.invalidate(ctx, key)

	return nil
}

func (r *settingsRepository) SetMultiple(ctx context.Context, settings map[config.SiteSettingKey]string, updatedBy uuid.UUID, tx ...*sql.Tx) error {
	if err := r.dao.SetMultiple(ctx, settings, updatedBy, tx...); err != nil {
		return err
	}

	touched := make([]config.SiteSettingKey, 0, len(settings))
	for key := range settings {
		touched = append(touched, key)
	}

	r.invalidate(ctx, touched...)

	return nil
}

func (r *settingsRepository) Delete(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) error {
	if err := r.dao.Delete(ctx, key, tx...); err != nil {
		return err
	}

	r.invalidate(ctx, key)

	return nil
}

func (r *settingsRepository) invalidate(ctx context.Context, keys ...config.SiteSettingKey) {
	if len(keys) == 0 {
		return
	}

	cacheKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		cacheKeys = append(cacheKeys, cache.Setting.Key(string(key)))
	}

	if err := r.cache.Del(ctx, cacheKeys...); err != nil {
		logger.Log.Error().Err(err).Msg("failed to invalidate setting caches after write")
	}
}
