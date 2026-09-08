package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
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

	vanityRoleJoinRow = sqlcgen.VanityRole
)

func toVanityRoleRow(row vanityRoleJoinRow) model.VanityRoleRow {
	return model.VanityRoleRow{
		ID:        row.ID,
		Label:     row.Label,
		Color:     row.Color,
		IsSystem:  row.IsSystem,
		SortOrder: int(row.SortOrder),
	}
}

func (r *vanityRoleDAO) List(ctx context.Context, tx ...*sql.Tx) ([]model.VanityRoleRow, error) {
	rows, err := genQueries(r.db, tx).ListVanityRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list vanity roles: %w", err)
	}

	var result []model.VanityRoleRow
	for _, row := range rows {
		result = append(result, toVanityRoleRow(row))
	}

	return result, nil
}

func (r *vanityRoleDAO) GetByID(ctx context.Context, id string, tx ...*sql.Tx) (*model.VanityRoleRow, error) {
	row, err := genQueries(r.db, tx).GetVanityRoleByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get vanity role: %w", err)
	}

	return new(toVanityRoleRow(row)), nil
}

func (r *vanityRoleDAO) Create(ctx context.Context, s spec.NewVanityRole, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).CreateVanityRole(ctx, sqlcgen.CreateVanityRoleParams{
		ID:        s.ID,
		Label:     s.Label,
		Color:     s.Color,
		SortOrder: int32(s.SortOrder),
	})
	if err != nil {
		return fmt.Errorf("create vanity role: %w", err)
	}

	return nil
}

func (r *vanityRoleDAO) Update(ctx context.Context, s spec.VanityRoleUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateVanityRole(ctx, sqlcgen.UpdateVanityRoleParams{
		Label:     s.Label,
		Color:     s.Color,
		SortOrder: int32(s.SortOrder),
		ID:        s.ID,
	})
	if err != nil {
		return fmt.Errorf("update vanity role: %w", err)
	}

	return nil
}

func (r *vanityRoleDAO) Delete(ctx context.Context, id string, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteVanityRole(ctx, id); err != nil {
		return fmt.Errorf("delete vanity role: %w", err)
	}

	return nil
}

func (r *vanityRoleDAO) AssignToUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).AssignVanityRoleToUser(ctx, sqlcgen.AssignVanityRoleToUserParams{
		UserID:       s.UserID,
		VanityRoleID: s.RoleID,
	})
	if err != nil {
		return fmt.Errorf("assign vanity role: %w", err)
	}

	return nil
}

func (r *vanityRoleDAO) UnassignFromUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UnassignVanityRoleFromUser(ctx, sqlcgen.UnassignVanityRoleFromUserParams{
		UserID:       s.UserID,
		VanityRoleID: s.RoleID,
	})
	if err != nil {
		return fmt.Errorf("unassign vanity role: %w", err)
	}

	return nil
}

func (r *vanityRoleDAO) GetUsersForRole(ctx context.Context, q spec.VanityRoleUserQuery, tx ...*sql.Tx) ([]model.VanityRoleUserRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountVanityRoleUsers(ctx, sqlcgen.CountVanityRoleUsersParams{
		VanityRoleID: q.RoleID,
		Column2:      q.Search,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count vanity role users: %w", err)
	}

	rows, err := queries.ListVanityRoleUsers(ctx, sqlcgen.ListVanityRoleUsersParams{
		VanityRoleID: q.RoleID,
		Column2:      q.Search,
		Limit:        int32(q.Limit),
		Offset:       int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get vanity role users: %w", err)
	}

	var result []model.VanityRoleUserRow
	for _, row := range rows {
		result = append(result, model.VanityRoleUserRow{
			UserID:      row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
		})
	}

	return result, int(total), nil
}

func (r *vanityRoleDAO) GetRolesForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.VanityRoleRow, error) {
	rows, err := genQueries(r.db, tx).GetVanityRolesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get roles for user: %w", err)
	}

	var result []model.VanityRoleRow
	for _, row := range rows {
		result = append(result, toVanityRoleRow(row))
	}

	return result, nil
}

func (r *vanityRoleDAO) GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.VanityRoleRow, error) {
	result := make(map[uuid.UUID][]model.VanityRoleRow)
	if len(userIDs) == 0 {
		return result, nil
	}

	rows, err := genQueries(r.db, tx).GetVanityRolesForUsersBatch(ctx, joinUUIDs(userIDs))
	if err != nil {
		return nil, fmt.Errorf("get roles for users batch: %w", err)
	}

	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], toVanityRoleRow(vanityRoleJoinRow{
			ID:        row.ID,
			Label:     row.Label,
			Color:     row.Color,
			IsSystem:  row.IsSystem,
			SortOrder: row.SortOrder,
		}))
	}

	return result, nil
}

func (r *vanityRoleDAO) GetAllAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	rows, err := genQueries(r.db, tx).GetAllVanityRoleAssignments(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all vanity role assignments: %w", err)
	}

	result := make(map[string][]string)
	for _, row := range rows {
		userID := row.UserID.String()
		result[userID] = append(result[userID], row.VanityRoleID)
	}

	return result, nil
}
