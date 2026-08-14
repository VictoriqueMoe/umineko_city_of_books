package admin

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/giphy/banlist"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	userpkg "umineko_city_of_books/internal/user"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	SystemRoomSync interface {
		EnsureSystemRooms(ctx context.Context) error
		SyncSystemRoomMembership(ctx context.Context, userID uuid.UUID, newRole role.Role) error
	}

	Service interface {
		GetStats(ctx context.Context) (*dto.AdminStatsResponse, error)

		ListUsers(ctx context.Context, search string, page bounds.Page) (*dto.AdminUserListResponse, error)
		GetUser(ctx context.Context, targetID uuid.UUID) (*dto.AdminUserDetailResponse, error)
		SetUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error
		RemoveUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error
		BanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error
		UnbanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		LockUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error
		UnlockUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		DeleteUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		ResetUserPassword(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (string, error)
		SetUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, email string) error
		VerifyUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		UnverifyUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		SetUserDisplayName(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, displayName string) error
		SetDisplayNameLocked(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, locked bool) error
		ForceLogout(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		ListAccountsOnIP(ctx context.Context, targetID uuid.UUID) (*dto.AdminIPMatchesResponse, error)
		GetUserAuditLog(ctx context.Context, targetID uuid.UUID, page bounds.Page) (*dto.AuditLogListResponse, error)

		GetSettings(ctx context.Context) (*dto.SettingsResponse, error)
		UpdateSettings(ctx context.Context, actorID uuid.UUID, settings map[string]string) error
		SendTestEmail(ctx context.Context, actorID uuid.UUID) error

		GetAuditLog(ctx context.Context, action string, page bounds.Page) (*dto.AuditLogListResponse, error)

		CreateInvite(ctx context.Context, actorID uuid.UUID) (*dto.InviteResponse, error)
		ListInvites(ctx context.Context, page bounds.Page) (*dto.InviteListResponse, error)
		DeleteInvite(ctx context.Context, actorID uuid.UUID, code string) error

		ListVanityRoles(ctx context.Context) ([]dto.VanityRoleResponse, error)
		CreateVanityRole(ctx context.Context, actorID uuid.UUID, req dto.CreateVanityRoleRequest) (*dto.VanityRoleResponse, error)
		UpdateVanityRole(ctx context.Context, actorID uuid.UUID, id string, req dto.UpdateVanityRoleRequest) error
		DeleteVanityRole(ctx context.Context, actorID uuid.UUID, id string) error
		GetVanityRoleUsers(ctx context.Context, roleID string, search string, page bounds.Page) (*dto.VanityRoleUsersResponse, error)
		AssignVanityRole(ctx context.Context, actorID uuid.UUID, roleID string, userID uuid.UUID) error
		UnassignVanityRole(ctx context.Context, actorID uuid.UUID, roleID string, userID uuid.UUID) error

		GetPermissionSettings(ctx context.Context) (*dto.PermissionSettingsResponse, error)
		UpdateRolePermissions(ctx context.Context, actorID uuid.UUID, roleName string, perms []string) error
		UpdateVanityRolePermissions(ctx context.Context, actorID uuid.UUID, vanityRoleID string, perms []string) error

		ListBannedGifs(ctx context.Context) (*dto.BannedGiphyListResponse, error)
		AddBannedGif(ctx context.Context, actorID uuid.UUID, req dto.AddBannedGiphyRequest) (*dto.AddBannedGiphyResponse, error)
		RemoveBannedGif(ctx context.Context, actorID uuid.UUID, kind, value string) error
	}

	service struct {
		userRepo       repository.UserRepository
		roleRepo       repository.RoleRepository
		statsRepo      repository.StatsRepository
		auditRepo      repository.AuditLogRepository
		inviteRepo     repository.InviteRepository
		vanityRoleRepo repository.VanityRoleRepository
		permissionRepo repository.PermissionRepository
		giphyBanlist   banlist.Service
		authz          authz.Service
		settingsSvc    settings.Service
		sessionMgr     *session.Manager
		uploadSvc      upload.Service
		hub            *ws.Hub
		chatSync       SystemRoomSync
		emailSvc       email.Service
		authSvc        auth.Service
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
	permissionRepo repository.PermissionRepository,
	giphyBanlist banlist.Service,
	authzService authz.Service,
	settingsSvc settings.Service,
	sessionMgr *session.Manager,
	uploadSvc upload.Service,
	hub *ws.Hub,
	chatSync SystemRoomSync,
	emailSvc email.Service,
	authSvc auth.Service,
) Service {
	return &service{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		statsRepo:      statsRepo,
		auditRepo:      auditRepo,
		inviteRepo:     inviteRepo,
		vanityRoleRepo: vanityRoleRepo,
		permissionRepo: permissionRepo,
		giphyBanlist:   giphyBanlist,
		authz:          authzService,
		settingsSvc:    settingsSvc,
		sessionMgr:     sessionMgr,
		uploadSvc:      uploadSvc,
		hub:            hub,
		chatSync:       chatSync,
		emailSvc:       emailSvc,
		authSvc:        authSvc,
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

func (s *service) rejectBotTarget(ctx context.Context, targetID uuid.UUID) error {
	usr, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if usr != nil && usr.IsBot {
		return ErrBotAccountProtected
	}

	return nil
}

func (s *service) audit(ctx context.Context, actorID uuid.UUID, action, targetType, targetID string) {
	s.auditDetails(ctx, actorID, action, targetType, targetID, "")
}

func (s *service) auditDetails(ctx context.Context, actorID uuid.UUID, action, targetType, targetID, details string) {
	if err := s.auditRepo.Create(ctx, repository.NewAuditEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    details,
	}); err != nil {
		logger.Log.Error().Err(err).Str("action", action).Msg("failed to write audit log")
	}
}

func (s *service) auditSubject(ctx context.Context, actorID uuid.UUID, action, targetType, targetID string, subjectID uuid.UUID) {
	if err := s.auditRepo.CreateForSubject(ctx, repository.NewAuditSubjectEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		SubjectID:  subjectID,
	}); err != nil {
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

func (s *service) ListUsers(ctx context.Context, search string, page bounds.Page) (*dto.AdminUserListResponse, error) {
	users, total, err := s.userRepo.ListAll(ctx, search, page.Limit(), page.Offset())
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
			Locked:      u.LockedAt != nil,
			CreatedAt:   u.CreatedAt,
		}
	}

	return &dto.AdminUserListResponse{
		Users:  items,
		Total:  total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
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
			Locked:      u.LockedAt != nil,
			CreatedAt:   u.CreatedAt,
		},
		Email:             u.Email,
		EmailVerified:     u.EmailVerified,
		DisplayNameLocked: u.DisplayNameLocked,
		BanReason:         u.BanReason,
		LockReason:        u.LockReason,
	}
	if u.IP != nil {
		resp.IP = *u.IP
	}

	if u.BannedAt != nil {
		resp.BannedAt = *u.BannedAt
	}
	if u.BannedBy != nil {
		banner, err := s.userRepo.GetByID(ctx, *u.BannedBy)
		if err != nil {
			return nil, fmt.Errorf("get banned_by user: %w", err)
		}
		if banner != nil {
			resp.BannedBy = banner.ToResponse()
		}
	}
	if u.LockedAt != nil {
		resp.LockedAt = *u.LockedAt
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

func (s *service) assertCanGrantRole(ctx context.Context, actorID uuid.UUID, granted role.Role) error {
	grantedRank, known := roleRank[granted]
	if !known || granted == "" {
		return ErrUnknownRole
	}

	actorRole, err := s.authz.GetRole(ctx, actorID)
	if err != nil {
		return fmt.Errorf("get actor role: %w", err)
	}

	if grantedRank >= roleRank[actorRole] {
		return ErrRoleOutranksActor
	}

	return nil
}

func (s *service) SetUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	if err := s.assertCanGrantRole(ctx, actorID, r); err != nil {
		return err
	}

	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.roleRepo.SetRole(ctx, targetID, r); err != nil {
			return fmt.Errorf("set role: %w", err)
		}
		s.auditDetails(ctx, actorID, "set_role", "user", targetID.String(), string(r))
		if s.chatSync != nil {
			if err := s.chatSync.EnsureSystemRooms(ctx); err != nil {
				logger.Log.Error().Err(err).Msg("ensure system rooms after role change")
			}
			if err := s.chatSync.SyncSystemRoomMembership(ctx, targetID, r); err != nil {
				logger.Log.Error().Err(err).Str("user_id", targetID.String()).Msg("sync system rooms after role set")
			}
		}
		s.broadcastRoleChange(targetID, string(r))
		return nil
	})
}

func (s *service) RemoveUserRole(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, r role.Role) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.roleRepo.RemoveRole(ctx, targetID, r); err != nil {
			return fmt.Errorf("remove role: %w", err)
		}
		s.auditDetails(ctx, actorID, "remove_role", "user", targetID.String(), string(r))
		if s.chatSync != nil {
			if err := s.chatSync.SyncSystemRoomMembership(ctx, targetID, ""); err != nil {
				logger.Log.Error().Err(err).Str("user_id", targetID.String()).Msg("sync system rooms after role remove")
			}
		}
		s.broadcastRoleChange(targetID, "")
		return nil
	})
}

func (s *service) broadcastRoleChange(userID uuid.UUID, newRole string) {
	s.hub.Broadcast(ws.Message{
		Type: "role_changed",
		Data: map[string]any{
			"user_id": userID,
			"role":    newRole,
		},
	})
}

func (s *service) BanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.userRepo.BanUser(ctx, targetID, actorID, reason); err != nil {
			return fmt.Errorf("ban user: %w", err)
		}
		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			logger.Log.Error().Err(err).Str("user_id", targetID.String()).Msg("failed to invalidate sessions after ban")
		}
		s.auditDetails(ctx, actorID, "ban_user", "user", targetID.String(), reason)
		s.broadcastBanChange(targetID, true, reason)
		return nil
	})
}

func (s *service) UnbanUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.UnbanUser(ctx, targetID); err != nil {
			return fmt.Errorf("unban user: %w", err)
		}
		s.audit(ctx, actorID, "unban_user", "user", targetID.String())
		s.broadcastBanChange(targetID, false, "")
		return nil
	})
}

func (s *service) broadcastBanChange(userID uuid.UUID, banned bool, reason string) {
	s.hub.Broadcast(ws.Message{
		Type: "ban_changed",
		Data: map[string]any{
			"user_id":    userID,
			"banned":     banned,
			"ban_reason": reason,
		},
	})
}

func (s *service) LockUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, reason string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		if err := s.userRepo.LockUser(ctx, targetID, actorID, reason); err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		s.auditDetails(ctx, actorID, "lock_user", "user", targetID.String(), reason)
		s.broadcastLockChange(targetID, true, reason)
		return nil
	})
}

func (s *service) UnlockUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.UnlockUser(ctx, targetID); err != nil {
			return fmt.Errorf("unlock user: %w", err)
		}
		s.audit(ctx, actorID, "unlock_user", "user", targetID.String())
		s.broadcastLockChange(targetID, false, "")
		return nil
	})
}

func (s *service) broadcastLockChange(userID uuid.UUID, locked bool, reason string) {
	s.hub.SendToUser(userID, ws.Message{
		Type: "lock_changed",
		Data: map[string]any{
			"user_id":     userID,
			"locked":      locked,
			"lock_reason": reason,
		},
	})
}

func (s *service) DeleteUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	user, _ := s.userRepo.GetByID(ctx, targetID)

	return s.guardedAction(ctx, actorID, targetID, func() error {
		if user != nil && user.IsBot {
			return ErrBotAccountProtected
		}

		if err := s.userRepo.AdminDeleteAccount(ctx, targetID); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		if user != nil {
			s.uploadSvc.Delete(user.AvatarURL)
			s.uploadSvc.Delete(user.BannerURL)
		}
		s.audit(ctx, actorID, "delete_user", "user", targetID.String())
		return nil
	})
}

func (s *service) ResetUserPassword(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (string, error) {
	var newPassword string
	err := s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.rejectBotTarget(ctx, targetID); err != nil {
			return err
		}

		generated, err := generatePassword()
		if err != nil {
			return fmt.Errorf("generate password: %w", err)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(generated), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		if err := s.userRepo.SetPasswordHash(ctx, targetID, string(passwordHash)); err != nil {
			return fmt.Errorf("set password: %w", err)
		}

		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			logger.Log.Warn().Err(err).Str("user_id", targetID.String()).Msg("failed to invalidate sessions after password reset")
		}
		s.audit(ctx, actorID, "reset_password", "user", targetID.String())
		newPassword = generated
		return nil
	})
	if err != nil {
		return "", err
	}
	return newPassword, nil
}

func (s *service) SetUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, email string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		previous := s.currentEmail(ctx, targetID)

		if err := s.authSvc.SetEmailForUser(ctx, targetID, email); err != nil {
			return fmt.Errorf("set email: %w", err)
		}

		s.auditDetails(ctx, actorID, "set_user_email", "user", targetID.String(), changeDetails(previous, email))
		return nil
	})
}

func (s *service) currentEmail(ctx context.Context, targetID uuid.UUID) string {
	usr, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil || usr == nil {
		return ""
	}
	return usr.Email
}

func (s *service) currentDisplayName(ctx context.Context, targetID uuid.UUID) string {
	usr, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil || usr == nil {
		return ""
	}
	return usr.DisplayName
}

func changeDetails(previous, next string) string {
	if previous == "" {
		return next
	}
	return fmt.Sprintf("%s -> %s", previous, next)
}

func (s *service) VerifyUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.authSvc.MarkEmailVerified(ctx, targetID); err != nil {
			return fmt.Errorf("verify email: %w", err)
		}

		s.audit(ctx, actorID, "verify_user_email", "user", targetID.String())
		return nil
	})
}

func (s *service) UnverifyUserEmail(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.authSvc.MarkEmailUnverified(ctx, targetID); err != nil {
			return fmt.Errorf("unverify email: %w", err)
		}

		s.audit(ctx, actorID, "unverify_user_email", "user", targetID.String())
		return nil
	})
}

func (s *service) SetUserDisplayName(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, displayName string) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		clamped := userpkg.ClampDisplayName(displayName)
		if clamped == "" {
			return ErrEmptyDisplayName
		}

		previous := s.currentDisplayName(ctx, targetID)

		if err := s.userRepo.SetDisplayName(ctx, targetID, clamped); err != nil {
			return fmt.Errorf("set display name: %w", err)
		}

		s.auditDetails(ctx, actorID, "set_display_name", "user", targetID.String(), changeDetails(previous, clamped))
		s.broadcastDisplayNameChange(targetID, clamped)
		return nil
	})
}

func (s *service) SetDisplayNameLocked(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, locked bool) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.userRepo.SetDisplayNameLocked(ctx, targetID, locked); err != nil {
			return fmt.Errorf("set display name lock: %w", err)
		}

		action := "unlock_display_name"
		if locked {
			action = "lock_display_name"
		}
		s.audit(ctx, actorID, action, "user", targetID.String())
		return nil
	})
}

func (s *service) broadcastDisplayNameChange(userID uuid.UUID, displayName string) {
	s.hub.Broadcast(ws.Message{
		Type: "profile_changed",
		Data: map[string]any{
			"user_id":      userID,
			"display_name": displayName,
		},
	})
}

func (s *service) ForceLogout(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error {
	return s.guardedAction(ctx, actorID, targetID, func() error {
		if err := s.sessionMgr.DeleteAllForUser(ctx, targetID); err != nil {
			return fmt.Errorf("delete sessions: %w", err)
		}

		s.audit(ctx, actorID, "force_logout", "user", targetID.String())
		return nil
	})
}

func (s *service) ListAccountsOnIP(ctx context.Context, targetID uuid.UUID) (*dto.AdminIPMatchesResponse, error) {
	target, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if target == nil {
		return nil, ErrUserNotFound
	}

	if target.IP == nil || *target.IP == "" {
		return &dto.AdminIPMatchesResponse{Users: []dto.AdminUserItem{}}, nil
	}

	users, err := s.userRepo.ListByIP(ctx, *target.IP, targetID)
	if err != nil {
		return nil, fmt.Errorf("list users by ip: %w", err)
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
			Locked:      u.LockedAt != nil,
			CreatedAt:   u.CreatedAt,
		}
	}

	return &dto.AdminIPMatchesResponse{IP: *target.IP, Users: items}, nil
}

func (s *service) GetUserAuditLog(ctx context.Context, targetID uuid.UUID, page bounds.Page) (*dto.AuditLogListResponse, error) {
	entries, total, err := s.auditRepo.ListForUser(ctx, targetID, page.Limit(), page.Offset())
	if err != nil {
		return nil, fmt.Errorf("get user audit log: %w", err)
	}

	return &dto.AuditLogListResponse{
		Entries: toAuditLogEntries(entries),
		Total:   total,
		Limit:   page.Limit(),
		Offset:  page.Offset(),
	}, nil
}

func generatePassword() (string, error) {
	const alphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const length = 16

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	out := make([]byte, length)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

func (s *service) GetSettings(ctx context.Context) (*dto.SettingsResponse, error) {
	all := s.settingsSvc.GetAll(ctx)

	result := make(map[string]string, len(all))
	for k, v := range all {
		if def, ok := config.SettingByKey(k); ok && def.Secret && v != "" {
			result[string(k)] = config.SecretMask
			continue
		}
		result[string(k)] = v
	}

	return &dto.SettingsResponse{Settings: result}, nil
}

func (s *service) UpdateSettings(ctx context.Context, actorID uuid.UUID, settings map[string]string) error {
	current := s.settingsSvc.GetAll(ctx)

	typed := make(map[config.SiteSettingKey]string, len(settings))
	for k, v := range settings {
		key := config.SiteSettingKey(k)
		if def, ok := config.SettingByKey(key); ok && def.Secret && v == config.SecretMask {
			continue
		}
		typed[key] = v
	}

	changedKeys := make([]string, 0, len(typed))
	for k, v := range typed {
		if current[k] != v {
			changedKeys = append(changedKeys, string(k))
		}
	}
	slices.Sort(changedKeys)

	if err := s.settingsSvc.SetMultiple(ctx, typed, actorID); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	if newRules, ok := typed[config.SettingRulesPage.Key]; ok && newRules != current[config.SettingRulesPage.Key] {
		s.hub.Broadcast(ws.Message{
			Type: "rules_page_changed",
			Data: map[string]any{},
		})
	}

	details := fmt.Sprintf("changed %d of %d submitted settings", len(changedKeys), len(settings))
	s.auditDetails(ctx, actorID, "update_settings", "settings", strings.Join(changedKeys, ","), details)

	return nil
}

func (s *service) SendTestEmail(ctx context.Context, actorID uuid.UUID) error {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("get actor: %w", err)
	}
	if actor == nil {
		return ErrUserNotFound
	}
	if actor.Email == "" {
		return ErrNoEmailAddress
	}

	siteName := s.settingsSvc.Get(ctx, config.SettingSiteName)
	subject := "Test email from " + siteName
	body := fmt.Sprintf("<p>This is a test email from %s confirming your email settings are working.</p>", siteName)

	if err := s.emailSvc.SendTest(ctx, actor.Email, subject, body); err != nil {
		return fmt.Errorf("send test email: %w", err)
	}

	s.audit(ctx, actorID, "send_test_email", "settings", "")
	return nil
}

func (s *service) GetAuditLog(ctx context.Context, action string, page bounds.Page) (*dto.AuditLogListResponse, error) {
	entries, total, err := s.auditRepo.List(ctx, action, page.Limit(), page.Offset())
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	return &dto.AuditLogListResponse{
		Entries: toAuditLogEntries(entries),
		Total:   total,
		Limit:   page.Limit(),
		Offset:  page.Offset(),
	}, nil
}

func toAuditLogEntries(entries []repository.AuditLogEntry) []dto.AuditLogEntryResponse {
	items := make([]dto.AuditLogEntryResponse, len(entries))
	for i, e := range entries {
		items[i] = dto.AuditLogEntryResponse{
			ID:              e.ID,
			ActorID:         e.ActorID,
			ActorName:       e.ActorName,
			Action:          e.Action,
			TargetType:      e.TargetType,
			TargetID:        e.TargetID,
			Details:         e.Details,
			CreatedAt:       e.CreatedAt,
			SubjectID:       e.SubjectID,
			SubjectName:     e.SubjectName,
			SubjectUsername: e.SubjectUsername,
		}
	}
	return items
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

func (s *service) ListInvites(ctx context.Context, page bounds.Page) (*dto.InviteListResponse, error) {
	invites, total, err := s.inviteRepo.List(ctx, page.Limit(), page.Offset())
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
		Limit:   page.Limit(),
		Offset:  page.Offset(),
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
	req.SortOrder = max(req.SortOrder, 0)

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

	if s.isChatbotOptInRole(ctx, id) {
		return ErrVanityRoleOptInLocked
	}

	if err := s.vanityRoleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete vanity role: %w", err)
	}
	s.audit(ctx, actorID, "delete_vanity_role", "vanity_role", id)
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
	rows, total, err := s.vanityRoleRepo.GetUsersForRole(ctx, roleID, search, page.Limit(), page.Offset())
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

	if err := s.vanityRoleRepo.AssignToUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("assign vanity role: %w", err)
	}
	s.auditSubject(ctx, actorID, "assign_vanity_role", "vanity_role", roleID, userID)
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

	if err := s.vanityRoleRepo.UnassignFromUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("unassign vanity role: %w", err)
	}
	s.auditSubject(ctx, actorID, "unassign_vanity_role", "vanity_role", roleID, userID)
	s.broadcastVanityRolesChanged()
	return nil
}

func (s *service) broadcastVanityRolesChanged() {
	s.hub.Broadcast(ws.Message{
		Type: "vanity_roles_changed",
		Data: map[string]any{},
	})
}
