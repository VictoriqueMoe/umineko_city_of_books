package authz

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*service, *repository.MockRoleRepository, *repository.MockUserRepository) {
	roleRepo := repository.NewMockRoleRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	svc := NewService(roleRepo, userRepo).(*service)
	return svc, roleRepo, userRepo
}

func TestIsBanned_True(t *testing.T) {
	// given
	svc, _, userRepo := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(true, nil)

	// when
	got := svc.IsBanned(context.Background(), userID)

	// then
	assert.True(t, got)
}

func TestIsBanned_False(t *testing.T) {
	// given
	svc, _, userRepo := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, nil)

	// when
	got := svc.IsBanned(context.Background(), userID)

	// then
	assert.False(t, got)
}

func TestIsBanned_RepoErrorReturnsFalse(t *testing.T) {
	// given
	svc, _, userRepo := newTestService(t)
	userID := uuid.New()
	userRepo.EXPECT().IsBanned(mock.Anything, userID).Return(false, errors.New("db down"))

	// when
	got := svc.IsBanned(context.Background(), userID)

	// then
	assert.False(t, got)
}

func TestCan_NilUserIDDenied(t *testing.T) {
	// given
	svc, _, _ := newTestService(t)

	// when
	got := svc.Can(context.Background(), uuid.Nil, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_RepoErrorDenied(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", errors.New("db down"))

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_NoRoleDenied(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", nil)

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_UnknownRoleDenied(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("gardener", nil)

	// when
	got := svc.Can(context.Background(), userID, PermViewAdminPanel)

	// then
	assert.False(t, got)
}

func TestCan_SuperAdminGrantsEverything(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleSuperAdmin, nil)

	// when
	got := svc.Can(context.Background(), userID, PermManageSettings)

	// then
	assert.True(t, got)
}

func TestCan_AdminGrantsEverything(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleAdmin, nil)

	// when
	got := svc.Can(context.Background(), userID, PermDeleteAnyUser)

	// then
	assert.True(t, got)
}

func TestCan_ModeratorHasSpecificPerms(t *testing.T) {
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
		{"manage settings denied", PermManageSettings, false},
		{"manage roles denied", PermManageRoles, false},
		{"delete any user denied", PermDeleteAnyUser, false},
		{"view audit log denied", PermViewAuditLog, false},
		{"resolve suggestion denied", PermResolveSuggestion, false},
		{"manage vanity roles denied", PermManageVanityRoles, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, roleRepo, _ := newTestService(t)
			userID := uuid.New()
			roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleModerator, nil)

			// when
			got := svc.Can(context.Background(), userID, tc.perm)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetRole_OK(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return(RoleAdmin, nil)

	// when
	got, err := svc.GetRole(context.Background(), userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, got)
}

func TestGetRole_RepoError(t *testing.T) {
	// given
	svc, roleRepo, _ := newTestService(t)
	userID := uuid.New()
	roleRepo.EXPECT().GetRole(mock.Anything, userID).Return("", errors.New("boom"))

	// when
	_, err := svc.GetRole(context.Background(), userID)

	// then
	require.Error(t, err)
}
