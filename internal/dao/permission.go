package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	PermissionDAO interface {
		GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error)
		SetRolePermissions(ctx context.Context, s spec.RolePermissionsUpdate, tx ...*sql.Tx) error
		GetVanityRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error)
		SetVanityRolePermissions(ctx context.Context, s spec.VanityRolePermissionsUpdate, tx ...*sql.Tx) error
		GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	permissionDAO struct {
		db *sql.DB
	}
)

func (r *permissionDAO) GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	rows, err := genQueries(r.db, tx).ListRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}

	groups := make(map[string][]string)
	for _, row := range rows {
		groups[row.Role] = append(groups[row.Role], row.Permission)
	}

	return groups, nil
}

func (r *permissionDAO) SetRolePermissions(ctx context.Context, s spec.RolePermissionsUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		queries := genQueries(r.db, []*sql.Tx{tx})

		if err := queries.DeleteRolePermissions(ctx, s.RoleName); err != nil {
			return fmt.Errorf("clear role permissions: %w", err)
		}

		for _, perm := range s.Permissions {
			err := queries.InsertRolePermission(ctx, sqlcgen.InsertRolePermissionParams{
				Role:       s.RoleName,
				Permission: perm,
			})
			if err != nil {
				return fmt.Errorf("insert role permission: %w", err)
			}
		}

		return nil
	})
}

func (r *permissionDAO) GetVanityRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	rows, err := genQueries(r.db, tx).ListVanityRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get vanity role permissions: %w", err)
	}

	groups := make(map[string][]string)
	for _, row := range rows {
		groups[row.VanityRoleID] = append(groups[row.VanityRoleID], row.Permission)
	}

	return groups, nil
}

func (r *permissionDAO) SetVanityRolePermissions(ctx context.Context, s spec.VanityRolePermissionsUpdate, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		queries := genQueries(r.db, []*sql.Tx{tx})

		if err := queries.DeleteVanityRolePermissions(ctx, s.VanityRoleID); err != nil {
			return fmt.Errorf("clear vanity role permissions: %w", err)
		}

		for _, perm := range s.Permissions {
			err := queries.InsertVanityRolePermission(ctx, sqlcgen.InsertVanityRolePermissionParams{
				VanityRoleID: s.VanityRoleID,
				Permission:   perm,
			})
			if err != nil {
				return fmt.Errorf("insert vanity role permission: %w", err)
			}
		}

		return nil
	})
}

func (r *permissionDAO) GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	result, err := genQueries(r.db, tx).ListUserVanityRoleIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get vanity role ids for user: %w", err)
	}

	return result, nil
}
