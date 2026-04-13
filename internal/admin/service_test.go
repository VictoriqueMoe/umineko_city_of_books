package admin

import (
	"context"
	"errors"
	"sync"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeChatSync struct {
	ensureErr      error
	syncErr        error
	ensureCalls    int
	syncCalls      int
	lastSyncRole   role.Role
	lastSyncUserID uuid.UUID
	mu             sync.Mutex
}

func (f *fakeChatSync) EnsureSystemRooms(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureCalls++
	return f.ensureErr
}

func (f *fakeChatSync) SyncSystemRoomMembership(ctx context.Context, userID uuid.UUID, newRole role.Role) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.syncCalls++
	f.lastSyncUserID = userID
	f.lastSyncRole = newRole
	return f.syncErr
}

type testMocks struct {
	userRepo    *repository.MockUserRepository
	roleRepo    *repository.MockRoleRepository
	statsRepo   *repository.MockStatsRepository
	auditRepo   *repository.MockAuditLogRepository
	inviteRepo  *repository.MockInviteRepository
	vanityRepo  *repository.MockVanityRoleRepository
	sessionRepo *repository.MockSessionRepository
	authz       *authz.MockService
	settingsSvc *settings.MockService
	uploadSvc   *upload.MockService
	hub         *ws.Hub
	chatSync    *fakeChatSync
}

func newTestService(t *testing.T) (*service, *testMocks) {
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	statsRepo := repository.NewMockStatsRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	inviteRepo := repository.NewMockInviteRepository(t)
	vanityRepo := repository.NewMockVanityRoleRepository(t)
	sessionRepo := repository.NewMockSessionRepository(t)
	authzSvc := authz.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	hub := ws.NewHub()
	chatSync := &fakeChatSync{}
	sessionMgr := session.NewManager(sessionRepo, settingsSvc)

	svc := NewService(
		userRepo,
		roleRepo,
		statsRepo,
		auditRepo,
		inviteRepo,
		vanityRepo,
		authzSvc,
		settingsSvc,
		sessionMgr,
		uploadSvc,
		hub,
		chatSync,
	).(*service)

	return svc, &testMocks{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		statsRepo:   statsRepo,
		auditRepo:   auditRepo,
		inviteRepo:  inviteRepo,
		vanityRepo:  vanityRepo,
		sessionRepo: sessionRepo,
		authz:       authzSvc,
		settingsSvc: settingsSvc,
		uploadSvc:   uploadSvc,
		hub:         hub,
		chatSync:    chatSync,
	}
}

// --- GetStats ---

func TestGetStats_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.statsRepo.EXPECT().GetOverview(mock.Anything).Return(&repository.SiteStats{
		TotalUsers:     5,
		TotalTheories:  3,
		TotalResponses: 2,
		PostsByCorner:  map[string]int{"a": 1},
	}, nil)
	m.statsRepo.EXPECT().GetMostActiveUsers(mock.Anything, 10).Return([]repository.ActiveUser{
		{ID: userID, Username: "u", DisplayName: "U", AvatarURL: "/a.png", ActionCount: 7},
	}, nil)

	// when
	got, err := svc.GetStats(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, got.TotalUsers)
	assert.Equal(t, 3, got.TotalTheories)
	assert.Len(t, got.MostActiveUsers, 1)
	assert.Equal(t, userID, got.MostActiveUsers[0].ID)
	assert.Equal(t, 7, got.MostActiveUsers[0].ActionCount)
}

func TestGetStats_OverviewError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.statsRepo.EXPECT().GetOverview(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetStats(context.Background())

	// then
	require.Error(t, err)
}

func TestGetStats_ActiveUsersError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.statsRepo.EXPECT().GetOverview(mock.Anything).Return(&repository.SiteStats{}, nil)
	m.statsRepo.EXPECT().GetMostActiveUsers(mock.Anything, 10).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetStats(context.Background())

	// then
	require.Error(t, err)
}

// --- ListUsers ---

func TestListUsers_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	bannedAt := "2026-01-01"
	m.userRepo.EXPECT().ListAll(mock.Anything, "query", 10, 0).Return([]model.User{
		{ID: uid, Username: "a", DisplayName: "A", Role: string(authz.RoleAdmin), BannedAt: &bannedAt},
		{ID: uuid.New(), Username: "b", DisplayName: "B"},
	}, 2, nil)

	// when
	got, err := svc.ListUsers(context.Background(), "query", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, got.Total)
	assert.Len(t, got.Users, 2)
	assert.True(t, got.Users[0].Banned)
	assert.Equal(t, role.Role(authz.RoleAdmin), got.Users[0].Role)
	assert.False(t, got.Users[1].Banned)
}

func TestListUsers_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.userRepo.EXPECT().ListAll(mock.Anything, "", 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListUsers(context.Background(), "", 10, 0)

	// then
	require.Error(t, err)
}

// --- GetUser ---

func TestGetUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	ip := "127.0.0.1"
	bannedAt := "2026-01-01"
	m.userRepo.EXPECT().GetProfileByID(mock.Anything, uid).Return(&model.User{
		ID:                     uid,
		Username:               "a",
		IP:                     &ip,
		BannedAt:               &bannedAt,
		BanReason:              "spam",
		MysteryScoreAdjustment: 5,
		GMScoreAdjustment:      3,
	}, &model.UserStats{TheoryCount: 10, ResponseCount: 7}, nil)
	m.userRepo.EXPECT().GetDetectiveRawScore(mock.Anything, uid).Return(100, nil)
	m.userRepo.EXPECT().GetGMRawScore(mock.Anything, uid).Return(50, nil)

	// when
	got, err := svc.GetUser(context.Background(), uid)

	// then
	require.NoError(t, err)
	assert.Equal(t, uid, got.ID)
	assert.True(t, got.Banned)
	assert.Equal(t, "127.0.0.1", got.IP)
	assert.Equal(t, "spam", got.BanReason)
	assert.Equal(t, 10, got.TheoryCount)
	assert.Equal(t, 105, got.DetectiveScore)
	assert.Equal(t, 53, got.GMScore)
}

func TestGetUser_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	m.userRepo.EXPECT().GetProfileByID(mock.Anything, uid).Return(nil, nil, nil)

	// when
	_, err := svc.GetUser(context.Background(), uid)

	// then
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestGetUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	m.userRepo.EXPECT().GetProfileByID(mock.Anything, uid).Return(nil, nil, errors.New("boom"))

	// when
	_, err := svc.GetUser(context.Background(), uid)

	// then
	require.Error(t, err)
}

// --- guardedAction via SetUserRole ---

func TestSetUserRole_ProtectedSuperAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleSuperAdmin, nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleModerator)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestSetUserRole_ProtectedEqualRank(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleAdmin, nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleModerator)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestSetUserRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.roleRepo.EXPECT().SetRole(mock.Anything, target, authz.RoleAdmin).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "set_role", "user", target.String(), "").Return(nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, m.chatSync.ensureCalls)
	assert.Equal(t, 1, m.chatSync.syncCalls)
	assert.Equal(t, authz.RoleAdmin, m.chatSync.lastSyncRole)
}

func TestSetUserRole_SetRoleError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.roleRepo.EXPECT().SetRole(mock.Anything, target, authz.RoleAdmin).Return(errors.New("boom"))

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.Error(t, err)
}

func TestSetUserRole_ChatSyncErrorsLogged(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.chatSync.ensureErr = errors.New("ensure boom")
	m.chatSync.syncErr = errors.New("sync boom")
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.roleRepo.EXPECT().SetRole(mock.Anything, target, authz.RoleAdmin).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "set_role", "user", target.String(), "").Return(nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.NoError(t, err)
}

// --- RemoveUserRole ---

func TestRemoveUserRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleModerator, nil)
	m.roleRepo.EXPECT().RemoveRole(mock.Anything, target, authz.RoleModerator).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "remove_role", "user", target.String(), "").Return(nil)

	// when
	err := svc.RemoveUserRole(context.Background(), actor, target, authz.RoleModerator)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, m.chatSync.syncCalls)
	assert.Equal(t, role.Role(""), m.chatSync.lastSyncRole)
}

func TestRemoveUserRole_Protected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleAdmin, nil)

	// when
	err := svc.RemoveUserRole(context.Background(), actor, target, authz.RoleModerator)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestRemoveUserRole_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleModerator, nil)
	m.roleRepo.EXPECT().RemoveRole(mock.Anything, target, authz.RoleModerator).Return(errors.New("boom"))

	// when
	err := svc.RemoveUserRole(context.Background(), actor, target, authz.RoleModerator)

	// then
	require.Error(t, err)
}

// --- BanUser ---

func TestBanUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.userRepo.EXPECT().BanUser(mock.Anything, target, actor, "reason").Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "ban_user", "user", target.String(), "").Return(nil)

	// when
	err := svc.BanUser(context.Background(), actor, target, "reason")

	// then
	require.NoError(t, err)
}

func TestBanUser_SessionDeleteErrorSwallowed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.userRepo.EXPECT().BanUser(mock.Anything, target, actor, "reason").Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(errors.New("session boom"))
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "ban_user", "user", target.String(), "").Return(nil)

	// when
	err := svc.BanUser(context.Background(), actor, target, "reason")

	// then
	require.NoError(t, err)
}

func TestBanUser_Protected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleSuperAdmin, nil)

	// when
	err := svc.BanUser(context.Background(), actor, target, "r")

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestBanUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.userRepo.EXPECT().BanUser(mock.Anything, target, actor, "r").Return(errors.New("boom"))

	// when
	err := svc.BanUser(context.Background(), actor, target, "r")

	// then
	require.Error(t, err)
}

// --- UnbanUser ---

func TestUnbanUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().UnbanUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "unban_user", "user", target.String(), "").Return(nil)

	// when
	err := svc.UnbanUser(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestUnbanUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().UnbanUser(mock.Anything, target).Return(errors.New("boom"))

	// when
	err := svc.UnbanUser(context.Background(), actor, target)

	// then
	require.Error(t, err)
}

// --- DeleteUser ---

func TestDeleteUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID:        target,
		AvatarURL: "/a.png",
		BannerURL: "/b.png",
	}, nil)
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.userRepo.EXPECT().AdminDeleteAccount(mock.Anything, target).Return(nil)
	m.uploadSvc.EXPECT().Delete("/a.png").Return(nil)
	m.uploadSvc.EXPECT().Delete("/b.png").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "delete_user", "user", target.String(), "").Return(nil)

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestDeleteUser_UserLookupFailsStillDeletes(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(nil, errors.New("not found"))
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.userRepo.EXPECT().AdminDeleteAccount(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "delete_user", "user", target.String(), "").Return(nil)

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestDeleteUser_Protected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleAdmin, nil)

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestDeleteUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(""), nil)
	m.userRepo.EXPECT().AdminDeleteAccount(mock.Anything, target).Return(errors.New("boom"))

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	require.Error(t, err)
}

// --- GetSettings ---

func TestGetSettings_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{
		"site_name": "umineko",
		"foo":       "bar",
	})

	// when
	got, err := svc.GetSettings(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, "umineko", got.Settings["site_name"])
	assert.Equal(t, "bar", got.Settings["foo"])
}

// --- UpdateSettings ---

func TestUpdateSettings_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "update_settings", "settings", "", "").Return(nil)

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"site_name": "umineko"})

	// then
	require.NoError(t, err)
}

func TestUpdateSettings_Error(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(errors.New("boom"))

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"site_name": "umineko"})

	// then
	require.Error(t, err)
}

// --- GetAuditLog ---

func TestGetAuditLog_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.auditRepo.EXPECT().List(mock.Anything, "ban_user", 20, 0).Return([]repository.AuditLogEntry{
		{ID: 1, ActorID: actor, ActorName: "victorique", Action: "ban_user", TargetType: "user", TargetID: "t", Details: "d", CreatedAt: "now"},
	}, 1, nil)

	// when
	got, err := svc.GetAuditLog(context.Background(), "ban_user", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, "ban_user", got.Entries[0].Action)
	assert.Equal(t, "victorique", got.Entries[0].ActorName)
}

func TestGetAuditLog_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.auditRepo.EXPECT().List(mock.Anything, "", 20, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetAuditLog(context.Background(), "", 20, 0)

	// then
	require.Error(t, err)
}

// --- CreateInvite ---

func TestCreateInvite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "create_invite", "invite", mock.AnythingOfType("string"), "").Return(nil)

	// when
	got, err := svc.CreateInvite(context.Background(), actor)

	// then
	require.NoError(t, err)
	assert.Len(t, got.Code, 8)
	assert.Equal(t, actor, got.CreatedBy)
}

func TestCreateInvite_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), actor).Return(errors.New("boom"))

	// when
	_, err := svc.CreateInvite(context.Background(), actor)

	// then
	require.Error(t, err)
}

// --- ListInvites ---

func TestListInvites_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	m.inviteRepo.EXPECT().List(mock.Anything, 10, 0).Return([]repository.Invite{
		{Code: "abc", CreatedBy: creator, CreatedAt: "t"},
	}, 1, nil)

	// when
	got, err := svc.ListInvites(context.Background(), 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	require.Len(t, got.Invites, 1)
	assert.Equal(t, "abc", got.Invites[0].Code)
}

func TestListInvites_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.inviteRepo.EXPECT().List(mock.Anything, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListInvites(context.Background(), 10, 0)

	// then
	require.Error(t, err)
}

// --- DeleteInvite ---

func TestDeleteInvite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Delete(mock.Anything, "abc").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "delete_invite", "invite", "abc", "").Return(nil)

	// when
	err := svc.DeleteInvite(context.Background(), actor, "abc")

	// then
	require.NoError(t, err)
}

func TestDeleteInvite_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Delete(mock.Anything, "abc").Return(errors.New("boom"))

	// when
	err := svc.DeleteInvite(context.Background(), actor, "abc")

	// then
	require.Error(t, err)
}

// --- ListVanityRoles ---

func TestListVanityRoles_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().List(mock.Anything).Return([]repository.VanityRoleRow{
		{ID: "r1", Label: "L", Color: "#ff0000", IsSystem: true, SortOrder: 1},
	}, nil)

	// when
	got, err := svc.ListVanityRoles(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "r1", got[0].ID)
	assert.True(t, got[0].IsSystem)
}

func TestListVanityRoles_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().List(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListVanityRoles(context.Background())

	// then
	require.Error(t, err)
}

// --- CreateVanityRole ---

func TestCreateVanityRole_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		req  dto.CreateVanityRoleRequest
	}{
		{"empty label", dto.CreateVanityRoleRequest{Label: "   ", Color: "#ff0000"}},
		{"bad color", dto.CreateVanityRoleRequest{Label: "ok", Color: "red"}},
		{"short hex", dto.CreateVanityRoleRequest{Label: "ok", Color: "#fff"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			_, err := svc.CreateVanityRole(context.Background(), uuid.New(), tc.req)

			// then
			require.Error(t, err)
		})
	}
}

func TestCreateVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), "gold", "#ffcc00", 3).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "create_vanity_role", "vanity_role", mock.AnythingOfType("string"), "").Return(nil)

	// when
	got, err := svc.CreateVanityRole(context.Background(), actor, dto.CreateVanityRoleRequest{
		Label:     "  gold  ",
		Color:     "#ffcc00",
		SortOrder: 3,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, "gold", got.Label)
	assert.Equal(t, "#ffcc00", got.Color)
	assert.Equal(t, 3, got.SortOrder)
	assert.False(t, got.IsSystem)
}

func TestCreateVanityRole_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), "gold", "#ffcc00", 0).Return(errors.New("boom"))

	// when
	_, err := svc.CreateVanityRole(context.Background(), actor, dto.CreateVanityRoleRequest{
		Label: "gold",
		Color: "#ffcc00",
	})

	// then
	require.Error(t, err)
}

// --- UpdateVanityRole ---

func TestUpdateVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	id := "r1"
	m.vanityRepo.EXPECT().GetByID(mock.Anything, id).Return(&repository.VanityRoleRow{ID: id, IsSystem: false}, nil)
	m.vanityRepo.EXPECT().Update(mock.Anything, id, "silver", "#cccccc", 2).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "update_vanity_role", "vanity_role", id, "").Return(nil)

	// when
	err := svc.UpdateVanityRole(context.Background(), actor, id, dto.UpdateVanityRoleRequest{
		Label:     "silver",
		Color:     "#cccccc",
		SortOrder: 2,
	})

	// then
	require.NoError(t, err)
}

func TestUpdateVanityRole_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, nil)

	// when
	err := svc.UpdateVanityRole(context.Background(), uuid.New(), "r1", dto.UpdateVanityRoleRequest{Label: "x", Color: "#000000"})

	// then
	assert.ErrorIs(t, err, ErrVanityRoleNotFound)
}

func TestUpdateVanityRole_GetError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, errors.New("boom"))

	// when
	err := svc.UpdateVanityRole(context.Background(), uuid.New(), "r1", dto.UpdateVanityRoleRequest{Label: "x", Color: "#000000"})

	// then
	require.Error(t, err)
}

func TestUpdateVanityRole_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		req  dto.UpdateVanityRoleRequest
	}{
		{"empty label", dto.UpdateVanityRoleRequest{Label: " ", Color: "#000000"}},
		{"bad color", dto.UpdateVanityRoleRequest{Label: "x", Color: "nope"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)

			// when
			err := svc.UpdateVanityRole(context.Background(), uuid.New(), "r1", tc.req)

			// then
			require.Error(t, err)
		})
	}
}

func TestUpdateVanityRole_UpdateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().Update(mock.Anything, "r1", "x", "#000000", 0).Return(errors.New("boom"))

	// when
	err := svc.UpdateVanityRole(context.Background(), uuid.New(), "r1", dto.UpdateVanityRoleRequest{Label: "x", Color: "#000000"})

	// then
	require.Error(t, err)
}

// --- DeleteVanityRole ---

func TestDeleteVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1", IsSystem: false}, nil)
	m.vanityRepo.EXPECT().Delete(mock.Anything, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "delete_vanity_role", "vanity_role", "r1", "").Return(nil)

	// when
	err := svc.DeleteVanityRole(context.Background(), actor, "r1")

	// then
	require.NoError(t, err)
}

func TestDeleteVanityRole_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, nil)

	// when
	err := svc.DeleteVanityRole(context.Background(), uuid.New(), "r1")

	// then
	assert.ErrorIs(t, err, ErrVanityRoleNotFound)
}

func TestDeleteVanityRole_GetError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, errors.New("boom"))

	// when
	err := svc.DeleteVanityRole(context.Background(), uuid.New(), "r1")

	// then
	require.Error(t, err)
}

func TestDeleteVanityRole_SystemRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1", IsSystem: true}, nil)

	// when
	err := svc.DeleteVanityRole(context.Background(), uuid.New(), "r1")

	// then
	assert.ErrorIs(t, err, ErrSystemRole)
}

func TestDeleteVanityRole_DeleteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().Delete(mock.Anything, "r1").Return(errors.New("boom"))

	// when
	err := svc.DeleteVanityRole(context.Background(), uuid.New(), "r1")

	// then
	require.Error(t, err)
}

// --- GetVanityRoleUsers ---

func TestGetVanityRoleUsers_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	m.vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "r1", "q", 10, 0).Return([]repository.VanityRoleUserRow{
		{UserID: uid, Username: "u", DisplayName: "U", AvatarURL: "/a.png"},
	}, 1, nil)

	// when
	got, err := svc.GetVanityRoleUsers(context.Background(), "r1", "q", 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	require.Len(t, got.Users, 1)
	assert.Equal(t, uid, got.Users[0].ID)
}

func TestGetVanityRoleUsers_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "r1", "", 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetVanityRoleUsers(context.Background(), "r1", "", 10, 0)

	// then
	require.Error(t, err)
}

// --- AssignVanityRole ---

func TestAssignVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, target, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "assign_vanity_role", "vanity_role", "r1:"+target.String(), "").Return(nil)

	// when
	err := svc.AssignVanityRole(context.Background(), actor, "r1", target)

	// then
	require.NoError(t, err)
}

func TestAssignVanityRole_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, nil)

	// when
	err := svc.AssignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	assert.ErrorIs(t, err, ErrVanityRoleNotFound)
}

func TestAssignVanityRole_GetError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, errors.New("boom"))

	// when
	err := svc.AssignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	require.Error(t, err)
}

func TestAssignVanityRole_SystemRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1", IsSystem: true}, nil)

	// when
	err := svc.AssignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	assert.ErrorIs(t, err, ErrSystemRole)
}

func TestAssignVanityRole_AssignError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, target, "r1").Return(errors.New("boom"))

	// when
	err := svc.AssignVanityRole(context.Background(), uuid.New(), "r1", target)

	// then
	require.Error(t, err)
}

// --- UnassignVanityRole ---

func TestUnassignVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().UnassignFromUser(mock.Anything, target, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "unassign_vanity_role", "vanity_role", "r1:"+target.String(), "").Return(nil)

	// when
	err := svc.UnassignVanityRole(context.Background(), actor, "r1", target)

	// then
	require.NoError(t, err)
}

func TestUnassignVanityRole_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, nil)

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	assert.ErrorIs(t, err, ErrVanityRoleNotFound)
}

func TestUnassignVanityRole_GetError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(nil, errors.New("boom"))

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	require.Error(t, err)
}

func TestUnassignVanityRole_SystemRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1", IsSystem: true}, nil)

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	assert.ErrorIs(t, err, ErrSystemRole)
}

func TestUnassignVanityRole_UnassignError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().UnassignFromUser(mock.Anything, target, "r1").Return(errors.New("boom"))

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", target)

	// then
	require.Error(t, err)
}
