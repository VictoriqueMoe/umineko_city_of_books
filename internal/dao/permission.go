package dao

import (
	"context"
	"database/sql"
	"fmt"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	permissionDAO struct {
		db *sql.DB
	}
)

func (r *permissionDAO) GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT role, permission FROM role_permissions ORDER BY role, permission`,
	)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}

	return utils.ScanGroups[string, string](rows, "role permission")
}

func (r *permissionDAO) SetRolePermissions(ctx context.Context, roleName string, perms []string, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role = $1`, roleName); err != nil {
			return fmt.Errorf("clear role permissions: %w", err)
		}

		for _, perm := range perms {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO role_permissions (role, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				roleName, perm,
			); err != nil {
				return fmt.Errorf("insert role permission: %w", err)
			}
		}

		return nil
	})
}

func (r *permissionDAO) GetVanityRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT vrp.vanity_role_id, vrp.permission
		 FROM vanity_role_permissions vrp
		 JOIN vanity_roles vr ON vr.id = vrp.vanity_role_id
		 WHERE vr.is_system = FALSE
		 ORDER BY vrp.vanity_role_id, vrp.permission`,
	)
	if err != nil {
		return nil, fmt.Errorf("get vanity role permissions: %w", err)
	}

	return utils.ScanGroups[string, string](rows, "vanity role permission")
}

func (r *permissionDAO) SetVanityRolePermissions(ctx context.Context, vanityRoleID string, perms []string, tx ...*sql.Tx) error {
	return db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM vanity_role_permissions WHERE vanity_role_id = $1`, vanityRoleID); err != nil {
			return fmt.Errorf("clear vanity role permissions: %w", err)
		}

		for _, perm := range perms {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO vanity_role_permissions (vanity_role_id, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				vanityRoleID, perm,
			); err != nil {
				return fmt.Errorf("insert vanity role permission: %w", err)
			}
		}

		return nil
	})
}

func (r *permissionDAO) GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT vanity_role_id FROM user_vanity_roles WHERE user_id = $1 ORDER BY vanity_role_id`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get vanity role ids for user: %w", err)
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan vanity role id: %w", err)
		}

		result = append(result, id)
	}

	return result, rows.Err()
}
