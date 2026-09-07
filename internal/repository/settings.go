package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
)

type (
	SettingsRepository interface {
		dao.SettingsDAO

		Reconcile(ctx context.Context, s spec.SettingsReconcile, tx ...*sql.Tx) error
	}
)

type settingsRepository struct {
	db *sql.DB
	dao.SettingsDAO
	cache *cache.Manager
}

func NewSettingsRepo(database *sql.DB, settingsDAO dao.SettingsDAO, c *cache.Manager) SettingsRepository {
	return &settingsRepository{db: database, SettingsDAO: settingsDAO, cache: c}
}

func (r *settingsRepository) Reconcile(ctx context.Context, s spec.SettingsReconcile, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if len(s.Missing) > 0 {
			update := spec.SettingsBulkUpdate{Values: s.Missing, UpdatedBy: s.UpdatedBy}

			if err := r.SettingsDAO.SetMultiple(ctx, update, tx); err != nil {
				return err
			}
		}

		for _, key := range s.Stale {
			if err := r.SettingsDAO.Delete(ctx, key, tx); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	touched := make([]config.SiteSettingKey, 0, len(s.Missing)+len(s.Stale))
	for key := range s.Missing {
		touched = append(touched, key)
	}
	touched = append(touched, s.Stale...)

	r.invalidate(ctx, touched...)

	return nil
}

func (r *settingsRepository) Get(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) (string, error) {
	load := func(ctx context.Context) (string, error) {
		return r.SettingsDAO.Get(ctx, key, tx...)
	}

	return r.cache.Load(ctx, cache.Setting, load, string(key))
}

func (r *settingsRepository) Set(ctx context.Context, s spec.SettingsUpdate, tx ...*sql.Tx) error {
	if err := r.SettingsDAO.Set(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidate(ctx, s.Key)

	return nil
}

func (r *settingsRepository) SetMultiple(ctx context.Context, s spec.SettingsBulkUpdate, tx ...*sql.Tx) error {
	if err := r.SettingsDAO.SetMultiple(ctx, s, tx...); err != nil {
		return err
	}

	touched := make([]config.SiteSettingKey, 0, len(s.Values))
	for key := range s.Values {
		touched = append(touched, key)
	}

	r.invalidate(ctx, touched...)

	return nil
}

func (r *settingsRepository) Delete(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) error {
	if err := r.SettingsDAO.Delete(ctx, key, tx...); err != nil {
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
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate setting caches after write")
	}
}
