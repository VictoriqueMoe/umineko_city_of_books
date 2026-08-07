package admin

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

var (
	editableRoleLabels = map[role.Role]string{
		authz.RoleModerator: "Moderator",
	}
)

func (s *service) GetPermissionSettings(ctx context.Context) (*dto.PermissionSettingsResponse, error) {
	catalogue := authz.RoleAssignablePermissions()
	permissions := make([]dto.PermissionCatalogueItem, len(catalogue))
	for i, def := range catalogue {
		permissions[i] = dto.PermissionCatalogueItem{
			Permission:       string(def.Permission),
			Label:            def.Label,
			VanityAssignable: def.Scope == authz.ScopeGeneral,
		}
	}

	rolePerms, err := s.permissionRepo.GetRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}

	editable := authz.EditableSystemRoles()
	roles := make([]dto.RolePermissionsItem, len(editable))
	for i, r := range editable {
		roles[i] = dto.RolePermissionsItem{
			Role:        string(r),
			Label:       editableRoleLabels[r],
			Permissions: sortedPermissions(rolePerms[string(r)]),
		}
	}

	vanityPerms, err := s.permissionRepo.GetVanityRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get vanity role permissions: %w", err)
	}

	rows, err := s.vanityRoleRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list vanity roles: %w", err)
	}

	vanityRoles := make([]dto.VanityRolePermissionsItem, 0, len(rows))
	for _, row := range rows {
		if row.IsSystem {
			continue
		}

		vanityRoles = append(vanityRoles, dto.VanityRolePermissionsItem{
			ID:          row.ID,
			Label:       row.Label,
			Color:       row.Color,
			SortOrder:   row.SortOrder,
			Permissions: sortedPermissions(vanityPerms[row.ID]),
		})
	}

	return &dto.PermissionSettingsResponse{
		Permissions: permissions,
		Roles:       roles,
		VanityRoles: vanityRoles,
	}, nil
}

func (s *service) UpdateRolePermissions(ctx context.Context, actorID uuid.UUID, roleName string, perms []string) error {
	target := role.Role(strings.TrimSpace(roleName))

	if authz.IsImmutableRole(target) {
		return ErrImmutableRole
	}

	if !authz.IsEditableSystemRole(target) {
		return ErrUnknownRole
	}

	clean, err := normalisePermissions(perms, false)
	if err != nil {
		return err
	}

	if err := s.permissionRepo.SetRolePermissions(ctx, string(target), clean); err != nil {
		return fmt.Errorf("set role permissions: %w", err)
	}

	s.auditDetails(ctx, actorID, "update_role_permissions", "role", string(target), strings.Join(clean, ","))
	s.broadcastPermissionsChanged()

	return nil
}

func (s *service) UpdateVanityRolePermissions(ctx context.Context, actorID uuid.UUID, vanityRoleID string, perms []string) error {
	existing, err := s.vanityRoleRepo.GetByID(ctx, vanityRoleID)
	if err != nil {
		return fmt.Errorf("get vanity role: %w", err)
	}

	if existing == nil {
		return ErrVanityRoleNotFound
	}

	if existing.IsSystem {
		return ErrSystemRole
	}

	clean, err := normalisePermissions(perms, true)
	if err != nil {
		return err
	}

	if err := s.permissionRepo.SetVanityRolePermissions(ctx, vanityRoleID, clean); err != nil {
		return fmt.Errorf("set vanity role permissions: %w", err)
	}

	s.auditDetails(ctx, actorID, "update_vanity_role_permissions", "vanity_role", vanityRoleID, strings.Join(clean, ","))
	s.broadcastPermissionsChanged()

	return nil
}

func (s *service) vanityRoleCarriesPermissions(ctx context.Context, vanityRoleID string) (bool, error) {
	table, err := s.permissionRepo.GetVanityRolePermissions(ctx)
	if err != nil {
		return false, fmt.Errorf("get vanity role permissions: %w", err)
	}

	return len(table[vanityRoleID]) > 0, nil
}

func normalisePermissions(perms []string, vanityOnly bool) ([]string, error) {
	seen := make(map[string]struct{}, len(perms))
	clean := make([]string, 0, len(perms))

	for _, raw := range perms {
		perm := authz.Permission(strings.TrimSpace(raw))

		if perm == authz.PermAll || !authz.IsKnownPermission(perm) {
			return nil, fmt.Errorf("%w: %q", ErrUnknownPermission, raw)
		}

		if !authz.IsRoleAssignable(perm) {
			return nil, fmt.Errorf("%w: %q", ErrRestrictedPermission, raw)
		}

		if vanityOnly && !authz.IsVanityAssignable(perm) {
			return nil, fmt.Errorf("%w: %q", ErrStaffPermission, raw)
		}

		if _, dupe := seen[string(perm)]; dupe {
			continue
		}

		seen[string(perm)] = struct{}{}
		clean = append(clean, string(perm))
	}

	slices.Sort(clean)

	return clean, nil
}

func sortedPermissions(perms []string) []string {
	result := slices.Clone(perms)
	slices.Sort(result)

	if result == nil {
		return []string{}
	}

	return result
}

func (s *service) broadcastPermissionsChanged() {
	s.hub.Broadcast(ws.Message{
		Type: "permissions_changed",
		Data: map[string]any{},
	})
}
