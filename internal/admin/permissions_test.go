package admin

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateRolePermissions_RejectsImmutableRoles(t *testing.T) {
	cases := []struct {
		name string
		role string
	}{
		{"admin is immutable", string(authz.RoleAdmin)},
		{"super admin is immutable", string(authz.RoleSuperAdmin)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			// when
			err := svc.UpdateRolePermissions(context.Background(), uuid.New(), tc.role, []string{string(authz.PermViewAdminPanel)})

			// then
			require.ErrorIs(t, err, ErrImmutableRole)
			m.permRepo.AssertNotCalled(t, "SetRolePermissions", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateRolePermissions_RejectsUnknownRole(t *testing.T) {
	// given
	svc, m := newTestService(t)

	// when
	err := svc.UpdateRolePermissions(context.Background(), uuid.New(), "gardener", nil)

	// then
	require.ErrorIs(t, err, ErrUnknownRole)
	m.permRepo.AssertNotCalled(t, "SetRolePermissions", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateRolePermissions_RejectsUnknownPermission(t *testing.T) {
	cases := []struct {
		name string
		perm string
	}{
		{"wildcard is not storable", string(authz.PermAll)},
		{"nonsense permission", "teleport"},
		{"empty permission", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			// when
			err := svc.UpdateRolePermissions(context.Background(), uuid.New(), string(authz.RoleModerator), []string{tc.perm})

			// then
			require.ErrorIs(t, err, ErrUnknownPermission)
			m.permRepo.AssertNotCalled(t, "SetRolePermissions", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateRolePermissions_AllowsStaffPermissionsOnModerator(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.permRepo.EXPECT().SetRolePermissions(mock.Anything, string(authz.RoleModerator),
		[]string{string(authz.PermBanUser), string(authz.PermViewAdminPanel)}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "update_role_permissions", "role", string(authz.RoleModerator),
		"ban_user,view_admin_panel").Return(nil)

	// when
	err := svc.UpdateRolePermissions(context.Background(), actor,
		string(authz.RoleModerator),
		[]string{string(authz.PermViewAdminPanel), string(authz.PermBanUser), string(authz.PermViewAdminPanel)})

	// then
	require.NoError(t, err)
}

func TestUpdateRolePermissions_RejectsRestrictedPermissions(t *testing.T) {
	cases := []struct {
		name string
		perm authz.Permission
	}{
		{"manage roles", authz.PermManageRoles},
		{"manage settings", authz.PermManageSettings},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)

			// when
			err := svc.UpdateRolePermissions(context.Background(), uuid.New(), string(authz.RoleModerator),
				[]string{string(authz.PermBanUser), string(tc.perm)})

			// then
			require.ErrorIs(t, err, ErrRestrictedPermission)
			m.permRepo.AssertNotCalled(t, "SetRolePermissions", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestGetPermissionSettings_CatalogueOmitsRestrictedPermissions(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.permRepo.EXPECT().GetRolePermissions(mock.Anything).Return(map[string][]string{}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(map[string][]string{}, nil)
	m.vanityRepo.EXPECT().List(mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetPermissionSettings(context.Background())

	// then
	require.NoError(t, err)

	names := make([]string, 0, len(got.Permissions))
	for _, item := range got.Permissions {
		names = append(names, item.Permission)
	}

	assert.NotContains(t, names, string(authz.PermManageRoles))
	assert.NotContains(t, names, string(authz.PermManageSettings))
	assert.Contains(t, names, string(authz.PermBanUser))
}

func TestUpdateRolePermissions_UnticksEverything(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.permRepo.EXPECT().SetRolePermissions(mock.Anything, string(authz.RoleModerator), []string{}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "update_role_permissions", "role", string(authz.RoleModerator), "").Return(nil)

	// when
	err := svc.UpdateRolePermissions(context.Background(), actor, string(authz.RoleModerator), nil)

	// then
	require.NoError(t, err)
}

func TestUpdateVanityRolePermissions_RejectsStaffPermissions(t *testing.T) {
	cases := []struct {
		name string
		perm authz.Permission
	}{
		{"manage vanity roles", authz.PermManageVanityRoles},
		{"ban user", authz.PermBanUser},
		{"view admin panel", authz.PermViewAdminPanel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)

			// when
			err := svc.UpdateVanityRolePermissions(context.Background(), uuid.New(), "r1", []string{string(tc.perm)})

			// then
			require.ErrorIs(t, err, ErrStaffPermission)
			m.permRepo.AssertNotCalled(t, "SetVanityRolePermissions", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateVanityRolePermissions_RejectsSystemVanityRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "system_top_detective").
		Return(&repository.VanityRoleRow{ID: "system_top_detective", IsSystem: true}, nil)

	// when
	err := svc.UpdateVanityRolePermissions(context.Background(), uuid.New(), "system_top_detective", []string{string(authz.PermUseChatbot)})

	// then
	require.ErrorIs(t, err, ErrSystemRole)
	m.permRepo.AssertNotCalled(t, "SetVanityRolePermissions", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateVanityRolePermissions_RejectsMissingRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "nope").Return(nil, nil)

	// when
	err := svc.UpdateVanityRolePermissions(context.Background(), uuid.New(), "nope", nil)

	// then
	require.ErrorIs(t, err, ErrVanityRoleNotFound)
}

func TestUpdateVanityRolePermissions_AcceptsAssignableSubset(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().SetVanityRolePermissions(mock.Anything, "r1", []string{string(authz.PermUseChatbot)}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, actor, "update_vanity_role_permissions", "vanity_role", "r1", "use_chatbot").Return(nil)

	// when
	err := svc.UpdateVanityRolePermissions(context.Background(), actor, "r1", []string{string(authz.PermUseChatbot)})

	// then
	require.NoError(t, err)
}

func TestGetPermissionSettings_NeverExposesImmutableRoles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.permRepo.EXPECT().GetRolePermissions(mock.Anything).
		Return(map[string][]string{string(authz.RoleModerator): {string(authz.PermViewAdminPanel)}}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).
		Return(map[string][]string{"r1": {string(authz.PermUseChatbot)}}, nil)
	m.vanityRepo.EXPECT().List(mock.Anything).Return([]repository.VanityRoleRow{
		{ID: "r1", Label: "Chatbot User", Color: "#ff0000", SortOrder: 1},
		{ID: "system_top_detective", Label: "Top Detective", IsSystem: true},
	}, nil)

	// when
	got, err := svc.GetPermissionSettings(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, got.Roles, 1)
	assert.Equal(t, string(authz.RoleModerator), got.Roles[0].Role)
	for _, r := range got.Roles {
		assert.NotEqual(t, string(authz.RoleAdmin), r.Role)
		assert.NotEqual(t, string(authz.RoleSuperAdmin), r.Role)
	}

	require.Len(t, got.VanityRoles, 1)
	assert.Equal(t, "r1", got.VanityRoles[0].ID)
	assert.Equal(t, []string{string(authz.PermUseChatbot)}, got.VanityRoles[0].Permissions)
}

func TestGetPermissionSettings_MarksAssignablePermissions(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.permRepo.EXPECT().GetRolePermissions(mock.Anything).Return(nil, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().List(mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetPermissionSettings(context.Background())

	// then
	require.NoError(t, err)
	require.NotEmpty(t, got.Permissions)

	assignable := make(map[string]bool, len(got.Permissions))
	for _, p := range got.Permissions {
		assignable[p.Permission] = p.VanityAssignable
		assert.NotEqual(t, string(authz.PermAll), p.Permission)
	}

	assert.True(t, assignable[string(authz.PermUseChatbot)])
	assert.False(t, assignable[string(authz.PermManageVanityRoles)])
	assert.False(t, assignable[string(authz.PermManageRoles)])
}

func TestGetPermissionSettings_RepoErrors(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.permRepo.EXPECT().GetRolePermissions(mock.Anything).Return(nil, errors.New("db down"))

	// when
	_, err := svc.GetPermissionSettings(context.Background())

	// then
	require.Error(t, err)
}

func TestAssignVanityRole_PermissionCarryingRoleObeysRankGate(t *testing.T) {
	cases := []struct {
		name       string
		actorRole  string
		targetRole string
		wantErr    error
	}{
		{"admin grants to ordinary user", string(authz.RoleAdmin), "", nil},
		{"moderator grants to ordinary user", string(authz.RoleModerator), "", nil},
		{"moderator cannot grant to a peer", string(authz.RoleModerator), string(authz.RoleModerator), ErrProtectedUser},
		{"admin cannot grant to a peer", string(authz.RoleAdmin), string(authz.RoleAdmin), ErrProtectedUser},
		{"nobody can grant to a super admin", string(authz.RoleAdmin), string(authz.RoleSuperAdmin), ErrProtectedUser},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			actor := uuid.New()
			target := uuid.New()
			m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
			m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).
				Return(map[string][]string{"r1": {string(authz.PermUseChatbot)}}, nil)
			m.authz.EXPECT().GetRole(mock.Anything, actor).Return(role.Role(tc.actorRole), nil)
			m.authz.EXPECT().GetRole(mock.Anything, target).Return(role.Role(tc.targetRole), nil)

			if tc.wantErr == nil {
				m.vanityRepo.EXPECT().AssignToUser(mock.Anything, target, "r1").Return(nil)
				m.auditRepo.EXPECT().CreateForSubject(mock.Anything, actor, "assign_vanity_role", "vanity_role", "r1", "", target).Return(nil)
			}

			// when
			err := svc.AssignVanityRole(context.Background(), actor, "r1", target)

			// then
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tc.wantErr)
			m.vanityRepo.AssertNotCalled(t, "AssignToUser", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestAssignVanityRole_DecorativeRoleSkipsRankGate(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&repository.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(map[string][]string{}, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, target, "r1").Return(nil)
	m.auditRepo.EXPECT().CreateForSubject(mock.Anything, actor, "assign_vanity_role", "vanity_role", "r1", "", target).Return(nil)

	// when
	err := svc.AssignVanityRole(context.Background(), actor, "r1", target)

	// then
	require.NoError(t, err)
	m.authz.AssertNotCalled(t, "GetRole", mock.Anything, mock.Anything)
}
