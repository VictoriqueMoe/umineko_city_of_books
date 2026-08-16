package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	settingsDAO struct {
		db *sql.DB
	}
)

func (r *settingsDAO) Get(ctx context.Context, key string, tx ...*sql.Tx) (string, error) {
	var value string
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT value FROM site_settings WHERE key = $1`, key,
	).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

func (r *settingsDAO) GetAll(ctx context.Context, tx ...*sql.Tx) (map[string]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx, `SELECT key, value FROM site_settings`)
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

func (r *settingsDAO) Set(ctx context.Context, key, value string, updatedBy uuid.UUID, tx ...*sql.Tx) error {
	var actor any
	if updatedBy != uuid.Nil {
		actor = updatedBy
	}
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO site_settings (key, value, updated_by, updated_at) VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		key, value, actor,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

func (r *settingsDAO) SetMultiple(ctx context.Context, settings map[string]string, updatedBy uuid.UUID, tx ...*sql.Tx) error {
	var actor any
	if updatedBy != uuid.Nil {
		actor = updatedBy
	}
	return db.WithTxOrJoin(ctx, r.db, tx, func(tx *sql.Tx) error {
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

func (r *settingsDAO) Delete(ctx context.Context, key string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM site_settings WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}
	return nil
}
