package admin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

type (
	Service interface {
		GetStats(ctx context.Context) (*dto.AdminStatsResponse, error)

		ListUsers(ctx context.Context, search string, limit, offset int) (*dto.AdminUserListResponse, error)
		GetUser(ctx context.Context, targetID uuid.UUID) (*dto.AdminUserDetailResponse, error)
		SetUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error
		RemoveUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error
		BanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error
		UnbanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		DeleteUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error

		GetSettings(ctx context.Context) (*dto.SettingsResponse, error)
		UpdateSettings(ctx context.Context, actorID uuid.UUID, settings map[string]string) error

		GetAuditLog(ctx context.Context, action string, limit, offset int) (*dto.AuditLogListResponse, error)

		CreateInvite(ctx context.Context, actorID uuid.UUID) (*dto.InviteResponse, error)
		ListInvites(ctx context.Context, limit, offset int) (*dto.InviteListResponse, error)
		DeleteInvite(ctx context.Context, actorID uuid.UUID, code string) error

		ListVanityRoles(ctx context.Context) ([]dto.VanityRoleResponse, error)
		CreateVanityRole(ctx context.Context, actorID uuid.UUID, req dto.CreateVanityRoleRequest) (*dto.VanityRoleResponse, error)
		UpdateVanityRole(ctx context.Context, actorID uuid.UUID, id string, req dto.UpdateVanityRoleRequest) error
		DeleteVanityRole(ctx context.Context, actorID uuid.UUID, id string) error
		GetVanityRoleUsers(ctx context.Context, roleID string, search string, limit, offset int) (*dto.VanityRoleUsersResponse, error)
		AssignVanityRole(ctx context.Context, actorID uuid.UUID, roleID string, userID uuid.UUID) error
		UnassignVanityRole(ctx context.Context, actorID uuid.UUID, roleID string, userID uuid.UUID) error
	}

	service struct {
		userRepo       repository.UserRepository
		roleRepo       repository.RoleRepository
		statsRepo      repository.StatsRepository
		auditRepo      repository.AuditLogRepository
		inviteRepo     repository.InviteRepository
		vanityRoleRepo repository.VanityRoleRepository
		authz          authz.Service
		settingsSvc    settings.Service
		sessionMgr     *session.Manager
		uploadSvc      upload.Service
		hub            *ws.Hub
	}
)

var (
	colorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

	roleRank = map[role.Role]int{
		"":                   0,
		authz.RoleModerator:  1,
		authz.RoleAdmin:      2,
		authz.RoleSuperAdmin: 3,
	}
)

func NewService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	statsRepo repository.StatsRepository,
	auditRepo repository.AuditLogRepository,
	inviteRepo repository.InviteRepository,
	vanityRoleRepo repository.VanityRoleRepository,
	authzService authz.Service,
	settingsSvc settings.Service,
	sessionMgr *session.Manager,
	uploadSvc upload.Service,
	hub *ws.Hub,
) Service {
	return &service{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		statsRepo:      statsRepo,
		auditRepo:      auditRepo,
		inviteRepo:     inviteRepo,
		vanityRoleRepo: vanityRoleRepo,
		authz:          authzService,
		settingsSvc:    settingsSvc,
		sessionMgr:     sessionMgr,
		uploadSvc:      uploadSvc,
		hub:            hub,
	}
}

func (s *service) guardedAction(ctx context.Context, actorID, targetID uuid.UUID, fn func() error) error {
	actorRole, _ := s.authz.GetRole(ctx, actorID)
	targetRole, _ := s.authz.GetRole(ctx, targetID)

	if targetRole == authz.RoleSuperAdmin {
		return ErrProtectedUser
	}

	if roleRank[targetRole] >= roleRank[actorRole] {
		return ErrProtectedUser
	}

	return fn()
}

func (s *service) audit(ctx context.Context, actorID uuid.UUID, action, targetType, targetID string) {
	if err := s.auditRepo.Create(ctx, actorID, action, targetType, targetID, ""); err != nil {
		logger.Log.Error().Err(err).Str("action", action).Msg("failed to write audit log")
	}
}

func (s *service) GetStats(ctx context.Context) (*dto.AdminStatsResponse, error) {
	stats, err := s.statsRepo.GetOverview(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	activeUsers, err := s.statsRepo.GetMostActiveUsers(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("get active users: %w", err)
	}

	mostActive := make([]dto.MostActiveUser, len(activeUsers))
	for i, u := range activeUsers {
		mostActive[i] = dto.MostActiveUser{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			ActionCount: u.ActionCount,
		}
	}

	return &dto.AdminStatsResponse{
		TotalUsers:      stats.TotalUsers,
		TotalTheories:   stats.TotalTheories,
		TotalResponses:  stats.TotalResponses,
		TotalVotes:      stats.TotalVotes,
		TotalPosts:      stats.TotalPosts,
		TotalComments:   stats.TotalComments,
		NewUsers24h:     stats.NewUsers24h,
		NewUsers7d:      stats.NewUsers7d,
		NewUsers30d:     stats.NewUsers30d,
		NewTheories24h:  stats.NewTheories24h,
		NewTheories7d:   stats.NewTheories7d,
		NewTheories30d:  stats.NewTheories30d,
		NewResponses24h: stats.NewResponses24h,
		NewResponses7d:  stats.NewResponses7d,
		NewResponses30d: stats.NewResponses30d,
		NewPosts24h:     stats.NewPosts24h,
		NewPosts7d:      stats.NewPosts7d,
		NewPosts30d:     stats.NewPosts30d,
		PostsByCorner:   stats.PostsByCorner,
		MostActiveUsers: mostActive,
	}, nil
}

func (s *service) ListUsers(ctx context.Context, search string, limit, offset int) (*dto.AdminUserListResponse, error) {
	users, total, err := s.userRepo.ListAll(ctx, search, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]dto.AdminUserItem, len(users))
	for i, u := range users {
		items[i] = dto.AdminUserItem{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        role.Role(u.Role),
			Banned:      u.BannedAt != nil,
			CreatedAt:   u.CreatedAt,
		}
	}

	return &dto.AdminUserListResponse{
		Users:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) GetUser(ctx context.Context, targetID uuid.UUID) (*dto.AdminUserDetailResponse, error) {
	u, stats, err := s.userRepo.GetProfileByID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	resp := &dto.AdminUserDetailResponse{
		AdminUserItem: dto.AdminUserItem{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        role.Role(u.Role),
			Banned:      u.BannedAt != nil,
			CreatedAt:   u.CreatedAt,
		},
		BanReason: u.BanReason,
	}
	if u.IP != nil {
		resp.IP = *u.IP
	}

	if u.BannedAt != nil {
		resp.BannedAt = *u.BannedAt
	}
	if stats != nil {
		resp.TheoryCount = stats.TheoryCount
		resp.ResponseCount = stats.ResponseCount
	}
	resp.MysteryScoreAdjustment = u.MysteryScoreAdjustment
	resp.GMScoreAdjustment = u.GMScoreAdjustment

	detectiveRaw, _ := s.userRepo.GetDetectiveRawScore(ctx, targetID)
	resp.DetectiveScore = detectiveRaw + u.MysteryScoreAdjustment

	gmRaw, _ := s.userRepo.GetGMRawScore(ctx, targetID)
	resp.GMScore = gmRaw + u.GMScoreAdjustment

	return resp, nil
}

func (s *service) SetUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.roleRepo.SetRole(ctx, targetID, r); err != nil {
			return fmt.Errorf("set role: %w", err)
		}
		s.audit(ctx, actorID, "set_role", "user", targetID.String())
		s.broadcastRoleChange(targetID, string(r))
		return nil
	})
}

func (s *service) RemoveUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.roleRepo.RemoveRole(ctx, targetID, r); err != nil {
			return fmt.Errorf("remove role: %w", err)
		}
		s.audit(ctx, actorID, "remove_role", "user", targetID.String())
		s.broadcastRoleChange(targetID, "")
		return nil
	})
}

func (s *service) broadcastRoleChange(userID uuid.UUID, newRole string) {
	s.hub.Broadcast(ws.Message{
		Type: "role_changed",
		Data: map[string]interface{}{
			"user_id": userID,
			"role":    newRole,
		},
	})
}

func (s *service) BanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.BanUser(ctx, targetID, actorID, reason); err != nil {
			return fmt.Errorf("ban user: %w", err)
		}
		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			logger.Log.Error().Err(err).Str("user_id", targetID.String()).Msg("failed to invalidate sessions after ban")
		}
		s.audit(ctx, actorID, "ban_user", "user", targetID.String())
		return nil
	})
}

func (s *service) UnbanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	if err := s.userRepo.UnbanUser(ctx, targetID); err != nil {
		return fmt.Errorf("unban user: %w", err)
	}
	s.audit(ctx, actorID, "unban_user", "user", targetID.String())
	return nil
}

func (s *service) DeleteUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	user, _ := s.userRepo.GetByID(ctx, targetID)

	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.AdminDeleteAccount(ctx, targetID); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		if user != nil {
			_ = s.uploadSvc.Delete(user.AvatarURL)
			_ = s.uploadSvc.Delete(user.BannerURL)
		}
		s.audit(ctx, actorID, "delete_user", "user", targetID.String())
		return nil
	})
}

func (s *service) GetSettings(ctx context.Context) (*dto.SettingsResponse, error) {
	all := s.settingsSvc.GetAll(ctx)
	result := make(map[string]string, len(all))
	for k, v := range all {
		result[string(k)] = v
	}
	return &dto.SettingsResponse{Settings: result}, nil
}

func (s *service) UpdateSettings(ctx context.Context, actorID uuid.UUID, settings map[string]string) error {
	typed := make(map[config.SiteSettingKey]string, len(settings))
	for k, v := range settings {
		typed[config.SiteSettingKey(k)] = v
	}

	if err := s.settingsSvc.SetMultiple(ctx, typed, actorID); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	s.audit(ctx, actorID, "update_settings", "settings", "")
	return nil
}

func (s *service) GetAuditLog(ctx context.Context, action string, limit, offset int) (*dto.AuditLogListResponse, error) {
	entries, total, err := s.auditRepo.List(ctx, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	items := make([]dto.AuditLogEntryResponse, len(entries))
	for i, e := range entries {
		items[i] = dto.AuditLogEntryResponse{
			ID:         e.ID,
			ActorID:    e.ActorID,
			ActorName:  e.ActorName,
			Action:     e.Action,
			TargetType: e.TargetType,
			TargetID:   e.TargetID,
			Details:    e.Details,
			CreatedAt:  e.CreatedAt,
		}
	}

	return &dto.AuditLogListResponse{
		Entries: items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *service) CreateInvite(ctx context.Context, actorID uuid.UUID) (*dto.InviteResponse, error) {
	code := uuid.New().String()[:8]
	if err := s.inviteRepo.Create(ctx, code, actorID); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	s.audit(ctx, actorID, "create_invite", "invite", code)

	return &dto.InviteResponse{
		Code:      code,
		CreatedBy: actorID,
		CreatedAt: "just now",
	}, nil
}

func (s *service) ListInvites(ctx context.Context, limit, offset int) (*dto.InviteListResponse, error) {
	invites, total, err := s.inviteRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}

	items := make([]dto.InviteResponse, len(invites))
	for i, inv := range invites {
		items[i] = dto.InviteResponse{
			Code:      inv.Code,
			CreatedBy: inv.CreatedBy,
			UsedBy:    inv.UsedBy,
			UsedAt:    inv.UsedAt,
			CreatedAt: inv.CreatedAt,
		}
	}

	return &dto.InviteListResponse{
		Invites: items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *service) DeleteInvite(ctx context.Context, actorID uuid.UUID, code string) error {
	if err := s.inviteRepo.Delete(ctx, code); err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	s.audit(ctx, actorID, "delete_invite", "invite", code)
	return nil
}

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
	id := uuid.New().String()
	if err := s.vanityRoleRepo.Create(ctx, id, strings.TrimSpace(req.Label), req.Color, req.SortOrder); err != nil {
		return nil, fmt.Errorf("create vanity role: %w", err)
	}
	s.audit(ctx, actorID, "create_vanity_role", "vanity_role", id)
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
	if strings.TrimSpace(req.Label) == "" {
		return fmt.Errorf("label is required")
	}
	if !colorRegex.MatchString(req.Color) {
		return fmt.Errorf("color must be a valid hex color (e.g. #ff0000)")
	}
	if err := s.vanityRoleRepo.Update(ctx, id, strings.TrimSpace(req.Label), req.Color, req.SortOrder); err != nil {
		return fmt.Errorf("update vanity role: %w", err)
	}
	s.audit(ctx, actorID, "update_vanity_role", "vanity_role", id)
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
	if err := s.vanityRoleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete vanity role: %w", err)
	}
	s.audit(ctx, actorID, "delete_vanity_role", "vanity_role", id)
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) GetVanityRoleUsers(ctx context.Context, roleID string, search string, limit, offset int) (*dto.VanityRoleUsersResponse, error) {
	rows, total, err := s.vanityRoleRepo.GetUsersForRole(ctx, roleID, search, limit, offset)
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
		Limit:  limit,
		Offset: offset,
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
	if err := s.vanityRoleRepo.AssignToUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("assign vanity role: %w", err)
	}
	s.audit(ctx, actorID, "assign_vanity_role", "vanity_role", roleID+":"+userID.String())
	s.broadcastVanityRolesChanged()
	return nil
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
	if err := s.vanityRoleRepo.UnassignFromUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("unassign vanity role: %w", err)
	}
	s.audit(ctx, actorID, "unassign_vanity_role", "vanity_role", roleID+":"+userID.String())
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) broadcastVanityRolesChanged() {
	s.hub.Broadcast(ws.Message{
		Type: "vanity_roles_changed",
		Data: map[string]interface{}{},
	})
}
