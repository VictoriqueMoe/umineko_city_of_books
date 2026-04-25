package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type (
	VanityRoleRepository interface {
		List(ctx context.Context) ([]VanityRoleRow, error)
		GetByID(ctx context.Context, id string) (*VanityRoleRow, error)
		Create(ctx context.Context, id, label, color string, sortOrder int) error
		Update(ctx context.Context, id, label, color string, sortOrder int) error
		Delete(ctx context.Context, id string) error
		AssignToUser(ctx context.Context, userID uuid.UUID, roleID string) error
		UnassignFromUser(ctx context.Context, userID uuid.UUID, roleID string) error
		GetUsersForRole(ctx context.Context, roleID string, search string, limit, offset int) ([]VanityRoleUserRow, int, error)
		GetRolesForUser(ctx context.Context, userID uuid.UUID) ([]VanityRoleRow, error)
		GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]VanityRoleRow, error)
		GetAllAssignments(ctx context.Context) (map[string][]string, error)
	}

	vanityRoleRepository struct {
		db *sql.DB
	}

	VanityRoleRow struct {
		ID        string
		Label     string
		Color     string
		IsSystem  bool
		SortOrder int
	}

	VanityRoleUserRow struct {
		UserID      uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
	}
)

func (r *vanityRoleRepository) List(ctx context.Context) ([]VanityRoleRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, label, color, is_system, sort_order FROM vanity_roles ORDER BY sort_order, label`,
	)
	if err != nil {
		return nil, fmt.Errorf("list vanity roles: %w", err)
	}
	defer rows.Close()

	var result []VanityRoleRow
	for rows.Next() {
		var row VanityRoleRow
		if err := rows.Scan(&row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan vanity role: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *vanityRoleRepository) GetByID(ctx context.Context, id string) (*VanityRoleRow, error) {
	var row VanityRoleRow
	err := r.db.QueryRowContext(ctx,
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

func (r *vanityRoleRepository) Create(ctx context.Context, id, label, color string, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO vanity_roles (id, label, color, sort_order) VALUES ($1, $2, $3, $4)`,
		id, label, color, sortOrder,
	)
	if err != nil {
		return fmt.Errorf("create vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleRepository) Update(ctx context.Context, id, label, color string, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vanity_roles SET label = $1, color = $2, sort_order = $3 WHERE id = $4`,
		label, color, sortOrder, id,
	)
	if err != nil {
		return fmt.Errorf("update vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM vanity_roles WHERE id = $1 AND is_system = FALSE`, id,
	)
	if err != nil {
		return fmt.Errorf("delete vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleRepository) AssignToUser(ctx context.Context, userID uuid.UUID, roleID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_vanity_roles (user_id, vanity_role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, roleID,
	)
	if err != nil {
		return fmt.Errorf("assign vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleRepository) UnassignFromUser(ctx context.Context, userID uuid.UUID, roleID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_vanity_roles WHERE user_id = $1 AND vanity_role_id = $2`,
		userID, roleID,
	)
	if err != nil {
		return fmt.Errorf("unassign vanity role: %w", err)
	}
	return nil
}

func (r *vanityRoleRepository) GetUsersForRole(ctx context.Context, roleID string, search string, limit, offset int) ([]VanityRoleUserRow, int, error) {
	args := []interface{}{roleID}
	where := " WHERE uvr.vanity_role_id = $1"
	if search != "" {
		wc := "%" + search + "%"
		args = append(args, wc, wc)
		where += fmt.Sprintf(" AND (u.username LIKE $%d OR u.display_name LIKE $%d)", len(args)-1, len(args))
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_vanity_roles uvr JOIN users u ON uvr.user_id = u.id`+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vanity role users: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT u.id, u.username, u.display_name, u.avatar_url
		 FROM user_vanity_roles uvr JOIN users u ON uvr.user_id = u.id`+where+`
		 ORDER BY LOWER(u.display_name)
		 LIMIT $%d OFFSET $%d`, limitIdx, offsetIdx), queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get vanity role users: %w", err)
	}
	defer rows.Close()

	var result []VanityRoleUserRow
	for rows.Next() {
		var row VanityRoleUserRow
		if err := rows.Scan(&row.UserID, &row.Username, &row.DisplayName, &row.AvatarURL); err != nil {
			return nil, 0, fmt.Errorf("scan vanity role user: %w", err)
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *vanityRoleRepository) GetRolesForUser(ctx context.Context, userID uuid.UUID) ([]VanityRoleRow, error) {
	rows, err := r.db.QueryContext(ctx,
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

	var result []VanityRoleRow
	for rows.Next() {
		var row VanityRoleRow
		if err := rows.Scan(&row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan vanity role: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *vanityRoleRepository) GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]VanityRoleRow, error) {
	result := make(map[uuid.UUID][]VanityRoleRow)
	if len(userIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i := range userIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = userIDs[i]
	}
	rows, err := r.db.QueryContext(ctx,
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
		var row VanityRoleRow
		if err := rows.Scan(&userID, &row.ID, &row.Label, &row.Color, &row.IsSystem, &row.SortOrder); err != nil {
			return nil, fmt.Errorf("scan batch vanity role: %w", err)
		}
		result[userID] = append(result[userID], row)
	}
	return result, rows.Err()
}

func (r *vanityRoleRepository) GetAllAssignments(ctx context.Context) (map[string][]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, vanity_role_id FROM user_vanity_roles ORDER BY user_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("get all vanity role assignments: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var userID, roleID string
		if err := rows.Scan(&userID, &roleID); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		result[userID] = append(result[userID], roleID)
	}
	return result, rows.Err()
}

func ExcludeVanityRoleIDs(ids []string, startIndex int) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
		args[i] = id
	}
	return " AND id NOT IN (" + strings.Join(placeholders, ", ") + ")", args
}
