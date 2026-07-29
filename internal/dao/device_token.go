package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	deviceTokenDAO struct {
		db *sql.DB
	}
)

func (r *deviceTokenDAO) Upsert(ctx context.Context, userID uuid.UUID, token, platform string) error {
	_, err := r.db.ExecContext(ctx,
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

func (r *deviceTokenDAO) TokensForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT token FROM device_tokens WHERE user_id = $1`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device tokens: %w", err)
	}

	return tokens, nil
}

func (r *deviceTokenDAO) Delete(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE token = $1 AND user_id = $2`, token, userID)
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	return nil
}

func (r *deviceTokenDAO) DeleteMany(ctx context.Context, userID uuid.UUID, tokens []string) error {
	for _, token := range tokens {
		if err := r.Delete(ctx, userID, token); err != nil {
			return err
		}
	}
	return nil
}
