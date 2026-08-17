package admin

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/giphy/banlist"
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
	permRepo    *repository.MockPermissionRepository
	sessionRepo *repository.MockSessionRepository
	bannedRepo  *repository.MockBannedGiphyRepository
	authz       *authz.MockService
	settingsSvc *settings.MockService
	uploadSvc   *upload.MockService
	hub         *ws.Hub
	chatSync    *fakeChatSync
	banlist     banlist.Service
	emailSvc    *email.MockService
	authSvc     *auth.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	statsRepo := repository.NewMockStatsRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	inviteRepo := repository.NewMockInviteRepository(t)
	vanityRepo := repository.NewMockVanityRoleRepository(t)
	permRepo := repository.NewMockPermissionRepository(t)
	sessionRepo := repository.NewMockSessionRepository(t)
	bannedRepo := repository.NewMockBannedGiphyRepository(t)
	bannedRepo.EXPECT().List(mock.Anything).Return(nil, nil).Maybe()
	banlistSvc, err := banlist.NewService(context.Background(), bannedRepo)
	require.NoError(t, err)
	authzSvc := authz.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	hub := ws.NewHub()
	chatSync := &fakeChatSync{}
	sessionMgr := session.NewManager(sessionRepo, settingsSvc)
	emailSvc := email.NewMockService(t)
	authSvc := auth.NewMockService(t)

	svc := NewService(
		userRepo,
		roleRepo,
		statsRepo,
		auditRepo,
		inviteRepo,
		vanityRepo,
		permRepo,
		banlistSvc,
		authzSvc,
		settingsSvc,
		sessionMgr,
		uploadSvc,
		hub,
		chatSync,
		emailSvc,
		authSvc,
	).(*service)

	return svc, &testMocks{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		statsRepo:   statsRepo,
		auditRepo:   auditRepo,
		inviteRepo:  inviteRepo,
		vanityRepo:  vanityRepo,
		permRepo:    permRepo,
		sessionRepo: sessionRepo,
		bannedRepo:  bannedRepo,
		authz:       authzSvc,
		settingsSvc: settingsSvc,
		uploadSvc:   uploadSvc,
		hub:         hub,
		chatSync:    chatSync,
		banlist:     banlistSvc,
		emailSvc:    emailSvc,
		authSvc:     authSvc,
	}
}

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

func TestListUsers_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	m.userRepo.EXPECT().ListAll(mock.Anything, "query", 10, 0).Return([]model.User{
		{ID: uid, Username: "a", DisplayName: "A", Role: string(authz.RoleAdmin), BannedAt: new("2026-01-01")},
		{ID: uuid.New(), Username: "b", DisplayName: "B"},
	}, 2, nil)

	// when
	got, err := svc.ListUsers(context.Background(), "query", bounds.NewPage(10, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, got.Total)
	assert.Len(t, got.Users, 2)
	assert.True(t, got.Users[0].Banned)
	assert.Equal(t, authz.RoleAdmin, got.Users[0].Role)
	assert.False(t, got.Users[1].Banned)
}

func TestListUsers_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.userRepo.EXPECT().ListAll(mock.Anything, "", 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListUsers(context.Background(), "", bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestGetUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	m.userRepo.EXPECT().GetProfileByID(mock.Anything, uid).Return(&model.User{
		ID:                     uid,
		Username:               "a",
		Email:                  "a@example.com",
		IP:                     new("127.0.0.1"),
		BannedAt:               new("2026-01-01"),
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
	assert.Equal(t, "a@example.com", got.Email)
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

func TestSetUserRole_CannotGrantAtOrAboveOwnRank(t *testing.T) {
	// given
	tests := []struct {
		name      string
		actorRole role.Role
		granted   role.Role
	}{
		{name: "admin cannot mint a super_admin", actorRole: authz.RoleAdmin, granted: authz.RoleSuperAdmin},
		{name: "admin cannot mint another admin", actorRole: authz.RoleAdmin, granted: authz.RoleAdmin},
		{name: "moderator cannot mint another moderator", actorRole: authz.RoleModerator, granted: authz.RoleModerator},
		{name: "super_admin cannot mint another super_admin", actorRole: authz.RoleSuperAdmin, granted: authz.RoleSuperAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, m := newTestService(t)
			actor := uuid.New()
			target := uuid.New()
			m.authz.EXPECT().GetRole(mock.Anything, actor).Return(tt.actorRole, nil)

			// when
			err := svc.SetUserRole(context.Background(), actor, target, tt.granted)

			// then
			assert.ErrorIs(t, err, ErrRoleOutranksActor)
		})
	}
}

func TestSetUserRole_UnknownRoleRejected(t *testing.T) {
	// given
	tests := []struct {
		name    string
		granted role.Role
	}{
		{name: "invented role", granted: role.Role("owner")},
		{name: "empty role", granted: role.Role("")},
		{name: "case mismatch", granted: role.Role("Admin")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			actor := uuid.New()
			target := uuid.New()

			// when
			err := svc.SetUserRole(context.Background(), actor, target, tt.granted)

			// then
			assert.ErrorIs(t, err, ErrUnknownRole)
		})
	}
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
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.roleRepo.EXPECT().SetRole(mock.Anything, target, authz.RoleAdmin).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionSetRole, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "admin", SubjectID: target}).Return(nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, m.chatSync.ensureCalls)
	assert.Equal(t, 1, m.chatSync.syncCalls)
	assert.Equal(t, authz.RoleAdmin, m.chatSync.lastSyncRole)
}

func TestSetUserRole_BotAccountIsProtected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target, IsBot: true}, nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.ErrorIs(t, err, ErrBotAccountProtected)
	m.roleRepo.AssertNotCalled(t, "SetRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestSetUserRole_SetRoleError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
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
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.roleRepo.EXPECT().SetRole(mock.Anything, target, authz.RoleAdmin).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionSetRole, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "admin", SubjectID: target}).Return(nil)

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.NoError(t, err)
}

func TestRemoveUserRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleModerator, nil)
	m.roleRepo.EXPECT().RemoveRole(mock.Anything, target, authz.RoleModerator).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionRemoveRole, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "moderator", SubjectID: target}).Return(nil)

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

func TestSetUserEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target, Email: "old@example.com"}, nil)
	m.authSvc.EXPECT().SetEmailForUser(mock.Anything, target, "new@example.com").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionSetUserEmail, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "old@example.com -> new@example.com", SubjectID: target}).Return(nil)

	// when
	err := svc.SetUserEmail(context.Background(), actor, target, "new@example.com")

	// then
	require.NoError(t, err)
}

func TestSetUserEmail_Protected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleAdmin, nil)

	// when
	err := svc.SetUserEmail(context.Background(), actor, target, "new@example.com")

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestSetUserEmail_AuthErrorPropagates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.authSvc.EXPECT().SetEmailForUser(mock.Anything, target, "nope").Return(auth.ErrInvalidEmail)

	// when
	err := svc.SetUserEmail(context.Background(), actor, target, "nope")

	// then
	assert.ErrorIs(t, err, auth.ErrInvalidEmail)
}

func TestVerifyUserEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.authSvc.EXPECT().MarkEmailVerified(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionVerifyUserEmail, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	err := svc.VerifyUserEmail(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestUnverifyUserEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.authSvc.EXPECT().MarkEmailUnverified(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionUnverifyUserEmail, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	err := svc.UnverifyUserEmail(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestUnverifyUserEmail_AlreadyUnverified(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.authSvc.EXPECT().MarkEmailUnverified(mock.Anything, target).Return(auth.ErrEmailNotVerified)

	// when
	err := svc.UnverifyUserEmail(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, auth.ErrEmailNotVerified)
}

func TestUnverifyUserEmail_Protected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleSuperAdmin, nil)

	// when
	err := svc.UnverifyUserEmail(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestSetUserDisplayName_ClampsAndAudits(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target, DisplayName: "Old Name"}, nil)
	m.userRepo.EXPECT().SetDisplayName(mock.Anything, target, "Beatrice the Golden").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionSetDisplayName, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "Old Name -> Beatrice the Golden", SubjectID: target}).Return(nil)

	// when
	err := svc.SetUserDisplayName(context.Background(), actor, target, "  <b>Beatrice</b>   the Golden  ")

	// then
	require.NoError(t, err)
}

func TestSetUserDisplayName_RejectsEmpty(t *testing.T) {
	// given
	tests := []struct {
		name  string
		input string
	}{
		{name: "blank", input: "   "},
		{name: "markup only", input: "<b></b>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, m := newTestService(t)
			actor := uuid.New()
			target := uuid.New()
			m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
			m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)

			// when
			err := svc.SetUserDisplayName(context.Background(), actor, target, tt.input)

			// then
			assert.ErrorIs(t, err, ErrEmptyDisplayName)
		})
	}
}

func TestSetDisplayNameLocked_AuditsPerDirection(t *testing.T) {
	// given
	tests := []struct {
		name   string
		locked bool
		action repository.AuditAction
	}{
		{name: "lock", locked: true, action: repository.AuditActionLockDisplayName},
		{name: "unlock", locked: false, action: repository.AuditActionUnlockDisplayName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, m := newTestService(t)
			actor := uuid.New()
			target := uuid.New()
			m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
			m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
			m.userRepo.EXPECT().SetDisplayNameLocked(mock.Anything, target, tt.locked).Return(nil)
			m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: tt.action, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

			// when
			err := svc.SetDisplayNameLocked(context.Background(), actor, target, tt.locked)

			// then
			require.NoError(t, err)
		})
	}
}

func TestForceLogout_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionForceLogout, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	err := svc.ForceLogout(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestForceLogout_SessionErrorFails(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(errors.New("boom"))

	// when
	err := svc.ForceLogout(context.Background(), actor, target)

	// then
	require.Error(t, err)
}

func TestListAccountsOnIP_NoIPReturnsEmpty(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)

	// when
	result, err := svc.ListAccountsOnIP(context.Background(), target)

	// then
	require.NoError(t, err)
	assert.Empty(t, result.IP)
	assert.Empty(t, result.Users)
}

func TestListAccountsOnIP_ReturnsSiblings(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	sibling := uuid.New()
	ip := "2a00:23c8:ec30:1001:65c3:a122:a356:90c4"
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target, IP: &ip}, nil)
	m.userRepo.EXPECT().ListByIP(mock.Anything, ip, target).Return([]model.User{
		{ID: sibling, Username: "alt", DisplayName: "Alt"},
	}, nil)

	// when
	result, err := svc.ListAccountsOnIP(context.Background(), target)

	// then
	require.NoError(t, err)
	assert.Equal(t, ip, result.IP)
	require.Len(t, result.Users, 1)
	assert.Equal(t, sibling, result.Users[0].ID)
}

func TestGetUserAuditLog_ScopesToTarget(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.auditRepo.EXPECT().ListForUser(mock.Anything, target, 20, 0).Return([]repository.AuditLogEntry{
		{ID: 1, Action: repository.AuditActionBanUser, TargetType: repository.AuditTargetUser, TargetID: target.String()},
	}, 1, nil)

	// when
	result, err := svc.GetUserAuditLog(context.Background(), target, bounds.NewPage(20, 0))

	// then
	require.NoError(t, err)
	require.Len(t, result.Entries, 1)
	assert.Equal(t, "ban_user", result.Entries[0].Action)
	assert.Equal(t, 1, result.Total)
}

func TestBanUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().BanUser(mock.Anything, target, actor, "reason").Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionBanUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "reason", SubjectID: target}).Return(nil)

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
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().BanUser(mock.Anything, target, actor, "reason").Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(errors.New("session boom"))
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionBanUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "reason", SubjectID: target}).Return(nil)

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
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().BanUser(mock.Anything, target, actor, "r").Return(errors.New("boom"))

	// when
	err := svc.BanUser(context.Background(), actor, target, "r")

	// then
	require.Error(t, err)
}

func TestUnbanUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().UnbanUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionUnbanUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

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
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().UnbanUser(mock.Anything, target).Return(errors.New("boom"))

	// when
	err := svc.UnbanUser(context.Background(), actor, target)

	// then
	require.Error(t, err)
}

func TestUnbanUser_ProtectedOutranksActor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleAdmin, nil)

	// when
	err := svc.UnbanUser(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestUnlockUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().UnlockUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionUnlockUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	err := svc.UnlockUser(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestUnlockUser_ProtectedOutranksActor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleSuperAdmin, nil)

	// when
	err := svc.UnlockUser(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestDeleteUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{
		ID:        target,
		Username:  "beatrice",
		AvatarURL: "/a.png",
		BannerURL: "/b.png",
	}, nil)
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().AdminDeleteAccount(mock.Anything, target).Return(nil)
	m.uploadSvc.EXPECT().Delete([]string{"/a.png"}).Return()
	m.uploadSvc.EXPECT().Delete([]string{"/b.png"}).Return()
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionDeleteUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "username=beatrice"}).Return(nil)

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	require.NoError(t, err)
}

func TestDeleteUser_BotAccountIsProtected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target, IsBot: true}, nil)
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	require.ErrorIs(t, err, ErrBotAccountProtected)
	m.userRepo.AssertNotCalled(t, "AdminDeleteAccount", mock.Anything, mock.Anything)
}

func TestDeleteUser_UserLookupFailsStillDeletes(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(nil, errors.New("not found"))
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().AdminDeleteAccount(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionDeleteUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: ""}).Return(nil)

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
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().AdminDeleteAccount(mock.Anything, target).Return(errors.New("boom"))

	// when
	err := svc.DeleteUser(context.Background(), actor, target)

	// then
	require.Error(t, err)
}

func TestResetUserPassword_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().SetPasswordHash(mock.Anything, target, mock.Anything).Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionResetPassword, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	password, err := svc.ResetUserPassword(context.Background(), actor, target)

	// then
	require.NoError(t, err)
	assert.Len(t, password, 16)
}

func TestResetUserPassword_SessionDeleteErrorSwallowed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().SetPasswordHash(mock.Anything, target, mock.Anything).Return(nil)
	m.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, target).Return(errors.New("session boom"))
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionResetPassword, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	password, err := svc.ResetUserPassword(context.Background(), actor, target)

	// then
	require.NoError(t, err)
	assert.Len(t, password, 16)
}

func TestResetUserPassword_Protected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleSuperAdmin, nil)

	// when
	password, err := svc.ResetUserPassword(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
	assert.Empty(t, password)
}

func TestResetUserPassword_SetPasswordError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleSuperAdmin, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().SetPasswordHash(mock.Anything, target, mock.Anything).Return(errors.New("boom"))

	// when
	password, err := svc.ResetUserPassword(context.Background(), actor, target)

	// then
	require.Error(t, err)
	assert.Empty(t, password)
}

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

func TestUpdateSettings_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{"site_name": "old"})
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry repository.NewAuditEntry) bool {
		return entry.ActorID == actor && entry.Action == repository.AuditActionUpdateSettings && entry.TargetType == repository.AuditTargetSettings && entry.TargetID == "site_name"
	})).Return(nil)

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"site_name": "umineko"})

	// then
	require.NoError(t, err)
}

func TestUpdateSettings_Error(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{})
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(errors.New("boom"))

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"site_name": "umineko"})

	// then
	require.Error(t, err)
}

func TestGetSettings_MasksSecrets(t *testing.T) {
	tests := []struct {
		name   string
		key    config.SiteSettingKey
		stored string
		want   string
	}{
		{name: "smtp password is masked", key: config.SettingSMTPPassword.Key, stored: "hunter2", want: config.SecretMask},
		{name: "livekit api secret is masked", key: config.SettingLiveKitAPISecret.Key, stored: "lk-secret", want: config.SecretMask},
		{name: "cloudflare api token is masked", key: config.SettingCloudflareAPIToken.Key, stored: "cf-token", want: config.SecretMask},
		{name: "turnstile secret key is masked", key: config.SettingTurnstileSecretKey.Key, stored: "ts-secret", want: config.SecretMask},
		{name: "sentry dsn is masked", key: config.SettingSentryDSN.Key, stored: "https://abc@sentry.example/1", want: config.SecretMask},
		{name: "valkey url is masked", key: config.SettingValkeyURL.Key, stored: "redis://user:pw@valkey:6379", want: config.SecretMask},
		{name: "unset secret stays empty", key: config.SettingSMTPPassword.Key, stored: "", want: ""},
		{name: "site name is returned verbatim", key: config.SettingSiteName.Key, stored: "City of Books", want: "City of Books"},
		{name: "smtp username is returned verbatim", key: config.SettingSMTPUsername.Key, stored: "postmaster", want: "postmaster"},
		{name: "livekit api key is returned verbatim", key: config.SettingLiveKitAPIKey.Key, stored: "lk-key", want: "lk-key"},
		{name: "turnstile site key is returned verbatim", key: config.SettingTurnstileSiteKey.Key, stored: "ts-site", want: "ts-site"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{tc.key: tc.stored})

			// when
			got, err := svc.GetSettings(context.Background())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Settings[string(tc.key)])
		})
	}
}

func TestUpdateSettings_SecretsAndAudit(t *testing.T) {
	tests := []struct {
		name         string
		stored       map[config.SiteSettingKey]string
		submitted    map[string]string
		wantStored   map[config.SiteSettingKey]string
		wantTargetID string
	}{
		{
			name:         "masked secret is left untouched",
			stored:       map[config.SiteSettingKey]string{"smtp_password": "hunter2", "site_name": "old"},
			submitted:    map[string]string{"smtp_password": config.SecretMask, "site_name": "new"},
			wantStored:   map[config.SiteSettingKey]string{"site_name": "new"},
			wantTargetID: "site_name",
		},
		{
			name:         "real secret value replaces the stored one",
			stored:       map[config.SiteSettingKey]string{"smtp_password": "hunter2"},
			submitted:    map[string]string{"smtp_password": "correct-horse"},
			wantStored:   map[config.SiteSettingKey]string{"smtp_password": "correct-horse"},
			wantTargetID: "smtp_password",
		},
		{
			name:         "mask on a non secret setting is stored verbatim",
			stored:       map[config.SiteSettingKey]string{"site_name": "old"},
			submitted:    map[string]string{"site_name": config.SecretMask},
			wantStored:   map[config.SiteSettingKey]string{"site_name": config.SecretMask},
			wantTargetID: "site_name",
		},
		{
			name:         "audit names every changed key in order",
			stored:       map[config.SiteSettingKey]string{"site_name": "old", "smtp_host": "mail", "base_url": "http://a"},
			submitted:    map[string]string{"site_name": "new", "smtp_host": "mail", "base_url": "http://b"},
			wantStored:   map[config.SiteSettingKey]string{"site_name": "new", "smtp_host": "mail", "base_url": "http://b"},
			wantTargetID: "base_url,site_name",
		},
		{
			name:         "unchanged submission names no keys",
			stored:       map[config.SiteSettingKey]string{"site_name": "old"},
			submitted:    map[string]string{"site_name": "old"},
			wantStored:   map[config.SiteSettingKey]string{"site_name": "old"},
			wantTargetID: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			actor := uuid.New()
			m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(tc.stored)
			m.settingsSvc.EXPECT().SetMultiple(mock.Anything, tc.wantStored, actor).Return(nil)
			m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry repository.NewAuditEntry) bool {
				return entry.ActorID == actor && entry.Action == repository.AuditActionUpdateSettings && entry.TargetType == repository.AuditTargetSettings && entry.TargetID == tc.wantTargetID
			})).Return(nil)

			// when
			err := svc.UpdateSettings(context.Background(), actor, tc.submitted)

			// then
			require.NoError(t, err)
		})
	}
}

func TestUpdateSettings_AuditDetailsHideValues(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	var gotDetails string
	m.settingsSvc.EXPECT().GetAll(mock.Anything).Return(map[config.SiteSettingKey]string{"smtp_password": "hunter2"})
	m.settingsSvc.EXPECT().SetMultiple(mock.Anything, mock.Anything, actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry repository.NewAuditEntry) bool {
		return entry.ActorID == actor && entry.Action == repository.AuditActionUpdateSettings && entry.TargetType == repository.AuditTargetSettings && entry.TargetID == "smtp_password"
	})).
		Run(func(_ context.Context, entry repository.NewAuditEntry, _ ...*sql.Tx) { gotDetails = entry.Details }).
		Return(nil)

	// when
	err := svc.UpdateSettings(context.Background(), actor, map[string]string{"smtp_password": "correct-horse"})

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, gotDetails)
	assert.NotContains(t, gotDetails, "hunter2")
	assert.NotContains(t, gotDetails, "correct-horse")
}

func TestSendTestEmail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(&model.User{ID: actor, Email: "admin@example.com"}, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().SendTest(mock.Anything, "admin@example.com", mock.Anything, mock.Anything).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionSendTestEmail, TargetType: repository.AuditTargetSettings, TargetID: "", Details: ""}).Return(nil)

	// when
	err := svc.SendTestEmail(context.Background(), actor)

	// then
	require.NoError(t, err)
}

func TestSendTestEmail_NoEmailAddress(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(&model.User{ID: actor, Email: ""}, nil)

	// when
	err := svc.SendTestEmail(context.Background(), actor)

	// then
	require.ErrorIs(t, err, ErrNoEmailAddress)
}

func TestSendTestEmail_SendError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actor).Return(&model.User{ID: actor, Email: "admin@example.com"}, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("City of Books")
	m.emailSvc.EXPECT().SendTest(mock.Anything, "admin@example.com", mock.Anything, mock.Anything).Return(email.ErrNotConfigured)

	// when
	err := svc.SendTestEmail(context.Background(), actor)

	// then
	require.Error(t, err)
}

func TestGetAuditLog_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.auditRepo.EXPECT().List(mock.Anything, repository.AuditActionBanUser, 20, 0).Return([]repository.AuditLogEntry{
		{ID: 1, ActorID: actor, ActorName: "victorique", Action: repository.AuditActionBanUser, TargetType: repository.AuditTargetUser, TargetID: "t", Details: "d", CreatedAt: "now"},
	}, 1, nil)

	// when
	got, err := svc.GetAuditLog(context.Background(), repository.AuditActionBanUser, bounds.NewPage(20, 0))

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
	m.auditRepo.EXPECT().List(mock.Anything, repository.AuditAction(""), 20, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetAuditLog(context.Background(), "", bounds.NewPage(20, 0))

	// then
	require.Error(t, err)
}

func TestCreateInvite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry repository.NewAuditEntry) bool {
		return entry.ActorID == actor && entry.Action == repository.AuditActionCreateInvite && entry.TargetType == repository.AuditTargetInvite && entry.Details == ""
	})).Return(nil)

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

func TestListInvites_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	creator := uuid.New()
	m.inviteRepo.EXPECT().List(mock.Anything, 10, 0).Return([]repository.Invite{
		{Code: "abc", CreatedBy: creator, CreatedAt: "t"},
	}, 1, nil)

	// when
	got, err := svc.ListInvites(context.Background(), bounds.NewPage(10, 0))

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
	_, err := svc.ListInvites(context.Background(), bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestDeleteInvite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.inviteRepo.EXPECT().Delete(mock.Anything, "abc").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionDeleteInvite, TargetType: repository.AuditTargetInvite, TargetID: "abc", Details: ""}).Return(nil)

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
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry repository.NewAuditEntry) bool {
		return entry.ActorID == actor && entry.Action == repository.AuditActionCreateVanityRole && entry.TargetType == repository.AuditTargetVanityRole && entry.Details == ""
	})).Return(nil)

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

func TestUpdateVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	id := "r1"
	m.vanityRepo.EXPECT().GetByID(mock.Anything, id).Return(&repository.VanityRoleRow{ID: id, IsSystem: false}, nil)
	m.vanityRepo.EXPECT().Update(mock.Anything, id, "silver", "#cccccc", 2).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionUpdateVanityRole, TargetType: repository.AuditTargetVanityRole, TargetID: id, Details: ""}).Return(nil)

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

func TestDeleteVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1", IsSystem: false}, nil)
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingChatbotEnabled).Return(false)
	m.vanityRepo.EXPECT().Delete(mock.Anything, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionDeleteVanityRole, TargetType: repository.AuditTargetVanityRole, TargetID: "r1", Details: ""}).Return(nil)

	// when
	err := svc.DeleteVanityRole(context.Background(), actor, "r1")

	// then
	require.NoError(t, err)
}

func TestDeleteVanityRole_RefusedWhileItIsTheOptInRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "patron").Return(&repository.VanityRoleRow{ID: "patron"}, nil)
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingChatbotEnabled).Return(true)
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingChatbotRequirePermission).Return(true)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingChatbotOptInRole).Return("patron")

	// when
	err := svc.DeleteVanityRole(context.Background(), actor, "patron")

	// then
	require.ErrorIs(t, err, ErrVanityRoleOptInLocked)
	m.vanityRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
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
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingChatbotEnabled).Return(false)
	m.vanityRepo.EXPECT().Delete(mock.Anything, "r1").Return(errors.New("boom"))

	// when
	err := svc.DeleteVanityRole(context.Background(), uuid.New(), "r1")

	// then
	require.Error(t, err)
}

func TestGetVanityRoleUsers_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	uid := uuid.New()
	m.vanityRepo.EXPECT().GetUsersForRole(mock.Anything, "r1", "q", 10, 0).Return([]repository.VanityRoleUserRow{
		{UserID: uid, Username: "u", DisplayName: "U", AvatarURL: "/a.png"},
	}, 1, nil)

	// when
	got, err := svc.GetVanityRoleUsers(context.Background(), "r1", "q", bounds.NewPage(10, 0))

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
	_, err := svc.GetVanityRoleUsers(context.Background(), "r1", "", bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestAssignVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, target, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionAssignVanityRole, TargetType: repository.AuditTargetVanityRole, TargetID: "r1", Details: "", SubjectID: target}).Return(nil)

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
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, target, "r1").Return(errors.New("boom"))

	// when
	err := svc.AssignVanityRole(context.Background(), uuid.New(), "r1", target)

	// then
	require.Error(t, err)
}

func TestUnassignVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().UnassignFromUser(mock.Anything, target, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionUnassignVanityRole, TargetType: repository.AuditTargetVanityRole, TargetID: "r1", Details: "", SubjectID: target}).Return(nil)

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
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().UnassignFromUser(mock.Anything, target, "r1").Return(errors.New("boom"))

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", target)

	// then
	require.Error(t, err)
}
