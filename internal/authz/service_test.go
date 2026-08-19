package authz

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testDeps struct {
	roleRepo    *repository.MockRoleRepository
	userRepo    *repository.MockUserRepository
	permRepo    *repository.MockPermissionRepository
	settingsSvc *settings.MockService
}

func newTestService(t *testing.T) (*service, *testDeps) {
	deps := &testDeps{
		roleRepo:    repository.NewMockRoleRepository(t),
		userRepo:    repository.NewMockUserRepository(t),
		permRepo:    repository.NewMockPermissionRepository(t),
		settingsSvc: settings.NewMockService(t),
	}
	svc := NewService(deps.roleRepo, deps.userRepo, deps.permRepo, deps.settingsSvc).(*service)

	return svc, deps
}

func seededModeratorTable() map[string][]string {
	perms := DefaultRolePermissions(RoleModerator)
	stored := make([]string, len(perms))
	for i, p := range perms {
		stored[i] = string(p)
	}

	return map[string][]string{string(RoleModerator): stored}
}

func TestIsBanned_True(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(true, nil)

	// when
	got := svc.IsBanned(context.Background(), userID)

	// then
	assert.True(t, got)
}

func TestIsBanned_False(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, nil)

	// when
	got := svc.IsBanned(context.Background(), userID)

	// then
	assert.False(t, got)
}

func TestIsBanned_RepoErrorTreatsTheAccountAsBanned(t *testing.T) {
	// given a ban lookup that cannot be answered
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, errors.New("db down"))

	// when
	got := svc.IsBanned(context.Background(), userID)

	// then a database failure must never hand access to a banned account
	assert.True(t, got)
}

func TestIsLocked_RepoErrorTreatsTheAccountAsLocked(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.userRepo.EXPECT().IsLocked(mock.Anything, userID).Return(false, errors.New("db down"))

	// when
	got := svc.IsLocked(context.Background(), userID)

	// then
	assert.True(t, got)
}

func TestRequiresEmailVerification_RepoErrorTreatsTheAccountAsUnverified(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.userRepo.EXPECT().RequiresEmailVerification(mock.Anything, userID).Return(false, errors.New("db down"))

	// when
	got := svc.RequiresEmailVerification(context.Background(), userID)

	// then
	assert.True(t, got)
}

func TestCan_NilUserIDDenied(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	got := svc.Can(context.Background(), uuid.Nil, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_RoleRepoErrorDenied(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", errors.New("db down"))

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_PermissionRepoErrorDeniesClosed(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)
	deps.permRepo.EXPECT().GetRolePermissions(mock.Anything).Return(nil, errors.New("db down"))

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_NoRoleAndNoVanityRolesDenied(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_StaffPermissionNeverConsultsVanityRoles(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)

	// when
	got := svc.Can(context.Background(), userID, PermManageVanityRoles)

	// then
	assert.False(t, got)
	deps.permRepo.AssertNotCalled(t, "GetVanityRoleIDsForUser", mock.Anything, mock.Anything)
	deps.permRepo.AssertNotCalled(t, "GetVanityRolePermissions", mock.Anything)
}

func TestCan_UnknownRoleDenied(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("gardener", nil)

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_SuperAdminGrantsEverythingWithoutReadingTables(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleSuperAdmin, nil)

	// when
	got := svc.Can(context.Background(), userID, PermManageSettings)

	// then
	assert.True(t, got)
	deps.permRepo.AssertNotCalled(t, "GetRolePermissions", mock.Anything)
}

func TestCan_AdminGrantsEverythingWithoutReadingTables(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleAdmin, nil)

	// when
	got := svc.Can(context.Background(), userID, PermDeleteAnyUser)

	// then
	assert.True(t, got)
	deps.permRepo.AssertNotCalled(t, "GetRolePermissions", mock.Anything)
}

func TestCan_ModeratorUsesStoredTable(t *testing.T) {
	cases := []struct {
		name string
		perm Permission
		want bool
	}{
		{"view admin panel allowed", PermViewAdminPanel, true},
		{"view stats allowed", PermViewStats, true},
		{"view users allowed", PermViewUsers, true},
		{"delete any theory allowed", PermDeleteAnyTheory, true},
		{"delete any response allowed", PermDeleteAnyResponse, true},
		{"delete any post allowed", PermDeleteAnyPost, true},
		{"delete any comment allowed", PermDeleteAnyComment, true},
		{"edit any theory allowed", PermEditAnyTheory, true},
		{"edit any post allowed", PermEditAnyPost, true},
		{"edit any comment allowed", PermEditAnyComment, true},
		{"ban user allowed", PermBanUser, true},
		{"edit mystery score allowed", PermEditMysteryScore, true},
		{"edit any journal allowed", PermEditAnyJournal, true},
		{"delete any journal allowed", PermDeleteAnyJournal, true},
		{"manage user account allowed", PermManageUserAccount, true},
		{"manage user email denied", PermManageUserEmail, false},
		{"set email verified denied", PermSetEmailVerified, false},
		{"reset password denied", PermResetPassword, false},
		{"delete any user denied", PermDeleteAnyUser, false},
		{"view audit log denied", PermViewAuditLog, false},
		{"resolve suggestion denied", PermResolveSuggestion, false},
		{"manage vanity roles denied", PermManageVanityRoles, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, deps := newTestService(t)
			userID := uuid.New()
			deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)
			deps.permRepo.EXPECT().GetRolePermissions(mock.Anything).Return(seededModeratorTable(), nil)

			// when
			got := svc.Can(context.Background(), userID, tc.perm)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCan_RestrictedPermissionsNeverConsultTheStoredTable(t *testing.T) {
	cases := []struct {
		name string
		perm Permission
	}{
		{"manage settings", PermManageSettings},
		{"manage roles", PermManageRoles},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, deps := newTestService(t)
			userID := uuid.New()
			deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)

			// when
			got := svc.Can(context.Background(), userID, tc.perm)

			// then
			assert.False(t, got, "a restricted permission must never resolve for a non-immutable role")
			deps.permRepo.AssertNotCalled(t, "GetRolePermissions", mock.Anything)
		})
	}
}

func TestCan_UntickedModeratorPermissionIsRevoked(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)
	deps.permRepo.EXPECT().GetRolePermissions(mock.Anything).
		Return(map[string][]string{string(RoleModerator): {string(PermViewAdminPanel)}}, nil)

	// when
	got := svc.Can(context.Background(), userID, PermBanUser)

	// then
	assert.False(t, got)
}

func TestCan_VanityRoleGrantsPermissionToRolelessUser(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
	deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return([]string{"vanity-a"}, nil)
	deps.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).
		Return(map[string][]string{"vanity-a": {string(PermUseChatbot)}}, nil)

	// when
	got := svc.Can(context.Background(), userID, PermUseChatbot)

	// then
	assert.True(t, got)
}

func TestCan_UnionsAcrossSeveralVanityRoles(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
	deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).
		Return([]string{"vanity-a", "vanity-b"}, nil)
	deps.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(map[string][]string{
		"vanity-a": {},
		"vanity-b": {string(PermUseChatbot)},
	}, nil)

	// when
	got := svc.Can(context.Background(), userID, PermUseChatbot)

	// then
	assert.True(t, got)
}

func TestCan_VanityRoleWithoutMatchingPermissionDenied(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
	deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return([]string{"vanity-a"}, nil)
	deps.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).
		Return(map[string][]string{"vanity-a": {}}, nil)

	// when
	got := svc.Can(context.Background(), userID, PermUseChatbot)

	// then
	assert.False(t, got)
}

func TestCan_SmuggledStaffPermissionOnVanityRoleIsIgnored(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)

	// when
	got := svc.Can(context.Background(), userID, PermManageRoles)

	// then
	assert.False(t, got)
	deps.permRepo.AssertNotCalled(t, "GetVanityRolePermissions", mock.Anything)
}

func TestCan_ModeratorFallsBackToVanityRoleForAssignablePermission(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)
	deps.permRepo.EXPECT().GetRolePermissions(mock.Anything).
		Return(map[string][]string{string(RoleModerator): {string(PermViewAdminPanel)}}, nil)
	deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return([]string{"vanity-a"}, nil)
	deps.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).
		Return(map[string][]string{"vanity-a": {string(PermUseChatbot)}}, nil)

	// when
	got := svc.Can(context.Background(), userID, PermUseChatbot)

	// then
	assert.True(t, got)
}

func TestCan_VanityRepoErrorDeniesClosed(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
	deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return(nil, errors.New("db down"))

	// when
	got := svc.Can(context.Background(), userID, PermUseChatbot)

	// then
	assert.False(t, got)
}

func TestEffectivePermissions(t *testing.T) {
	t.Run("nil user has none", func(t *testing.T) {
		// given
		svc, _ := newTestService(t)

		// when
		got := svc.EffectivePermissions(context.Background(), uuid.Nil)

		// then
		assert.Nil(t, got)
	})

	t.Run("admin gets the whole catalogue", func(t *testing.T) {
		// given
		svc, deps := newTestService(t)
		userID := uuid.New()
		deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleAdmin, nil)

		// when
		got := svc.EffectivePermissions(context.Background(), userID)

		// then
		assert.Len(t, got, len(PermissionCatalogue()))
		assert.NotContains(t, got, PermAll)
	})

	t.Run("roleless user gets only vanity grants", func(t *testing.T) {
		// given
		svc, deps := newTestService(t)
		userID := uuid.New()
		deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
		deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return([]string{"vanity-a"}, nil)
		deps.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).
			Return(map[string][]string{"vanity-a": {string(PermUseChatbot)}}, nil)

		// when
		got := svc.EffectivePermissions(context.Background(), userID)

		// then
		assert.Equal(t, []Permission{PermUseChatbot}, got)
	})

	t.Run("a staff permission smuggled onto a vanity role is filtered out", func(t *testing.T) {
		// given
		svc, deps := newTestService(t)
		userID := uuid.New()
		deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)
		deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return([]string{"vanity-a"}, nil)
		deps.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(map[string][]string{
			"vanity-a": {string(PermManageRoles), string(PermUseChatbot)},
		}, nil)

		// when
		got := svc.EffectivePermissions(context.Background(), userID)

		// then
		assert.Equal(t, []Permission{PermUseChatbot}, got)
	})

	t.Run("moderator gets the stored table", func(t *testing.T) {
		// given
		svc, deps := newTestService(t)
		userID := uuid.New()
		deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)
		deps.permRepo.EXPECT().GetRolePermissions(mock.Anything).
			Return(map[string][]string{string(RoleModerator): {string(PermViewAdminPanel), string(PermBanUser)}}, nil)
		deps.permRepo.EXPECT().GetVanityRoleIDsForUser(mock.Anything, userID).Return(nil, nil).Maybe()

		// when
		got := svc.EffectivePermissions(context.Background(), userID)

		// then
		assert.ElementsMatch(t, []Permission{PermViewAdminPanel, PermBanUser}, got)
	})
}

func TestGetRole_OK(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleAdmin, nil)

	// when
	got, err := svc.GetRole(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, got)
}

func TestGetRole_RepoError(t *testing.T) {
	// given
	svc, deps := newTestService(t)
	userID := uuid.New()
	deps.roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", errors.New("boom"))

	// when
	_, err := svc.GetRole(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestIsRestrictedNewAccount(t *testing.T) {
	tests := []struct {
		name  string
		hours int
		user  *model.User
		err   error
		want  bool
	}{
		{name: "a member inside the window is restricted", hours: 24, user: &model.User{CreatedAt: time.Now().Add(-time.Hour).Format(time.RFC3339)}, want: true},
		{name: "a member past the window is not", hours: 24, user: &model.User{CreatedAt: time.Now().Add(-48 * time.Hour).Format(time.RFC3339)}, want: false},
		{name: "brand new staff are exempt", hours: 24, user: &model.User{CreatedAt: time.Now().Format(time.RFC3339), Role: string(role.RoleModerator)}, want: false},
		{name: "a zero threshold disables the rule", hours: 0, user: &model.User{CreatedAt: time.Now().Format(time.RFC3339)}, want: false},
		{name: "a lookup failure does not restrict", hours: 24, user: nil, err: errors.New("db down"), want: false},
		{name: "a missing user does not restrict", hours: 24, user: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, deps := newTestService(t)
			userID := uuid.New()
			deps.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingNewAccountHours).Return(tc.hours)
			if tc.hours > 0 {
				deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(tc.user, tc.err)
			}

			// when
			got := svc.IsRestrictedNewAccount(context.Background(), userID)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
