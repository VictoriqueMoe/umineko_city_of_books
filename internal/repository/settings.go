package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	SettingsDAO interface {
		Get(ctx context.Context, key string, tx ...*sql.Tx) (string, error)
		GetAll(ctx context.Context, tx ...*sql.Tx) (map[string]string, error)
		Set(ctx context.Context, key, value string, updatedBy uuid.UUID, tx ...*sql.Tx) error
		SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID, tx ...*sql.Tx) error
		Delete(ctx context.Context, key string, tx ...*sql.Tx) error
	}

	SettingsRepository interface {
		SettingsDAO

		Reconcile(ctx context.Context, spec SettingsReconcile, tx ...*sql.Tx) error
	}

	SettingsReconcile struct {
		Missing   map[string]string
		Stale     []string
		UpdatedBy uuid.UUID
	}
)

type settingsRepository struct {
	db  *sql.DB
	dao SettingsDAO
}

func NewSettingsRepo(database *sql.DB, dao SettingsDAO) SettingsRepository {
	return &settingsRepository{db: database, dao: dao}
}

func (r *settingsRepository) Reconcile(ctx context.Context, spec SettingsReconcile, tx ...*sql.Tx) error {
	return db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
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
}

func (r *settingsRepository) Get(ctx context.Context, key string, tx ...*sql.Tx) (string, error) {
	return r.dao.Get(ctx, key, tx...)
}

func (r *settingsRepository) GetAll(ctx context.Context, tx ...*sql.Tx) (map[string]string, error) {
	return r.dao.GetAll(ctx, tx...)
}

func (r *settingsRepository) Set(ctx context.Context, key, value string, updatedBy uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Set(ctx, key, value, updatedBy, tx...)
}

func (r *settingsRepository) SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.SetMultiple(ctx, settings, updatedBy, tx...)
}

func (r *settingsRepository) Delete(ctx context.Context, key string, tx ...*sql.Tx) error {
	return r.dao.Delete(ctx, key, tx...)
}
