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
		GetUsersByRoles(ctx context.Context, roles []role.Role) ([]uuid.UUID, error)
	}

	roleRepository struct {
		db *sql.DB
	}
)

func (r *roleRepository) GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error) {
	var result string
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM user_roles WHERE user_id = $1 LIMIT 1`, userID,
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
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1 AND role = $2`, userID, string(rl),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check role: %w", err)
	}
	return count > 0, nil
}

func (r *roleRepository) SetRole(ctx context.Context, userID uuid.UUID, rl role.Role) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE user_id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("clear existing role: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, userID, string(rl),
	)
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	return nil
}

func (r *roleRepository) RemoveRole(ctx context.Context, userID uuid.UUID, rl role.Role) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role = $2`, userID, string(rl),
	)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	return nil
}

func (r *roleRepository) GetUsersByRoles(ctx context.Context, roles []role.Role) ([]uuid.UUID, error) {
	if len(roles) == 0 {
		return nil, nil
	}
	placeholders := "$1"
	args := []interface{}{string(roles[0])}
	for i := 1; i < len(roles); i++ {
		args = append(args, string(roles[i]))
		placeholders += fmt.Sprintf(", $%d", len(args))
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT user_id FROM user_roles WHERE role IN (`+placeholders+`)`, args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get users by roles: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, rows.Err()
}
