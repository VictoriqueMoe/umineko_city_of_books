package admin

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListVanityRoles_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().List(mock.Anything).Return([]model.VanityRoleRow{
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
	m.vanityRepo.EXPECT().Create(mock.Anything, mock.Anything).Run(func(_ context.Context, s spec.NewVanityRole, _ ...*sql.Tx) {
		assert.NotEmpty(t, s.ID)
		assert.Equal(t, "gold", s.Label)
		assert.Equal(t, "#ffcc00", s.Color)
		assert.Equal(t, 3, s.SortOrder)
	}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry audit.NewEntry) bool {
		return entry.ActorID == actor && entry.Action == audit.ActionCreateVanityRole && entry.TargetType == audit.TargetVanityRole && entry.Details == ""
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
	m.vanityRepo.EXPECT().Create(mock.Anything, mock.Anything).Run(func(_ context.Context, s spec.NewVanityRole, _ ...*sql.Tx) {
		assert.NotEmpty(t, s.ID)
		assert.Equal(t, "gold", s.Label)
		assert.Equal(t, "#ffcc00", s.Color)
		assert.Equal(t, 0, s.SortOrder)
	}).Return(errors.New("boom"))

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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, id).Return(&model.VanityRoleRow{ID: id, IsSystem: false}, nil)
	m.vanityRepo.EXPECT().Update(mock.Anything, spec.VanityRoleUpdate{ID: id, Label: "silver", Color: "#cccccc", SortOrder: 2}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: actor, Action: audit.ActionUpdateVanityRole, TargetType: audit.TargetVanityRole, TargetID: id, Details: ""}).Return(nil)

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
			m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)

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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)
	m.vanityRepo.EXPECT().Update(mock.Anything, spec.VanityRoleUpdate{ID: "r1", Label: "x", Color: "#000000", SortOrder: 0}).Return(errors.New("boom"))

	// when
	err := svc.UpdateVanityRole(context.Background(), uuid.New(), "r1", dto.UpdateVanityRoleRequest{Label: "x", Color: "#000000"})

	// then
	require.Error(t, err)
}

func TestDeleteVanityRole_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1", IsSystem: false}, nil)
	m.settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingChatbotEnabled).Return(false)
	m.vanityRepo.EXPECT().Delete(mock.Anything, "r1").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: actor, Action: audit.ActionDeleteVanityRole, TargetType: audit.TargetVanityRole, TargetID: "r1", Details: ""}).Return(nil)

	// when
	err := svc.DeleteVanityRole(context.Background(), actor, "r1")

	// then
	require.NoError(t, err)
}

func TestDeleteVanityRole_RefusedWhileItIsTheOptInRole(t *testing.T) {
	// given
	svc, m := newTestService(t)
	actor := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "patron").Return(&model.VanityRoleRow{ID: "patron"}, nil)
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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1", IsSystem: true}, nil)

	// when
	err := svc.DeleteVanityRole(context.Background(), uuid.New(), "r1")

	// then
	assert.ErrorIs(t, err, ErrSystemRole)
}

func TestDeleteVanityRole_DeleteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)
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
	m.vanityRepo.EXPECT().GetUsersForRole(mock.Anything, spec.VanityRoleUserQuery{RoleID: "r1", Search: "q", Limit: 10, Offset: 0}).Return([]model.VanityRoleUserRow{
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
	m.vanityRepo.EXPECT().GetUsersForRole(mock.Anything, spec.VanityRoleUserQuery{RoleID: "r1", Search: "", Limit: 10, Offset: 0}).Return(nil, 0, errors.New("boom"))

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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, spec.VanityRoleAssignment{UserID: target, RoleID: "r1"}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: actor, Action: audit.ActionAssignVanityRole, TargetType: audit.TargetVanityRole, TargetID: "r1", Details: "", SubjectID: target}).Return(nil)

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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1", IsSystem: true}, nil)

	// when
	err := svc.AssignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	assert.ErrorIs(t, err, ErrSystemRole)
}

func TestAssignVanityRole_AssignError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().AssignToUser(mock.Anything, spec.VanityRoleAssignment{UserID: target, RoleID: "r1"}).Return(errors.New("boom"))

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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().UnassignFromUser(mock.Anything, spec.VanityRoleAssignment{UserID: target, RoleID: "r1"}).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: actor, Action: audit.ActionUnassignVanityRole, TargetType: audit.TargetVanityRole, TargetID: "r1", Details: "", SubjectID: target}).Return(nil)

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
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1", IsSystem: true}, nil)

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", uuid.New())

	// then
	assert.ErrorIs(t, err, ErrSystemRole)
}

func TestUnassignVanityRole_UnassignError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	m.vanityRepo.EXPECT().GetByID(mock.Anything, "r1").Return(&model.VanityRoleRow{ID: "r1"}, nil)
	m.permRepo.EXPECT().GetVanityRolePermissions(mock.Anything).Return(nil, nil)
	m.vanityRepo.EXPECT().UnassignFromUser(mock.Anything, spec.VanityRoleAssignment{UserID: target, RoleID: "r1"}).Return(errors.New("boom"))

	// when
	err := svc.UnassignVanityRole(context.Background(), uuid.New(), "r1", target)

	// then
	require.Error(t, err)
}
