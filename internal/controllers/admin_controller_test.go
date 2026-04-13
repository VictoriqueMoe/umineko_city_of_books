package controllers

import (
	"errors"
	"net/http"
	"testing"

	adminsvc "umineko_city_of_books/internal/admin"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newAdminHarness(t *testing.T) (*testutil.Harness, *adminsvc.MockService, *repository.MockUserRepository) {
	h := testutil.NewHarness(t)
	ms := adminsvc.NewMockService(t)
	ur := repository.NewMockUserRepository(t)

	s := &Service{
		AdminService: ms,
		UserRepo:     ur,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllAdminRoutes() {
		setup(h.App)
	}
	return h, ms, ur
}

func adminFactory(t *testing.T) (*testutil.Harness, *adminsvc.MockService) {
	h, ms, _ := newAdminHarness(t)
	return h, ms
}

func TestAdminGetStats_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/stats", nil, authz.PermViewStats)
}

func TestAdminGetStats_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewStats, true)
	expected := &dto.AdminStatsResponse{TotalUsers: 42}
	ms.EXPECT().GetStats(mock.Anything).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/stats").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.AdminStatsResponse](t, body)
	assert.Equal(t, 42, got.TotalUsers)
}

func TestAdminGetStats_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewStats, true)
	ms.EXPECT().GetStats(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/stats").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminListUsers_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/users", nil, authz.PermViewUsers)
}

func TestAdminListUsers_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	expected := &dto.AdminUserListResponse{Total: 5, Limit: 20, Offset: 0}
	ms.EXPECT().ListUsers(mock.Anything, "", 20, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/users").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.AdminUserListResponse](t, body)
	assert.Equal(t, 5, got.Total)
}

func TestAdminListUsers_CustomQuery(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	ms.EXPECT().ListUsers(mock.Anything, "beato", 50, 10).Return(&dto.AdminUserListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/admin/users?search=beato&limit=50&offset=10").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminListUsers_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	ms.EXPECT().ListUsers(mock.Anything, "", 20, 0).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/users").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminGetUser_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/users/"+uuid.NewString(), nil, authz.PermViewUsers)
}

func TestAdminGetUser_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)

	// when
	status, body := h.NewRequest("GET", "/admin/users/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminGetUser_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewUsers, true)
	expected := &dto.AdminUserDetailResponse{AdminUserItem: dto.AdminUserItem{ID: targetID, Username: "beato"}}
	ms.EXPECT().GetUser(mock.Anything, targetID).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/users/"+targetID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.AdminUserDetailResponse](t, body)
	assert.Equal(t, "beato", got.Username)
}

func TestAdminGetUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"user not found", adminsvc.ErrUserNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermViewUsers, true)
			ms.EXPECT().GetUser(mock.Anything, targetID).Return(nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("GET", "/admin/users/"+targetID.String()).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminSetRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "POST", "/admin/users/"+uuid.NewString()+"/role", dto.SetRoleRequest{Role: "admin"}, authz.PermManageRoles)
}

func TestAdminSetRole_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)

	// when
	status, body := h.NewRequest("POST", "/admin/users/not-a-uuid/role").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SetRoleRequest{Role: "admin"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminSetRole_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)

	// when
	status, _ := h.NewRequest("POST", "/admin/users/"+uuid.NewString()+"/role").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminSetRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().SetUserRole(mock.Anything, userID, targetID, role.Role("admin")).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/users/"+targetID.String()+"/role").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SetRoleRequest{Role: "admin"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminSetRole_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"user not found", adminsvc.ErrUserNotFound, http.StatusNotFound},
		{"protected user", adminsvc.ErrProtectedUser, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageRoles, true)
			ms.EXPECT().SetUserRole(mock.Anything, userID, targetID, role.Role("admin")).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/admin/users/"+targetID.String()+"/role").
				WithCookie("valid-cookie").
				WithJSONBody(dto.SetRoleRequest{Role: "admin"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminRemoveRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "DELETE", "/admin/users/"+uuid.NewString()+"/role", dto.SetRoleRequest{Role: "admin"}, authz.PermManageRoles)
}

func TestAdminRemoveRole_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)

	// when
	status, body := h.NewRequest("DELETE", "/admin/users/not-a-uuid/role").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SetRoleRequest{Role: "admin"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminRemoveRole_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/users/"+uuid.NewString()+"/role").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminRemoveRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().RemoveUserRole(mock.Anything, userID, targetID, role.Role("admin")).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/users/"+targetID.String()+"/role").
		WithCookie("valid-cookie").
		WithJSONBody(dto.SetRoleRequest{Role: "admin"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminRemoveRole_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"user not found", adminsvc.ErrUserNotFound, http.StatusNotFound},
		{"protected user", adminsvc.ErrProtectedUser, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageRoles, true)
			ms.EXPECT().RemoveUserRole(mock.Anything, userID, targetID, role.Role("admin")).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("DELETE", "/admin/users/"+targetID.String()+"/role").
				WithCookie("valid-cookie").
				WithJSONBody(dto.SetRoleRequest{Role: "admin"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminBanUser_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "POST", "/admin/users/"+uuid.NewString()+"/ban", dto.BanUserRequest{Reason: "spam"}, authz.PermBanUser)
}

func TestAdminBanUser_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermBanUser, true)

	// when
	status, body := h.NewRequest("POST", "/admin/users/not-a-uuid/ban").
		WithCookie("valid-cookie").
		WithJSONBody(dto.BanUserRequest{Reason: "spam"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminBanUser_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermBanUser, true)

	// when
	status, _ := h.NewRequest("POST", "/admin/users/"+uuid.NewString()+"/ban").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminBanUser_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermBanUser, true)
	ms.EXPECT().BanUser(mock.Anything, userID, targetID, "spam").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/users/"+targetID.String()+"/ban").
		WithCookie("valid-cookie").
		WithJSONBody(dto.BanUserRequest{Reason: "spam"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminBanUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"user not found", adminsvc.ErrUserNotFound, http.StatusNotFound},
		{"protected user", adminsvc.ErrProtectedUser, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermBanUser, true)
			ms.EXPECT().BanUser(mock.Anything, userID, targetID, "spam").Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/admin/users/"+targetID.String()+"/ban").
				WithCookie("valid-cookie").
				WithJSONBody(dto.BanUserRequest{Reason: "spam"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminUnbanUser_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "POST", "/admin/users/"+uuid.NewString()+"/unban", nil, authz.PermBanUser)
}

func TestAdminUnbanUser_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermBanUser, true)

	// when
	status, body := h.NewRequest("POST", "/admin/users/not-a-uuid/unban").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminUnbanUser_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermBanUser, true)
	ms.EXPECT().UnbanUser(mock.Anything, userID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/users/"+targetID.String()+"/unban").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminUnbanUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"user not found", adminsvc.ErrUserNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermBanUser, true)
			ms.EXPECT().UnbanUser(mock.Anything, userID, targetID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/admin/users/"+targetID.String()+"/unban").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminDeleteUser_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "DELETE", "/admin/users/"+uuid.NewString(), nil, authz.PermDeleteAnyUser)
}

func TestAdminDeleteUser_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermDeleteAnyUser, true)

	// when
	status, body := h.NewRequest("DELETE", "/admin/users/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminDeleteUser_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermDeleteAnyUser, true)
	ms.EXPECT().DeleteUser(mock.Anything, userID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/users/"+targetID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminDeleteUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"user not found", adminsvc.ErrUserNotFound, http.StatusNotFound},
		{"protected user", adminsvc.ErrProtectedUser, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermDeleteAnyUser, true)
			ms.EXPECT().DeleteUser(mock.Anything, userID, targetID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("DELETE", "/admin/users/"+targetID.String()).WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminGetSettings_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/settings", nil, authz.PermManageSettings)
}

func TestAdminGetSettings_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	expected := &dto.SettingsResponse{Settings: map[string]string{"site_name": "umineko"}}
	ms.EXPECT().GetSettings(mock.Anything).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/settings").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SettingsResponse](t, body)
	assert.Equal(t, "umineko", got.Settings["site_name"])
}

func TestAdminGetSettings_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	ms.EXPECT().GetSettings(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/settings").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminUpdateSettings_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "PUT", "/admin/settings", dto.UpdateSettingsRequest{Settings: map[string]string{}}, authz.PermManageSettings)
}

func TestAdminUpdateSettings_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, _ := h.NewRequest("PUT", "/admin/settings").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminUpdateSettings_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	settings := map[string]string{"site_name": "umineko"}
	ms.EXPECT().UpdateSettings(mock.Anything, userID, settings).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/admin/settings").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateSettingsRequest{Settings: settings}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminUpdateSettings_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	settings := map[string]string{"x": "y"}
	ms.EXPECT().UpdateSettings(mock.Anything, userID, settings).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("PUT", "/admin/settings").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateSettingsRequest{Settings: settings}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminGetAuditLog_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/audit-log", nil, authz.PermViewAuditLog)
}

func TestAdminGetAuditLog_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewAuditLog, true)
	expected := &dto.AuditLogListResponse{Total: 3, Limit: 50, Offset: 0}
	ms.EXPECT().GetAuditLog(mock.Anything, "", 50, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/audit-log").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.AuditLogListResponse](t, body)
	assert.Equal(t, 3, got.Total)
}

func TestAdminGetAuditLog_CustomQuery(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewAuditLog, true)
	ms.EXPECT().GetAuditLog(mock.Anything, "ban_user", 10, 20).Return(&dto.AuditLogListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/admin/audit-log?action=ban_user&limit=10&offset=20").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminGetAuditLog_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermViewAuditLog, true)
	ms.EXPECT().GetAuditLog(mock.Anything, "", 50, 0).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/audit-log").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminCreateInvite_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "POST", "/admin/invites", nil, authz.PermManageRoles)
}

func TestAdminCreateInvite_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	expected := &dto.InviteResponse{Code: "abcd1234", CreatedBy: userID, CreatedAt: "just now"}
	ms.EXPECT().CreateInvite(mock.Anything, userID).Return(expected, nil)

	// when
	status, body := h.NewRequest("POST", "/admin/invites").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[dto.InviteResponse](t, body)
	assert.Equal(t, "abcd1234", got.Code)
}

func TestAdminCreateInvite_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().CreateInvite(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("POST", "/admin/invites").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminListInvites_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/invites", nil, authz.PermManageRoles)
}

func TestAdminListInvites_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	expected := &dto.InviteListResponse{Total: 7, Limit: 50, Offset: 0}
	ms.EXPECT().ListInvites(mock.Anything, 50, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/invites").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.InviteListResponse](t, body)
	assert.Equal(t, 7, got.Total)
}

func TestAdminListInvites_CustomQuery(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().ListInvites(mock.Anything, 5, 15).Return(&dto.InviteListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/admin/invites?limit=5&offset=15").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminListInvites_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().ListInvites(mock.Anything, 50, 0).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/invites").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminDeleteInvite_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "DELETE", "/admin/invites/abc123", nil, authz.PermManageRoles)
}

func TestAdminDeleteInvite_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().DeleteInvite(mock.Anything, userID, "abc123").Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/invites/abc123").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminDeleteInvite_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageRoles, true)
	ms.EXPECT().DeleteInvite(mock.Anything, userID, "abc123").Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/admin/invites/abc123").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminUpdateMysteryScore_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "PUT", "/admin/users/"+uuid.NewString()+"/mystery-score", map[string]int{"desired_score": 100}, authz.PermEditMysteryScore)
}

func TestAdminUpdateMysteryScore_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)

	// when
	status, body := h.NewRequest("PUT", "/admin/users/not-a-uuid/mystery-score").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]int{"desired_score": 100}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminUpdateMysteryScore_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)

	// when
	status, body := h.NewRequest("PUT", "/admin/users/"+uuid.NewString()+"/mystery-score").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestAdminUpdateMysteryScore_OK(t *testing.T) {
	// given
	h, _, ur := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)
	ur.EXPECT().GetDetectiveRawScore(mock.Anything, targetID).Return(30, nil)
	ur.EXPECT().UpdateMysteryScoreAdjustment(mock.Anything, targetID, 70).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/admin/users/"+targetID.String()+"/mystery-score").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]int{"desired_score": 100}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestAdminUpdateMysteryScore_UpdateFails(t *testing.T) {
	// given
	h, _, ur := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)
	ur.EXPECT().GetDetectiveRawScore(mock.Anything, targetID).Return(0, nil)
	ur.EXPECT().UpdateMysteryScoreAdjustment(mock.Anything, targetID, 100).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("PUT", "/admin/users/"+targetID.String()+"/mystery-score").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]int{"desired_score": 100}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to update")
}

func TestAdminUpdateGMScore_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "PUT", "/admin/users/"+uuid.NewString()+"/gm-score", map[string]int{"desired_score": 50}, authz.PermEditMysteryScore)
}

func TestAdminUpdateGMScore_InvalidID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)

	// when
	status, body := h.NewRequest("PUT", "/admin/users/not-a-uuid/gm-score").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]int{"desired_score": 50}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAdminUpdateGMScore_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)

	// when
	status, body := h.NewRequest("PUT", "/admin/users/"+uuid.NewString()+"/gm-score").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestAdminUpdateGMScore_OK(t *testing.T) {
	// given
	h, _, ur := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)
	ur.EXPECT().GetGMRawScore(mock.Anything, targetID).Return(10, nil)
	ur.EXPECT().UpdateGMScoreAdjustment(mock.Anything, targetID, 40).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/admin/users/"+targetID.String()+"/gm-score").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]int{"desired_score": 50}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestAdminUpdateGMScore_UpdateFails(t *testing.T) {
	// given
	h, _, ur := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditMysteryScore, true)
	ur.EXPECT().GetGMRawScore(mock.Anything, targetID).Return(0, nil)
	ur.EXPECT().UpdateGMScoreAdjustment(mock.Anything, targetID, 50).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("PUT", "/admin/users/"+targetID.String()+"/gm-score").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]int{"desired_score": 50}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to update")
}

func TestAdminListVanityRoles_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/vanity-roles", nil, authz.PermManageVanityRoles)
}

func TestAdminListVanityRoles_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	expected := []dto.VanityRoleResponse{{ID: "r1", Label: "VIP", Color: "#ff0000"}}
	ms.EXPECT().ListVanityRoles(mock.Anything).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/vanity-roles").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[[]dto.VanityRoleResponse](t, body)
	require.Len(t, got, 1)
	assert.Equal(t, "VIP", got[0].Label)
}

func TestAdminListVanityRoles_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	ms.EXPECT().ListVanityRoles(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/vanity-roles").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminCreateVanityRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "POST", "/admin/vanity-roles", dto.CreateVanityRoleRequest{Label: "VIP", Color: "#ff0000"}, authz.PermManageVanityRoles)
}

func TestAdminCreateVanityRole_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)

	// when
	status, _ := h.NewRequest("POST", "/admin/vanity-roles").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminCreateVanityRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	req := dto.CreateVanityRoleRequest{Label: "VIP", Color: "#ff0000", SortOrder: 1}
	expected := &dto.VanityRoleResponse{ID: "r1", Label: "VIP", Color: "#ff0000"}
	ms.EXPECT().CreateVanityRole(mock.Anything, userID, req).Return(expected, nil)

	// when
	status, body := h.NewRequest("POST", "/admin/vanity-roles").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[dto.VanityRoleResponse](t, body)
	assert.Equal(t, "r1", got.ID)
}

func TestAdminCreateVanityRole_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	req := dto.CreateVanityRoleRequest{Label: "VIP", Color: "#ff0000"}
	ms.EXPECT().CreateVanityRole(mock.Anything, userID, req).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("POST", "/admin/vanity-roles").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminUpdateVanityRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "PUT", "/admin/vanity-roles/r1", dto.UpdateVanityRoleRequest{Label: "VIP", Color: "#ff0000"}, authz.PermManageVanityRoles)
}

func TestAdminUpdateVanityRole_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)

	// when
	status, _ := h.NewRequest("PUT", "/admin/vanity-roles/r1").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminUpdateVanityRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	req := dto.UpdateVanityRoleRequest{Label: "VIP", Color: "#ff0000", SortOrder: 2}
	ms.EXPECT().UpdateVanityRole(mock.Anything, userID, "r1", req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/admin/vanity-roles/r1").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminUpdateVanityRole_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"vanity role not found", adminsvc.ErrVanityRoleNotFound, http.StatusNotFound},
		{"system role", adminsvc.ErrSystemRole, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
			req := dto.UpdateVanityRoleRequest{Label: "VIP", Color: "#ff0000"}
			ms.EXPECT().UpdateVanityRole(mock.Anything, userID, "r1", req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/admin/vanity-roles/r1").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminDeleteVanityRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "DELETE", "/admin/vanity-roles/r1", nil, authz.PermManageVanityRoles)
}

func TestAdminDeleteVanityRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	ms.EXPECT().DeleteVanityRole(mock.Anything, userID, "r1").Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/vanity-roles/r1").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminDeleteVanityRole_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"vanity role not found", adminsvc.ErrVanityRoleNotFound, http.StatusNotFound},
		{"system role", adminsvc.ErrSystemRole, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
			ms.EXPECT().DeleteVanityRole(mock.Anything, userID, "r1").Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("DELETE", "/admin/vanity-roles/r1").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminGetVanityRoleUsers_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "GET", "/admin/vanity-roles/r1/users", nil, authz.PermManageVanityRoles)
}

func TestAdminGetVanityRoleUsers_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	expected := &dto.VanityRoleUsersResponse{Total: 2, Limit: 20, Offset: 0}
	ms.EXPECT().GetVanityRoleUsers(mock.Anything, "r1", "", 20, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/admin/vanity-roles/r1/users").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.VanityRoleUsersResponse](t, body)
	assert.Equal(t, 2, got.Total)
}

func TestAdminGetVanityRoleUsers_CustomQuery(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	ms.EXPECT().GetVanityRoleUsers(mock.Anything, "r1", "beato", 5, 10).Return(&dto.VanityRoleUsersResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/admin/vanity-roles/r1/users?search=beato&limit=5&offset=10").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminGetVanityRoleUsers_InternalError(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	ms.EXPECT().GetVanityRoleUsers(mock.Anything, "r1", "", 20, 0).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/admin/vanity-roles/r1/users").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestAdminAssignVanityRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "POST", "/admin/vanity-roles/r1/users", dto.AssignVanityRoleRequest{UserID: uuid.NewString()}, authz.PermManageVanityRoles)
}

func TestAdminAssignVanityRole_BadJSON(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)

	// when
	status, _ := h.NewRequest("POST", "/admin/vanity-roles/r1/users").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminAssignVanityRole_InvalidUserID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)

	// when
	status, body := h.NewRequest("POST", "/admin/vanity-roles/r1/users").
		WithCookie("valid-cookie").
		WithJSONBody(dto.AssignVanityRoleRequest{UserID: "not-a-uuid"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid user id")
}

func TestAdminAssignVanityRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	ms.EXPECT().AssignVanityRole(mock.Anything, userID, "r1", targetID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/vanity-roles/r1/users").
		WithCookie("valid-cookie").
		WithJSONBody(dto.AssignVanityRoleRequest{UserID: targetID.String()}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminAssignVanityRole_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"vanity role not found", adminsvc.ErrVanityRoleNotFound, http.StatusNotFound},
		{"system role", adminsvc.ErrSystemRole, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
			ms.EXPECT().AssignVanityRole(mock.Anything, userID, "r1", targetID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/admin/vanity-roles/r1/users").
				WithCookie("valid-cookie").
				WithJSONBody(dto.AssignVanityRoleRequest{UserID: targetID.String()}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAdminUnassignVanityRole_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, adminFactory, "DELETE", "/admin/vanity-roles/r1/users/"+uuid.NewString(), nil, authz.PermManageVanityRoles)
}

func TestAdminUnassignVanityRole_InvalidUserID(t *testing.T) {
	// given
	h, _, _ := newAdminHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)

	// when
	status, body := h.NewRequest("DELETE", "/admin/vanity-roles/r1/users/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid userId")
}

func TestAdminUnassignVanityRole_OK(t *testing.T) {
	// given
	h, ms, _ := newAdminHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
	ms.EXPECT().UnassignVanityRole(mock.Anything, userID, "r1", targetID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/vanity-roles/r1/users/"+targetID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestAdminUnassignVanityRole_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"vanity role not found", adminsvc.ErrVanityRoleNotFound, http.StatusNotFound},
		{"system role", adminsvc.ErrSystemRole, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms, _ := newAdminHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageVanityRoles, true)
			ms.EXPECT().UnassignVanityRole(mock.Anything, userID, "r1", targetID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("DELETE", "/admin/vanity-roles/r1/users/"+targetID.String()).
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}
