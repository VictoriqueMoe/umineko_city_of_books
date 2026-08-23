package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/repository"
)

type (
	deviceTokenDAO struct {
		db *sql.DB
	}
)

func (r *deviceTokenDAO) Upsert(ctx context.Context, userID uuid.UUID, token, platform string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO device_tokens (token, user_id, platform, last_seen)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform, last_seen = NOW()`,
		token, userID, platform,
	)
	if err != nil {
		return fmt.Errorf("upsert device token: %w", err)
	}
	return nil
}

func (r *deviceTokenDAO) RegistrationsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]repository.DeviceRegistration, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT token, platform FROM device_tokens WHERE user_id = $1`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var registrations []repository.DeviceRegistration
	for rows.Next() {
		var reg repository.DeviceRegistration
		if err := rows.Scan(&reg.Token, &reg.Platform); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		registrations = append(registrations, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device tokens: %w", err)
	}

	return registrations, nil
}

func (r *deviceTokenDAO) Delete(ctx context.Context, userID uuid.UUID, token string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx, `DELETE FROM device_tokens WHERE token = $1 AND user_id = $2`, token, userID)
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	return nil
}

func (r *deviceTokenDAO) DeleteMany(ctx context.Context, userID uuid.UUID, tokens []string, tx ...*sql.Tx) error {
	for _, token := range tokens {
		if err := r.Delete(ctx, userID, token, tx...); err != nil {
			return err
		}
	}
	return nil
}
