package admin

import (
	"context"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) ListVanityRoles(ctx context.Context) ([]dto.VanityRoleResponse, error) {
	rows, err := s.vanityRoleRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list vanity roles: %w", err)
	}
	result := make([]dto.VanityRoleResponse, len(rows))
	for i, r := range rows {
		result[i] = dto.VanityRoleResponse{
			ID:        r.ID,
			Label:     r.Label,
			Color:     r.Color,
			IsSystem:  r.IsSystem,
			SortOrder: r.SortOrder,
		}
	}
	return result, nil
}

func (s *service) CreateVanityRole(ctx context.Context, actorID uuid.UUID, req dto.CreateVanityRoleRequest) (*dto.VanityRoleResponse, error) {
	if strings.TrimSpace(req.Label) == "" {
		return nil, fmt.Errorf("label is required")
	}
	if !colorRegex.MatchString(req.Color) {
		return nil, fmt.Errorf("color must be a valid hex color (e.g. #ff0000)")
	}
	req.SortOrder = max(req.SortOrder, 0)

	id := uuid.New().String()
	newRole := spec.NewVanityRole{
		ID:        id,
		Label:     strings.TrimSpace(req.Label),
		Color:     req.Color,
		SortOrder: req.SortOrder,
	}

	if err := s.vanityRoleRepo.Create(ctx, newRole); err != nil {
		return nil, fmt.Errorf("create vanity role: %w", err)
	}
	s.audit(ctx, actorID, audit.ActionCreateVanityRole, audit.TargetVanityRole, id)
	s.broadcastVanityRolesChanged()
	return &dto.VanityRoleResponse{
		ID:        id,
		Label:     strings.TrimSpace(req.Label),
		Color:     req.Color,
		IsSystem:  false,
		SortOrder: req.SortOrder,
	}, nil
}

func (s *service) UpdateVanityRole(ctx context.Context, actorID uuid.UUID, id string, req dto.UpdateVanityRoleRequest) error {
	existing, err := s.vanityRoleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get vanity role: %w", err)
	}
	if existing == nil {
		return ErrVanityRoleNotFound
	}
	if existing.IsSystem {
		return ErrSystemRole
	}
	if strings.TrimSpace(req.Label) == "" {
		return fmt.Errorf("label is required")
	}
	if !colorRegex.MatchString(req.Color) {
		return fmt.Errorf("color must be a valid hex color (e.g. #ff0000)")
	}
	req.SortOrder = max(req.SortOrder, 0)

	update := spec.VanityRoleUpdate{
		ID:        id,
		Label:     strings.TrimSpace(req.Label),
		Color:     req.Color,
		SortOrder: req.SortOrder,
	}

	if err := s.vanityRoleRepo.Update(ctx, update); err != nil {
		return fmt.Errorf("update vanity role: %w", err)
	}
	s.audit(ctx, actorID, audit.ActionUpdateVanityRole, audit.TargetVanityRole, id)
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) DeleteVanityRole(ctx context.Context, actorID uuid.UUID, id string) error {
	existing, err := s.vanityRoleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get vanity role: %w", err)
	}
	if existing == nil {
		return ErrVanityRoleNotFound
	}
	if existing.IsSystem {
		return ErrSystemRole
	}

	if s.isChatbotOptInRole(ctx, id) {
		return ErrVanityRoleOptInLocked
	}

	if err := s.vanityRoleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete vanity role: %w", err)
	}
	s.audit(ctx, actorID, audit.ActionDeleteVanityRole, audit.TargetVanityRole, id)
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) isChatbotOptInRole(ctx context.Context, id string) bool {
	if !s.settingsSvc.GetBool(ctx, config.SettingChatbotEnabled) {
		return false
	}

	if !s.settingsSvc.GetBool(ctx, config.SettingChatbotRequirePermission) {
		return false
	}

	return strings.TrimSpace(s.settingsSvc.Get(ctx, config.SettingChatbotOptInRole)) == id
}

func (s *service) GetVanityRoleUsers(ctx context.Context, roleID string, search string, page bounds.Page) (*dto.VanityRoleUsersResponse, error) {
	rows, total, err := s.vanityRoleRepo.GetUsersForRole(ctx, spec.VanityRoleUserQuery{
		RoleID: roleID,
		Search: search,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	})
	if err != nil {
		return nil, fmt.Errorf("get vanity role users: %w", err)
	}
	users := make([]dto.VanityRoleUserItem, len(rows))
	for i, r := range rows {
		users[i] = dto.VanityRoleUserItem{
			ID:          r.UserID,
			Username:    r.Username,
			DisplayName: r.DisplayName,
			AvatarURL:   r.AvatarURL,
		}
	}
	return &dto.VanityRoleUsersResponse{
		Users:  users,
		Total:  total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}, nil
}

func (s *service) AssignVanityRole(ctx context.Context, actorID uuid.UUID, roleID string, userID uuid.UUID) error {
	existing, err := s.vanityRoleRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("get vanity role: %w", err)
	}
	if existing == nil {
		return ErrVanityRoleNotFound
	}
	if existing.IsSystem {
		return ErrSystemRole
	}

	if err := s.guardPermissionCarryingRole(ctx, actorID, userID, roleID); err != nil {
		return err
	}

	if err := s.vanityRoleRepo.AssignToUser(ctx, spec.VanityRoleAssignment{UserID: userID, RoleID: roleID}); err != nil {
		return fmt.Errorf("assign vanity role: %w", err)
	}
	s.auditSubject(ctx, actorID, audit.ActionAssignVanityRole, audit.TargetVanityRole, roleID, userID)
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) guardPermissionCarryingRole(ctx context.Context, actorID, targetID uuid.UUID, roleID string) error {
	carries, err := s.vanityRoleCarriesPermissions(ctx, roleID)
	if err != nil {
		return err
	}

	if !carries {
		return nil
	}

	return s.guardedAction(ctx, actorID, targetID, func() error { return nil })
}

func (s *service) UnassignVanityRole(ctx context.Context, actorID uuid.UUID, roleID string, userID uuid.UUID) error {
	existing, err := s.vanityRoleRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("get vanity role: %w", err)
	}
	if existing == nil {
		return ErrVanityRoleNotFound
	}
	if existing.IsSystem {
		return ErrSystemRole
	}

	if err := s.guardPermissionCarryingRole(ctx, actorID, userID, roleID); err != nil {
		return err
	}

	if err := s.vanityRoleRepo.UnassignFromUser(ctx, spec.VanityRoleAssignment{UserID: userID, RoleID: roleID}); err != nil {
		return fmt.Errorf("unassign vanity role: %w", err)
	}
	s.auditSubject(ctx, actorID, audit.ActionUnassignVanityRole, audit.TargetVanityRole, roleID, userID)
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) broadcastVanityRolesChanged() {
	s.hub.Broadcast(ws.Message{
		Type: "vanity_roles_changed",
		Data: map[string]any{},
	})
}
