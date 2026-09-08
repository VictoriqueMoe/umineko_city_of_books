package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	SettingsDAO interface {
		Get(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) (string, error)
		GetAll(ctx context.Context, tx ...*sql.Tx) (map[config.SiteSettingKey]string, error)
		Set(ctx context.Context, s spec.SettingsUpdate, tx ...*sql.Tx) error
		SetMultiple(ctx context.Context, s spec.SettingsBulkUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) error
	}

	settingsDAO struct {
		db *sql.DB
	}
)

func settingsActor(updatedBy uuid.UUID) *uuid.UUID {
	if updatedBy == uuid.Nil {
		return nil
	}

	return &updatedBy
}

func (r *settingsDAO) Get(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) (string, error) {
	value, err := genQueries(r.db, tx).GetSiteSetting(ctx, string(key))
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}

	return value, nil
}

func (r *settingsDAO) GetAll(ctx context.Context, tx ...*sql.Tx) (map[config.SiteSettingKey]string, error) {
	rows, err := genQueries(r.db, tx).GetAllSiteSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}

	values := make(map[config.SiteSettingKey]string, len(rows))
	for _, row := range rows {
		values[config.SiteSettingKey(row.Key)] = row.Value
	}

	return values, nil
}

func (r *settingsDAO) Set(ctx context.Context, s spec.SettingsUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertSiteSetting(ctx, sqlcgen.UpsertSiteSettingParams{
		Key:       string(s.Key),
		Value:     s.Value,
		UpdatedBy: settingsActor(s.UpdatedBy),
	})
	if err != nil {
		return fmt.Errorf("set setting %q: %w", s.Key, err)
	}

	return nil
}

func (r *settingsDAO) SetMultiple(ctx context.Context, s spec.SettingsBulkUpdate, tx ...*sql.Tx) error {
	actor := settingsActor(s.UpdatedBy)

	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		queries := genQueries(r.db, []*sql.Tx{tx})

		for key, value := range s.Values {
			err := queries.UpsertSiteSetting(ctx, sqlcgen.UpsertSiteSettingParams{
				Key:       string(key),
				Value:     value,
				UpdatedBy: actor,
			})
			if err != nil {
				return fmt.Errorf("set setting %q: %w", key, err)
			}
		}

		return nil
	})
}

func (r *settingsDAO) Delete(ctx context.Context, key config.SiteSettingKey, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteSiteSetting(ctx, string(key)); err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}

	return nil
}
