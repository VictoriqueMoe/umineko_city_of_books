package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type (
	UserSecretRepository interface {
		Unlock(ctx context.Context, userID uuid.UUID, secretID string) error
		ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error)
		GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error)
	}

	userSecretRepository struct {
		db *sql.DB
	}
)

func (r *userSecretRepository) Unlock(ctx context.Context, userID uuid.UUID, secretID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO user_secrets (user_id, secret_id) VALUES (?, ?)`,
		userID, secretID,
	)
	if err != nil {
		return fmt.Errorf("unlock secret: %w", err)
	}
	return nil
}

func (r *userSecretRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT secret_id FROM user_secrets WHERE user_id = ? ORDER BY secret_id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user secrets: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user secret: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *userSecretRepository) GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM user_secrets WHERE secret_id = ?`,
		secretID,
	)
	if err != nil {
		return nil, fmt.Errorf("list secret holders: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan secret holder: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
