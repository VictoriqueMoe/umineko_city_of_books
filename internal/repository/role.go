package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	RoleRepository interface {
		GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error)
		HasRole(ctx context.Context, userID uuid.UUID, r role.Role) (bool, error)
		SetRole(ctx context.Context, userID uuid.UUID, r role.Role) error
		RemoveRole(ctx context.Context, userID uuid.UUID, r role.Role) error
	}

	roleRepository struct {
		db *sql.DB
	}
)

func (r *roleRepository) GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error) {
	var result string
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM user_roles WHERE user_id = ? LIMIT 1`, userID,
	).Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get role: %w", err)
	}
	return role.Role(result), nil
}

func (r *roleRepository) HasRole(ctx context.Context, userID uuid.UUID, rl role.Role) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = ?`, userID, string(rl),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check role: %w", err)
	}
	return count > 0, nil
}

func (r *roleRepository) SetRole(ctx context.Context, userID uuid.UUID, rl role.Role) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO user_roles (user_id, role) VALUES (?, ?)`, userID, string(rl),
	)
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	return nil
}

func (r *roleRepository) RemoveRole(ctx context.Context, userID uuid.UUID, rl role.Role) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE user_id = ? AND role = ?`, userID, string(rl),
	)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	return nil
}
