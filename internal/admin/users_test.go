package admin

import (
	"context"
	"errors"
	"testing"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingNewAccountHours).Return(24).Maybe()

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
	m.chatSync.EXPECT().EnsureSystemRooms(mock.Anything).Return(nil).Once()
	m.chatSync.EXPECT().SyncSystemRoomMembership(mock.Anything, target, authz.RoleAdmin).Return(nil).Once()

	// when
	err := svc.SetUserRole(context.Background(), actor, target, authz.RoleAdmin)

	// then
	require.NoError(t, err)
	m.chatSync.AssertExpectations(t)
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
	m.chatSync.EXPECT().EnsureSystemRooms(mock.Anything).Return(errors.New("ensure boom")).Once()
	m.chatSync.EXPECT().SyncSystemRoomMembership(mock.Anything, target, authz.RoleAdmin).Return(errors.New("sync boom")).Once()
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
	m.chatSync.EXPECT().SyncSystemRoomMembership(mock.Anything, target, role.Role("")).Return(nil).Once()

	// when
	err := svc.RemoveUserRole(context.Background(), actor, target, authz.RoleModerator)

	// then
	require.NoError(t, err)
	m.chatSync.AssertExpectations(t)
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

func TestApproveUser_OK(t *testing.T) {
	// given a moderator clearing a new member early
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, target).Return(&model.User{ID: target}, nil)
	m.userRepo.EXPECT().ApproveUser(mock.Anything, target, actor).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionApproveUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	err := svc.ApproveUser(context.Background(), actor, target)

	// then the approver is recorded, so the decision is attributable
	require.NoError(t, err)
}

func TestApproveUser_ProtectedOutranksActor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return(authz.RoleSuperAdmin, nil)

	// when
	err := svc.ApproveUser(context.Background(), actor, target)

	// then
	assert.ErrorIs(t, err, ErrProtectedUser)
}

func TestUnapproveUser_OK(t *testing.T) {
	// given a moderator taking an approval back
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.authz.EXPECT().GetRole(mock.Anything, actor).Return(authz.RoleModerator, nil)
	m.authz.EXPECT().GetRole(mock.Anything, target).Return("", nil)
	m.userRepo.EXPECT().UnapproveUser(mock.Anything, target).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{ActorID: actor, Action: repository.AuditActionUnapproveUser, TargetType: repository.AuditTargetUser, TargetID: target.String(), Details: "", SubjectID: target}).Return(nil)

	// when
	err := svc.UnapproveUser(context.Background(), actor, target)

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
