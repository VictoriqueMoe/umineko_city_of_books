package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	RoleDAO interface {
		GetRole(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (role.Role, error)
		GetRoles(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]role.Role, error)
		HasRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) (bool, error)
		SetRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) error
		RemoveRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) error
		GetUsersByRoles(ctx context.Context, roles []role.Role, tx ...*sql.Tx) ([]uuid.UUID, error)
	}

	roleDAO struct {
		db *sql.DB
	}
)

func joinRoles(roles []role.Role) string {
	parts := make([]string, 0, len(roles))
	for _, rl := range roles {
		parts = append(parts, string(rl))
	}

	return strings.Join(parts, ",")
}

func (r *roleDAO) GetRole(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (role.Role, error) {
	result, err := genQueries(r.db, tx).GetUserRole(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get role: %w", err)
	}

	return role.Role(result), nil
}

func (r *roleDAO) GetRoles(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]role.Role, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).GetUserRolesBatch(ctx, joinUUIDs(userIDs))
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}

	out := make(map[uuid.UUID]role.Role, len(userIDs))
	for _, row := range rows {
		out[row.UserID] = role.Role(row.Role)
	}

	return out, nil
}

func (r *roleDAO) HasRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountUserRole(ctx, sqlcgen.CountUserRoleParams{
		UserID: s.UserID,
		Role:   string(s.Role),
	})
	if err != nil {
		return false, fmt.Errorf("check role: %w", err)
	}

	return count > 0, nil
}

func (r *roleDAO) SetRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	if err := queries.ClearUserRoles(ctx, s.UserID); err != nil {
		return fmt.Errorf("clear existing role: %w", err)
	}

	err := queries.InsertUserRole(ctx, sqlcgen.InsertUserRoleParams{
		UserID: s.UserID,
		Role:   string(s.Role),
	})
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}

	return nil
}

func (r *roleDAO) RemoveRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).DeleteUserRole(ctx, sqlcgen.DeleteUserRoleParams{
		UserID: s.UserID,
		Role:   string(s.Role),
	})
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}

	return nil
}

func (r *roleDAO) GetUsersByRoles(ctx context.Context, roles []role.Role, tx ...*sql.Tx) ([]uuid.UUID, error) {
	if len(roles) == 0 {
		return nil, nil
	}

	users, err := genQueries(r.db, tx).GetUsersByRoles(ctx, joinRoles(roles))
	if err != nil {
		return nil, fmt.Errorf("get users by roles: %w", err)
	}

	return users, nil
}
