package admin

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/giphy/banlist"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
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
		ApproveUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
		UnapproveUser(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) error
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

		GetAuditLog(ctx context.Context, action audit.Action, page bounds.Page) (*dto.AuditLogListResponse, error)

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

func (s *service) audit(ctx context.Context, actorID uuid.UUID, action audit.Action, targetType audit.TargetType, targetID string) {
	s.auditDetails(ctx, actorID, action, targetType, targetID, "")
}

func (s *service) auditDetails(ctx context.Context, actorID uuid.UUID, action audit.Action, targetType audit.TargetType, targetID, details string) {
	if err := s.auditRepo.Create(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    details,
	}); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(action)).Msg("failed to write audit log")
	}
}

func (s *service) auditUser(ctx context.Context, actorID uuid.UUID, action audit.Action, subjectID uuid.UUID) {
	s.auditUserDetails(ctx, actorID, action, subjectID, "")
}

func (s *service) auditUserDetails(ctx context.Context, actorID uuid.UUID, action audit.Action, subjectID uuid.UUID, details string) {
	if err := s.auditRepo.Create(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: audit.TargetUser,
		TargetID:   subjectID.String(),
		Details:    details,
		SubjectID:  subjectID,
	}); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(action)).Msg("failed to write audit log")
	}
}

func (s *service) auditSubject(ctx context.Context, actorID uuid.UUID, action audit.Action, targetType audit.TargetType, targetID string, subjectID uuid.UUID) {
	if err := s.auditRepo.Create(ctx, audit.NewEntry{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		SubjectID:  subjectID,
	}); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("action", string(action)).Msg("failed to write audit log")
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
	users, total, err := s.userRepo.ListAll(ctx, spec.UserListFilter{
		Search: search,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	})
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

	resp.Restricted = u.IsRestrictedNewAccount(s.settingsSvc.GetInt(ctx, config.SettingNewAccountHours))
	if u.ApprovedAt != nil {
		resp.ApprovedAt = *u.ApprovedAt
	}
	if u.ApprovedBy != nil {
		approver, err := s.userRepo.GetByID(ctx, *u.ApprovedBy)
		if err != nil {
			return nil, fmt.Errorf("get approved_by user: %w", err)
		}
		if approver != nil {
			resp.ApprovedBy = approver.ToResponse()
		}
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
