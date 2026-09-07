package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	VanityRoleDAO interface {
		List(ctx context.Context, tx ...*sql.Tx) ([]model.VanityRoleRow, error)
		GetByID(ctx context.Context, id string, tx ...*sql.Tx) (*model.VanityRoleRow, error)
		Create(ctx context.Context, s spec.NewVanityRole, tx ...*sql.Tx) error
		Update(ctx context.Context, s spec.VanityRoleUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, id string, tx ...*sql.Tx) error
		AssignToUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error
		UnassignFromUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error
		GetUsersForRole(ctx context.Context, q spec.VanityRoleUserQuery, tx ...*sql.Tx) ([]model.VanityRoleUserRow, int, error)
		GetRolesForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.VanityRoleRow, error)
		GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.VanityRoleRow, error)
		GetAllAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error)
	}

	vanityRoleDAO struct {
		db *sql.DB
	}
)

func (r *vanityRoleDAO) List(ctx context.Context, tx ...*sql.Tx) ([]model.VanityRoleRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT id, label, color, is_system, sort_order FROM vanity_roles ORDER BY sort_order, label`,
	)
	if err != nil {
		return nil, fmt.Errorf("list vanity roles: %w", err)
	}
	defer rows.Close()

	var result []model.VanityRoleRow
	for rows.Next() {
		var row model.VanityRoleRow
		if err := rows.Scan(&row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan vanity role: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *vanityRoleDAO) GetByID(ctx context.Context, id string, tx ...*sql.Tx) (*model.VanityRoleRow, error) {
	var row model.VanityRoleRow
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT id, label, color, is_system, sort_order FROM vanity_roles WHERE id = $1`, id,
	).Scan(&row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get vanity role: %w", err)
	}
	return &row, nil
}

func (r *vanityRoleDAO) Create(ctx context.Context, s spec.NewVanityRole, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO vanity_roles (id, label, color, sort_order) VALUES ($1, $2, $3, $4)`,
		s.ID, s.Label, s.Color, s.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("create vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleDAO) Update(ctx context.Context, s spec.VanityRoleUpdate, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE vanity_roles SET label = $1, color = $2, sort_order = $3 WHERE id = $4`,
		s.Label, s.Color, s.SortOrder, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleDAO) Delete(ctx context.Context, id string, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM vanity_roles WHERE id = $1 AND is_system = FALSE`, id,
	)
	if err != nil {
		return fmt.Errorf("delete vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleDAO) AssignToUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`INSERT INTO user_vanity_roles (user_id, vanity_role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		s.UserID, s.RoleID,
	)
	if err != nil {
		return fmt.Errorf("assign vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleDAO) UnassignFromUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM user_vanity_roles WHERE user_id = $1 AND vanity_role_id = $2`,
		s.UserID, s.RoleID,
	)
	if err != nil {
		return fmt.Errorf("unassign vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleDAO) GetUsersForRole(ctx context.Context, q spec.VanityRoleUserQuery, tx ...*sql.Tx) ([]model.VanityRoleUserRow, int, error) {
	args := []any{q.RoleID}
	where := " WHERE uvr.vanity_role_id = $1"
	if q.Search != "" {
		wc := "%" + q.Search + "%"
		args = append(args, wc, wc)
		where += fmt.Sprintf(" AND (u.username LIKE $%d OR u.display_name LIKE $%d)", len(args)-1, len(args))
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_vanity_roles uvr JOIN users u ON uvr.user_id = u.id`+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vanity role users: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	queryArgs := append(args, q.Limit, q.Offset)
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		fmt.Sprintf(`SELECT u.id, u.username, u.display_name, u.avatar_url
		 FROM user_vanity_roles uvr JOIN users u ON uvr.user_id = u.id`+where+`
		 ORDER BY LOWER(u.display_name)
		 LIMIT $%d OFFSET $%d`, limitIdx, offsetIdx), queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get vanity role users: %w", err)
	}
	defer rows.Close()

	var result []model.VanityRoleUserRow
	for rows.Next() {
		var row model.VanityRoleUserRow
		if err := rows.Scan(&row.UserID, &row.Username, &row.DisplayName, &row.AvatarURL); err != nil {
			return nil, 0, fmt.Errorf("scan vanity role user: %w", err)
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *vanityRoleDAO) GetRolesForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.VanityRoleRow, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT vr.id, vr.label, vr.color, vr.is_system, vr.sort_order
		 FROM vanity_roles vr
		 JOIN user_vanity_roles uvr ON vr.id = uvr.vanity_role_id
		 WHERE uvr.user_id = $1
		 ORDER BY vr.sort_order, vr.label`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get roles for user: %w", err)
	}
	defer rows.Close()

	var result []model.VanityRoleRow
	for rows.Next() {
		var row model.VanityRoleRow
		if err := rows.Scan(&row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan vanity role: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *vanityRoleDAO) GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.VanityRoleRow, error) {
	result := make(map[uuid.UUID][]model.VanityRoleRow)
	if len(userIDs) == 0 {
		return result, nil
	}
	placeholders, args := utils.PlaceholderArgs(userIDs, 1)

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT uvr.user_id, vr.id, vr.label, vr.color, vr.is_system, vr.sort_order
		 FROM user_vanity_roles uvr
		 JOIN vanity_roles vr ON vr.id = uvr.vanity_role_id
		 WHERE uvr.user_id IN (`+strings.Join(placeholders, ",")+`)
		 ORDER BY vr.sort_order, vr.label`, args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get roles for users batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var row model.VanityRoleRow
		if err := rows.Scan(&userID, &row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan batch vanity role: %w", err)
		}
		result[userID] = append(result[userID], row)
	}
	return result, rows.Err()
}

func (r *vanityRoleDAO) GetAllAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`SELECT user_id, vanity_role_id FROM user_vanity_roles ORDER BY user_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("get all vanity role assignments: %w", err)
	}

	return utils.ScanGroups[string, string](rows, "assignment")
}
