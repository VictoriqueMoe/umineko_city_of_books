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
		GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string) ([]uuid.UUID, error)
		IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error)
		DeleteSecrets(ctx context.Context, secretIDs []string) error
	}

	userSecretRepository struct {
		db *sql.DB
	}
)

func (r *userSecretRepository) Unlock(ctx context.Context, userID uuid.UUID, secretID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_secrets (user_id, secret_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, secretID,
	)
	if err != nil {
		return fmt.Errorf("unlock secret: %w", err)
	}
	return nil
}

func (r *userSecretRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT secret_id FROM user_secrets WHERE user_id = $1 ORDER BY secret_id`,
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

func (r *userSecretRepository) GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string) ([]uuid.UUID, error) {
	if len(pieceIDs) == 0 {
		return nil, nil
	}
	placeholders := "$1"
	args := []interface{}{pieceIDs[0]}
	for i := 1; i < len(pieceIDs); i++ {
		args = append(args, pieceIDs[i])
		placeholders += fmt.Sprintf(",$%d", len(args))
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT user_id FROM user_secrets WHERE secret_id IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list piece participants: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan participant id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *userSecretRepository) IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM user_secrets WHERE secret_id = $1 LIMIT 1`,
		secretID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check secret solved: %w", err)
	}
	return true, nil
}

func (r *userSecretRepository) DeleteSecrets(ctx context.Context, secretIDs []string) error {
	if len(secretIDs) == 0 {
		return nil
	}
	placeholders := "$1"
	args := []interface{}{secretIDs[0]}
	for i := 1; i < len(secretIDs); i++ {
		args = append(args, secretIDs[i])
		placeholders += fmt.Sprintf(",$%d", len(args))
	}
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_secrets WHERE secret_id IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return fmt.Errorf("delete secrets: %w", err)
	}
	return nil
}

func (r *userSecretRepository) GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM user_secrets WHERE secret_id = $1`,
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
