package authz

import (
	"context"
	"slices"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

type (
	Service interface {
		Can(ctx context.Context, userID uuid.UUID, perm Permission) bool
		EffectivePermissions(ctx context.Context, userID uuid.UUID) []Permission
		GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error)
		GetRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]role.Role, error)
		IsBanned(ctx context.Context, userID uuid.UUID) bool
		IsLocked(ctx context.Context, userID uuid.UUID) bool
		RequiresEmailVerification(ctx context.Context, userID uuid.UUID) bool
		IsRestrictedNewAccount(ctx context.Context, userID uuid.UUID) bool
	}

	service struct {
		roleRepo    repository.RoleRepository
		userRepo    repository.UserRepository
		permRepo    repository.PermissionRepository
		settingsSvc settings.Service
	}
)

func NewService(roleRepo repository.RoleRepository, userRepo repository.UserRepository, permRepo repository.PermissionRepository, settingsSvc settings.Service) Service {
	return &service{roleRepo: roleRepo, userRepo: userRepo, permRepo: permRepo, settingsSvc: settingsSvc}
}

func (s *service) IsRestrictedNewAccount(ctx context.Context, userID uuid.UUID) bool {
	hours := s.settingsSvc.GetInt(ctx, config.SettingNewAccountHours)
	if hours <= 0 {
		return false
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return false
	}

	return !role.Role(user.Role).IsSiteStaff() && user.IsNewAccount(hours)
}

func (s *service) IsBanned(ctx context.Context, userID uuid.UUID) bool {
	banned, err := s.userRepo.IsBanned(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to check ban status, treating the account as restricted")
		return true
	}
	return banned
}

func (s *service) IsLocked(ctx context.Context, userID uuid.UUID) bool {
	locked, err := s.userRepo.IsLocked(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to check lock status, treating the account as restricted")
		return true
	}
	return locked
}

func (s *service) RequiresEmailVerification(ctx context.Context, userID uuid.UUID) bool {
	blocked, err := s.userRepo.RequiresEmailVerification(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to check email verification status, treating the account as restricted")
		return true
	}
	return blocked
}

func (s *service) Can(ctx context.Context, userID uuid.UUID, perm Permission) bool {
	if userID == uuid.Nil {
		return false
	}

	r, err := s.roleRepo.GetRole(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to get role for permission check")
		return false
	}

	if IsImmutableRole(r) {
		return true
	}

	if s.systemRoleGrants(ctx, r, perm) {
		return true
	}

	if !IsVanityAssignable(perm) {
		return false
	}

	return s.vanityRolesGrant(ctx, userID, perm)
}

func (s *service) EffectivePermissions(ctx context.Context, userID uuid.UUID) []Permission {
	if userID == uuid.Nil {
		return nil
	}

	r, err := s.roleRepo.GetRole(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to get role for effective permissions")
		return nil
	}

	if IsImmutableRole(r) {
		granted := make([]Permission, 0, len(permissionCatalogue))
		for _, def := range permissionCatalogue {
			granted = append(granted, def.Permission)
		}

		return granted
	}

	fromRole := make(map[string]struct{})
	fromVanity := make(map[string]struct{})

	if IsEditableSystemRole(r) {
		table, err := s.permRepo.GetRolePermissions(ctx)
		if err != nil {
			logger.Log.Error().Err(err).Msg("failed to load role permissions")
			return nil
		}

		for _, perm := range table[string(r)] {
			fromRole[perm] = struct{}{}
		}
	}

	ids, err := s.permRepo.GetVanityRoleIDsForUser(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to load vanity roles for effective permissions")
		ids = nil
	}

	if len(ids) > 0 {
		table, err := s.permRepo.GetVanityRolePermissions(ctx)
		if err != nil {
			logger.Log.Error().Err(err).Msg("failed to load vanity role permissions")
			table = nil
		}

		for _, id := range ids {
			for _, perm := range table[id] {
				fromVanity[perm] = struct{}{}
			}
		}
	}

	var result []Permission
	for _, def := range permissionCatalogue {
		if def.Scope == ScopeRestricted {
			continue
		}

		if _, ok := fromRole[string(def.Permission)]; ok {
			result = append(result, def.Permission)
			continue
		}

		if def.Scope != ScopeGeneral {
			continue
		}

		if _, ok := fromVanity[string(def.Permission)]; ok {
			result = append(result, def.Permission)
		}
	}

	return result
}

func (s *service) systemRoleGrants(ctx context.Context, r role.Role, perm Permission) bool {
	if !IsEditableSystemRole(r) {
		return false
	}

	if !IsRoleAssignable(perm) {
		return false
	}

	table, err := s.permRepo.GetRolePermissions(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to load role permissions")
		return false
	}

	return slices.Contains(table[string(r)], string(perm))
}

func (s *service) vanityRolesGrant(ctx context.Context, userID uuid.UUID, perm Permission) bool {
	ids, err := s.permRepo.GetVanityRoleIDsForUser(ctx, userID)
	if err != nil {
		logger.Log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to load vanity roles for permission check")
		return false
	}

	if len(ids) == 0 {
		return false
	}

	table, err := s.permRepo.GetVanityRolePermissions(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to load vanity role permissions")
		return false
	}

	for _, id := range ids {
		if slices.Contains(table[id], string(perm)) {
			return true
		}
	}

	return false
}

func (s *service) GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error) {
	return s.roleRepo.GetRole(ctx, userID)
}

func (s *service) GetRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]role.Role, error) {
	return s.roleRepo.GetRoles(ctx, userIDs)
}
