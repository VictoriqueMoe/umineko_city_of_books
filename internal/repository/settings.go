package repository

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/db"

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

	settingsRepository struct {
		db *sql.DB
	}
)

func (r *settingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM site_settings WHERE key = $1`, key,
	).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

func (r *settingsRepository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM site_settings`)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

func (r *settingsRepository) Set(ctx context.Context, key, value string, updatedBy uuid.UUID) error {
	var actor any
	if updatedBy != uuid.Nil {
		actor = updatedBy
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO site_settings (key, value, updated_by, updated_at) VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		key, value, actor,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

func (r *settingsRepository) SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID) error {
	var actor any
	if updatedBy != uuid.Nil {
		actor = updatedBy
	}
	return db.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		for key, value := range settings {
			_, err := tx.ExecContext(ctx,
				`INSERT INTO site_settings (key, value, updated_by, updated_at) VALUES ($1, $2, $3, NOW())
				 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
				key, value, actor,
			)
			if err != nil {
				return fmt.Errorf("set setting %q: %w", key, err)
			}
		}
		return nil
	})
}

func (r *settingsRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM site_settings WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}
	return nil
}
