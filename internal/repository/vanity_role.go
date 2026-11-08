package repository

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/model/spec"
)

type (
	VanityRoleRepository interface {
		dao.VanityRoleDAO
	}

	vanityRoleRepository struct {
		dao.VanityRoleDAO

		permissions PermissionRepository
	}
)

func ExcludeVanityRoleIDs(ids []string, startIndex int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := utils.PlaceholderArgs(ids, startIndex)

	return " AND id NOT IN (" + strings.Join(placeholders, ", ") + ")", args
}

func NewVanityRoleRepo(d dao.VanityRoleDAO, permissions PermissionRepository) VanityRoleRepository {
	return &vanityRoleRepository{VanityRoleDAO: d, permissions: permissions}
}

func (r *vanityRoleRepository) Create(ctx context.Context, s spec.NewVanityRole, tx ...*sql.Tx) error {
	if err := r.VanityRoleDAO.Create(ctx, s, tx...); err != nil {
		return err
	}

	r.permissions.InvalidateVanityAssignments(ctx)

	return nil
}

func (r *vanityRoleRepository) Update(ctx context.Context, s spec.VanityRoleUpdate, tx ...*sql.Tx) error {
	if err := r.VanityRoleDAO.Update(ctx, s, tx...); err != nil {
		return err
	}

	r.permissions.InvalidateVanityAssignments(ctx)

	return nil
}

func (r *vanityRoleRepository) Delete(ctx context.Context, id string, tx ...*sql.Tx) error {
	assignments, err := r.permissions.GetAllVanityRoleAssignments(ctx, tx...)
	if err != nil {
		return err
	}

	var holders []string
	for userID, roleIDs := range assignments {
		if slices.Contains(roleIDs, id) {
			holders = append(holders, userID)
		}
	}

	if err := r.VanityRoleDAO.Delete(ctx, id, tx...); err != nil {
		return err
	}

	r.permissions.InvalidateVanityAssignments(ctx, holders...)
	r.permissions.InvalidateVanityRolePermissions(ctx)

	return nil
}
