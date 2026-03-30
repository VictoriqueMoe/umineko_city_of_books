package admin

import (
	"context"
	"fmt"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

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
	}

	service struct {
		userRepo    repository.UserRepository
		roleRepo    repository.RoleRepository
		statsRepo   repository.StatsRepository
		auditRepo   repository.AuditLogRepository
		authz       authz.Service
		settingsSvc settings.Service
	}
)

func NewService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	statsRepo repository.StatsRepository,
	auditRepo repository.AuditLogRepository,
	authzService authz.Service,
	settingsSvc settings.Service,
) Service {
	return &service{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		statsRepo:   statsRepo,
		auditRepo:   auditRepo,
		authz:       authzService,
		settingsSvc: settingsSvc,
	}
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
		NewUsers24h:     stats.NewUsers24h,
		NewUsers7d:      stats.NewUsers7d,
		NewUsers30d:     stats.NewUsers30d,
		NewTheories24h:  stats.NewTheories24h,
		NewTheories7d:   stats.NewTheories7d,
		NewTheories30d:  stats.NewTheories30d,
		NewResponses24h: stats.NewResponses24h,
		NewResponses7d:  stats.NewResponses7d,
		NewResponses30d: stats.NewResponses30d,
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
		r, _ := s.authz.GetRole(ctx, u.ID)
		items[i] = dto.AdminUserItem{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        r,
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

	r, _ := s.authz.GetRole(ctx, u.ID)

	resp := &dto.AdminUserDetailResponse{
		AdminUserItem: dto.AdminUserItem{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        r,
			Banned:      u.BannedAt != nil,
			CreatedAt:   u.CreatedAt,
		},
		BanReason: u.BanReason,
	}

	if u.BannedAt != nil {
		resp.BannedAt = *u.BannedAt
	}
	if stats != nil {
		resp.TheoryCount = stats.TheoryCount
		resp.ResponseCount = stats.ResponseCount
	}

	return resp, nil
}

func (s *service) SetUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	if err := s.roleRepo.SetRole(ctx, targetID, r); err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	s.audit(ctx, actorID, "set_role", "user", targetID.String())
	return nil
}

func (s *service) RemoveUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	if err := s.roleRepo.RemoveRole(ctx, targetID, r); err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	s.audit(ctx, actorID, "remove_role", "user", targetID.String())
	return nil
}

func (s *service) BanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error {
	if err := s.userRepo.BanUser(ctx, targetID, actorID, reason); err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	s.audit(ctx, actorID, "ban_user", "user", targetID.String())
	return nil
}

func (s *service) UnbanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	if err := s.userRepo.UnbanUser(ctx, targetID); err != nil {
		return fmt.Errorf("unban user: %w", err)
	}
	s.audit(ctx, actorID, "unban_user", "user", targetID.String())
	return nil
}

func (s *service) DeleteUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	if err := s.userRepo.AdminDeleteAccount(ctx, targetID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	s.audit(ctx, actorID, "delete_user", "user", targetID.String())
	return nil
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
