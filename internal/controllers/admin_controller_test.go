package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	adminsvc "umineko_city_of_books/internal/admin"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/role"
	usersvc "umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	adminCtlMocks struct {
		admin   *adminsvc.MockService
		user    *usersvc.MockService
		actorID uuid.UUID
	}

	adminCtlExpect func(m adminCtlMocks, err error)

	adminCtlRoute struct {
		name   string
		perm   authz.Permission
		method string
		path   string
		body   any
	}

	adminCtlCase struct {
		name          string
		route         adminCtlRoute
		path          string
		body          any
		malformedBody bool
		expect        adminCtlExpect
		err           error
		wantCode      int
		wantBody      string
		wantJSON      any
	}
)

func adminCtlHarness(t *testing.T) (*testutil.Harness, adminCtlMocks) {
	h := testutil.NewHarness(t)
	m := adminCtlMocks{
		admin:   adminsvc.NewMockService(t),
		user:    usersvc.NewMockService(t),
		actorID: uuid.New(),
	}

	s := &Service{
		AdminService: m.admin,
		UserService:  m.user,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllAdminRoutes() {
		setup(h.App)
	}

	return h, m
}

func adminCtlRunPermissionFailures(t *testing.T, routes ...adminCtlRoute) {
	t.Helper()

	for _, r := range routes {
		t.Run(r.name+" permission failures", func(t *testing.T) {
			testutil.RunPermissionFailureSuite(t, adminCtlHarness, r.method, r.path, r.body, r.perm)
		})
	}
}

func adminCtlRun(t *testing.T, cases []adminCtlCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, m := adminCtlHarness(t)
			h.ExpectValidSession("valid-cookie", m.actorID)
			h.ExpectHasPermission(m.actorID, tc.route.perm, true)
			if tc.expect != nil {
				tc.expect(m, tc.err)
			}

			path := tc.route.path
			if tc.path != "" {
				path = tc.path
			}
			body := tc.route.body
			if tc.body != nil {
				body = tc.body
			}

			// when
			req := h.NewRequest(tc.route.method, path).WithCookie("valid-cookie")
			if tc.malformedBody {
				req = req.WithRawBody("not json", "application/json")
			} else if body != nil {
				req = req.WithJSONBody(body)
			}
			status, respBody := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			if tc.wantBody != "" {
				assert.Contains(t, string(respBody), tc.wantBody)
			}
			if tc.wantJSON != nil {
				want, err := json.Marshal(tc.wantJSON)
				require.NoError(t, err)
				assert.JSONEq(t, string(want), string(respBody))
			}
		})
	}
}

func TestAdminUserLookupRoutes(t *testing.T) {
	targetID := uuid.New()
	user := "/admin/users/" + targetID.String()
	stats := adminCtlRoute{name: "stats", perm: authz.PermViewStats, method: "GET", path: "/admin/stats"}
	listUsers := adminCtlRoute{name: "list users", perm: authz.PermViewUsers, method: "GET", path: "/admin/users"}
	getUser := adminCtlRoute{name: "get user", perm: authz.PermViewUsers, method: "GET", path: user}
	ipMatches := adminCtlRoute{name: "ip matches", perm: authz.PermViewUsers, method: "GET", path: user + "/ip-matches"}
	userAuditLog := adminCtlRoute{name: "user audit log", perm: authz.PermViewAuditLog, method: "GET", path: user + "/audit-log"}

	adminCtlRunPermissionFailures(t, stats, listUsers, getUser, ipMatches, userAuditLog)

	statsRes := &dto.AdminStatsResponse{TotalUsers: 42}
	usersRes := &dto.AdminUserListResponse{Total: 5, Limit: 20, Offset: 0}
	userRes := &dto.AdminUserDetailResponse{AdminUserItem: dto.AdminUserItem{ID: targetID, Username: "beato"}}
	ipRes := &dto.AdminIPMatchesResponse{IP: "10.0.0.1", Users: []dto.AdminUserItem{{ID: uuid.New(), Username: "alt"}}}
	auditRes := &dto.AuditLogListResponse{Entries: []dto.AuditLogEntryResponse{{ID: 1, Action: "ban_user"}}, Total: 1}
	expectStats := func(res *dto.AdminStatsResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().GetStats(mock.Anything).Return(res, err)
		}
	}
	expectListUsers := func(search string, page bounds.Page, res *dto.AdminUserListResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().ListUsers(mock.Anything, search, page).Return(res, err)
		}
	}
	expectGetUser := func(res *dto.AdminUserDetailResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().GetUser(mock.Anything, targetID).Return(res, err)
		}
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "stats", route: stats, expect: expectStats(statsRes), wantCode: http.StatusOK, wantJSON: statsRes},
		{name: "stats internal error", route: stats, expect: expectStats(nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "list users with the default query", route: listUsers, expect: expectListUsers("", bounds.NewPage(20, 0), usersRes), wantCode: http.StatusOK, wantJSON: usersRes},
		{name: "list users with a custom query", route: listUsers, path: "/admin/users?search=beato&limit=50&offset=10", expect: expectListUsers("beato", bounds.NewPage(50, 10), &dto.AdminUserListResponse{}), wantCode: http.StatusOK},
		{name: "list users internal error", route: listUsers, expect: expectListUsers("", bounds.NewPage(20, 0), nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "get user invalid id", route: getUser, path: "/admin/users/not-a-uuid", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "get user", route: getUser, expect: expectGetUser(userRes), wantCode: http.StatusOK, wantJSON: userRes},
		{name: "get user not found", route: getUser, expect: expectGetUser(nil), err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "get user internal error", route: getUser, expect: expectGetUser(nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{
			name:  "ip matches",
			route: ipMatches,
			expect: func(m adminCtlMocks, err error) {
				m.admin.EXPECT().ListAccountsOnIP(mock.Anything, targetID).Return(ipRes, err)
			},
			wantCode: http.StatusOK,
			wantJSON: ipRes,
		},
		{
			name:  "user audit log",
			route: userAuditLog,
			expect: func(m adminCtlMocks, err error) {
				m.admin.EXPECT().GetUserAuditLog(mock.Anything, targetID, bounds.NewPage(20, 0)).Return(auditRes, err)
			},
			wantCode: http.StatusOK,
			wantJSON: auditRes,
		},
	})
}

func TestAdminUserModerationRoutes(t *testing.T) {
	targetID := uuid.New()
	user := "/admin/users/" + targetID.String()
	invalidUser := "/admin/users/not-a-uuid"
	roleBody := dto.SetRoleRequest{Role: "admin"}
	setRole := adminCtlRoute{name: "set role", perm: authz.PermManageRoles, method: "POST", path: user + "/role", body: roleBody}
	removeRole := adminCtlRoute{name: "remove role", perm: authz.PermManageRoles, method: "DELETE", path: user + "/role", body: roleBody}
	ban := adminCtlRoute{name: "ban", perm: authz.PermBanUser, method: "POST", path: user + "/ban", body: dto.BanUserRequest{Reason: "spam"}}
	unban := adminCtlRoute{name: "unban", perm: authz.PermBanUser, method: "POST", path: user + "/unban"}
	forceLogout := adminCtlRoute{name: "force logout", perm: authz.PermBanUser, method: "POST", path: user + "/force-logout"}
	deleteUser := adminCtlRoute{name: "delete user", perm: authz.PermDeleteAnyUser, method: "DELETE", path: user}
	resetPassword := adminCtlRoute{name: "reset password", perm: authz.PermResetPassword, method: "POST", path: user + "/reset-password"}

	adminCtlRunPermissionFailures(t, setRole, removeRole, ban, unban, forceLogout, deleteUser, resetPassword)

	expectSetRole := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().SetUserRole(mock.Anything, m.actorID, targetID, role.Role("admin")).Return(err)
	}
	expectRemoveRole := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().RemoveUserRole(mock.Anything, m.actorID, targetID, role.Role("admin")).Return(err)
	}
	expectBan := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().BanUser(mock.Anything, m.actorID, targetID, "spam").Return(err)
	}
	expectUnban := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().UnbanUser(mock.Anything, m.actorID, targetID).Return(err)
	}
	expectForceLogout := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().ForceLogout(mock.Anything, m.actorID, targetID).Return(err)
	}
	expectDeleteUser := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().DeleteUser(mock.Anything, m.actorID, targetID).Return(err)
	}
	expectResetPassword := func(password string) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().ResetUserPassword(mock.Anything, m.actorID, targetID).Return(password, err)
		}
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "set role invalid id", route: setRole, path: invalidUser + "/role", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "set role malformed body", route: setRole, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "set role", route: setRole, expect: expectSetRole, wantCode: http.StatusOK},
		{name: "set role user not found", route: setRole, expect: expectSetRole, err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "set role protected user", route: setRole, expect: expectSetRole, err: adminsvc.ErrProtectedUser, wantCode: http.StatusForbidden},
		{name: "set role internal error", route: setRole, expect: expectSetRole, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "remove role invalid id", route: removeRole, path: invalidUser + "/role", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "remove role malformed body", route: removeRole, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "remove role", route: removeRole, expect: expectRemoveRole, wantCode: http.StatusOK},
		{name: "remove role user not found", route: removeRole, expect: expectRemoveRole, err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "remove role protected user", route: removeRole, expect: expectRemoveRole, err: adminsvc.ErrProtectedUser, wantCode: http.StatusForbidden},
		{name: "remove role internal error", route: removeRole, expect: expectRemoveRole, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "ban invalid id", route: ban, path: invalidUser + "/ban", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "ban malformed body", route: ban, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "ban", route: ban, expect: expectBan, wantCode: http.StatusOK},
		{name: "ban user not found", route: ban, expect: expectBan, err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "ban protected user", route: ban, expect: expectBan, err: adminsvc.ErrProtectedUser, wantCode: http.StatusForbidden},
		{name: "ban internal error", route: ban, expect: expectBan, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "unban invalid id", route: unban, path: invalidUser + "/unban", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "unban", route: unban, expect: expectUnban, wantCode: http.StatusOK},
		{name: "unban user not found", route: unban, expect: expectUnban, err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "unban internal error", route: unban, expect: expectUnban, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "force logout", route: forceLogout, expect: expectForceLogout, wantCode: http.StatusOK},
		{name: "delete user invalid id", route: deleteUser, path: invalidUser, wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "delete user", route: deleteUser, expect: expectDeleteUser, wantCode: http.StatusOK},
		{name: "delete user not found", route: deleteUser, expect: expectDeleteUser, err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "delete user protected user", route: deleteUser, expect: expectDeleteUser, err: adminsvc.ErrProtectedUser, wantCode: http.StatusForbidden},
		{name: "delete user internal error", route: deleteUser, expect: expectDeleteUser, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "reset password invalid id", route: resetPassword, path: invalidUser + "/reset-password", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "reset password", route: resetPassword, expect: expectResetPassword("hunter2hunter2hu"), wantCode: http.StatusOK, wantJSON: dto.AdminResetPasswordResponse{Password: "hunter2hunter2hu"}},
		{name: "reset password user not found", route: resetPassword, expect: expectResetPassword(""), err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "reset password protected user", route: resetPassword, expect: expectResetPassword(""), err: adminsvc.ErrProtectedUser, wantCode: http.StatusForbidden},
		{name: "reset password internal error", route: resetPassword, expect: expectResetPassword(""), err: assert.AnError, wantCode: http.StatusInternalServerError},
	})
}

func TestAdminUserAccountRoutes(t *testing.T) {
	targetID := uuid.New()
	user := "/admin/users/" + targetID.String()
	setEmail := adminCtlRoute{name: "set email", perm: authz.PermManageUserEmail, method: "PUT", path: user + "/email", body: dto.AdminSetEmailRequest{Email: "new@example.com"}}
	verifyEmail := adminCtlRoute{name: "verify email", perm: authz.PermSetEmailVerified, method: "POST", path: user + "/verify-email"}
	unverifyEmail := adminCtlRoute{name: "unverify email", perm: authz.PermSetEmailVerified, method: "POST", path: user + "/unverify-email"}
	displayName := adminCtlRoute{name: "display name", perm: authz.PermManageUserAccount, method: "PUT", path: user + "/display-name", body: dto.AdminSetDisplayNameRequest{DisplayName: "Beato"}}
	displayNameLock := adminCtlRoute{name: "display name lock", perm: authz.PermManageUserAccount, method: "PUT", path: user + "/display-name-lock", body: dto.AdminSetDisplayNameLockRequest{Locked: true}}

	adminCtlRunPermissionFailures(t, setEmail, verifyEmail, unverifyEmail, displayName, displayNameLock)

	expectSetEmail := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().SetUserEmail(mock.Anything, m.actorID, targetID, "new@example.com").Return(err)
	}
	expectVerify := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().VerifyUserEmail(mock.Anything, m.actorID, targetID).Return(err)
	}
	expectUnverify := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().UnverifyUserEmail(mock.Anything, m.actorID, targetID).Return(err)
	}
	expectDisplayName := func(name string) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().SetUserDisplayName(mock.Anything, m.actorID, targetID, name).Return(err)
		}
	}
	expectLock := func(locked bool) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().SetDisplayNameLocked(mock.Anything, m.actorID, targetID, locked).Return(err)
		}
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "set email", route: setEmail, expect: expectSetEmail, wantCode: http.StatusOK},
		{name: "set email invalid email", route: setEmail, expect: expectSetEmail, err: auth.ErrInvalidEmail, wantCode: http.StatusBadRequest},
		{name: "set email taken", route: setEmail, expect: expectSetEmail, err: auth.ErrEmailTaken, wantCode: http.StatusBadRequest},
		{name: "set email protected user", route: setEmail, expect: expectSetEmail, err: adminsvc.ErrProtectedUser, wantCode: http.StatusForbidden},
		{name: "set email user not found", route: setEmail, expect: expectSetEmail, err: adminsvc.ErrUserNotFound, wantCode: http.StatusNotFound},
		{name: "set email internal error", route: setEmail, expect: expectSetEmail, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "verify email", route: verifyEmail, expect: expectVerify, wantCode: http.StatusOK},
		{name: "verify email already verified", route: verifyEmail, expect: expectVerify, err: auth.ErrEmailAlreadyVerified, wantCode: http.StatusBadRequest},
		{name: "unverify email", route: unverifyEmail, expect: expectUnverify, wantCode: http.StatusOK},
		{name: "unverify email not verified", route: unverifyEmail, expect: expectUnverify, err: auth.ErrEmailNotVerified, wantCode: http.StatusBadRequest},
		{name: "display name", route: displayName, expect: expectDisplayName("Beato"), wantCode: http.StatusOK},
		{name: "display name blank rejected", route: displayName, body: dto.AdminSetDisplayNameRequest{DisplayName: "  "}, expect: expectDisplayName("  "), err: adminsvc.ErrEmptyDisplayName, wantCode: http.StatusBadRequest},
		{name: "display name lock", route: displayNameLock, expect: expectLock(true), wantCode: http.StatusOK},
		{name: "display name unlock", route: displayNameLock, body: dto.AdminSetDisplayNameLockRequest{Locked: false}, expect: expectLock(false), wantCode: http.StatusOK},
	})
}

func TestAdminSettingsRoutes(t *testing.T) {
	getSettings := adminCtlRoute{name: "get settings", perm: authz.PermManageSettings, method: "GET", path: "/admin/settings"}
	updateSettings := adminCtlRoute{name: "update settings", perm: authz.PermManageSettings, method: "PUT", path: "/admin/settings", body: dto.UpdateSettingsRequest{Settings: map[string]string{}}}
	testEmail := adminCtlRoute{name: "send test email", perm: authz.PermManageSettings, method: "POST", path: "/admin/settings/test-email"}
	auditLog := adminCtlRoute{name: "audit log", perm: authz.PermViewAuditLog, method: "GET", path: "/admin/audit-log"}

	adminCtlRunPermissionFailures(t, getSettings, updateSettings, testEmail, auditLog)

	settingsRes := &dto.SettingsResponse{Settings: map[string]string{"site_name": "umineko"}}
	auditRes := &dto.AuditLogListResponse{Total: 3, Limit: 50, Offset: 0}
	siteName := map[string]string{"site_name": "umineko"}
	other := map[string]string{"x": "y"}
	expectGetSettings := func(res *dto.SettingsResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().GetSettings(mock.Anything).Return(res, err)
		}
	}
	expectUpdateSettings := func(settings map[string]string) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().UpdateSettings(mock.Anything, m.actorID, settings).Return(err)
		}
	}
	expectTestEmail := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().SendTestEmail(mock.Anything, m.actorID).Return(err)
	}
	expectAuditLog := func(action audit.Action, page bounds.Page, res *dto.AuditLogListResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().GetAuditLog(mock.Anything, action, page).Return(res, err)
		}
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "get settings", route: getSettings, expect: expectGetSettings(settingsRes), wantCode: http.StatusOK, wantJSON: settingsRes},
		{name: "get settings internal error", route: getSettings, expect: expectGetSettings(nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "update settings malformed body", route: updateSettings, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "update settings", route: updateSettings, body: dto.UpdateSettingsRequest{Settings: siteName}, expect: expectUpdateSettings(siteName), wantCode: http.StatusOK},
		{name: "update settings internal error", route: updateSettings, body: dto.UpdateSettingsRequest{Settings: other}, expect: expectUpdateSettings(other), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "send test email", route: testEmail, expect: expectTestEmail, wantCode: http.StatusOK},
		{name: "send test email without an address", route: testEmail, expect: expectTestEmail, err: adminsvc.ErrNoEmailAddress, wantCode: http.StatusBadRequest, wantBody: "your account has no email address set"},
		{name: "send test email internal error", route: testEmail, expect: expectTestEmail, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "audit log with the default query", route: auditLog, expect: expectAuditLog("", bounds.NewPage(50, 0), auditRes), wantCode: http.StatusOK, wantJSON: auditRes},
		{name: "audit log with a custom query", route: auditLog, path: "/admin/audit-log?action=ban_user&limit=10&offset=20", expect: expectAuditLog(audit.ActionBanUser, bounds.NewPage(10, 20), &dto.AuditLogListResponse{}), wantCode: http.StatusOK},
		{name: "audit log internal error", route: auditLog, expect: expectAuditLog("", bounds.NewPage(50, 0), nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
	})
}

func TestAdminInviteRoutes(t *testing.T) {
	create := adminCtlRoute{name: "create invite", perm: authz.PermManageRoles, method: "POST", path: "/admin/invites"}
	list := adminCtlRoute{name: "list invites", perm: authz.PermManageRoles, method: "GET", path: "/admin/invites"}
	remove := adminCtlRoute{name: "delete invite", perm: authz.PermManageRoles, method: "DELETE", path: "/admin/invites/abc123"}

	adminCtlRunPermissionFailures(t, create, list, remove)

	inviteRes := &dto.InviteResponse{Code: "abcd1234", CreatedBy: uuid.New(), CreatedAt: "just now"}
	listRes := &dto.InviteListResponse{Total: 7, Limit: 50, Offset: 0}
	expectCreate := func(res *dto.InviteResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().CreateInvite(mock.Anything, m.actorID).Return(res, err)
		}
	}
	expectList := func(page bounds.Page, res *dto.InviteListResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().ListInvites(mock.Anything, page).Return(res, err)
		}
	}
	expectDelete := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().DeleteInvite(mock.Anything, m.actorID, "abc123").Return(err)
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "create invite", route: create, expect: expectCreate(inviteRes), wantCode: http.StatusCreated, wantJSON: inviteRes},
		{name: "create invite internal error does not leak the cause", route: create, expect: expectCreate(nil), err: assert.AnError, wantCode: http.StatusInternalServerError, wantJSON: map[string]string{"error": "admin action failed"}},
		{name: "list invites with the default query", route: list, expect: expectList(bounds.NewPage(50, 0), listRes), wantCode: http.StatusOK, wantJSON: listRes},
		{name: "list invites with a custom query", route: list, path: "/admin/invites?limit=5&offset=15", expect: expectList(bounds.NewPage(5, 15), &dto.InviteListResponse{}), wantCode: http.StatusOK},
		{name: "list invites internal error", route: list, expect: expectList(bounds.NewPage(50, 0), nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "delete invite", route: remove, expect: expectDelete, wantCode: http.StatusOK},
		{name: "delete invite internal error", route: remove, expect: expectDelete, err: assert.AnError, wantCode: http.StatusInternalServerError},
	})
}

func TestAdminScoreRoutes(t *testing.T) {
	targetID := uuid.New()
	user := "/admin/users/" + targetID.String()
	mystery := adminCtlRoute{name: "mystery score", perm: authz.PermEditMysteryScore, method: "PUT", path: user + "/mystery-score", body: map[string]int{"desired_score": 100}}
	gm := adminCtlRoute{name: "gm score", perm: authz.PermEditMysteryScore, method: "PUT", path: user + "/gm-score", body: map[string]int{"desired_score": 50}}

	adminCtlRunPermissionFailures(t, mystery, gm)

	expectMystery := func(raw, adjustment int) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.user.EXPECT().GetDetectiveRawScore(mock.Anything, targetID).Return(raw, nil)
			m.user.EXPECT().UpdateMysteryScoreAdjustment(mock.Anything, m.actorID, targetID, adjustment).Return(err)
		}
	}
	expectGM := func(raw, adjustment int) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.user.EXPECT().GetGMRawScore(mock.Anything, targetID).Return(raw, nil)
			m.user.EXPECT().UpdateGMScoreAdjustment(mock.Anything, m.actorID, targetID, adjustment).Return(err)
		}
	}
	expectMysteryLookupFails := func(m adminCtlMocks, err error) {
		m.user.EXPECT().GetDetectiveRawScore(mock.Anything, targetID).Return(0, err)
	}
	expectGMLookupFails := func(m adminCtlMocks, err error) {
		m.user.EXPECT().GetGMRawScore(mock.Anything, targetID).Return(0, err)
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "mystery score invalid id", route: mystery, path: "/admin/users/not-a-uuid/mystery-score", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "mystery score malformed body", route: mystery, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request"},
		{name: "mystery score", route: mystery, expect: expectMystery(30, 70), wantCode: http.StatusNoContent},
		{name: "mystery score update fails", route: mystery, expect: expectMystery(0, 100), err: assert.AnError, wantCode: http.StatusInternalServerError, wantBody: "failed to update"},
		{name: "mystery score raw lookup fails without writing an adjustment", route: mystery, expect: expectMysteryLookupFails, err: assert.AnError, wantCode: http.StatusInternalServerError, wantBody: "failed to read the current score"},
		{name: "gm score invalid id", route: gm, path: "/admin/users/not-a-uuid/gm-score", wantCode: http.StatusBadRequest, wantBody: "invalid id"},
		{name: "gm score malformed body", route: gm, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request"},
		{name: "gm score", route: gm, expect: expectGM(10, 40), wantCode: http.StatusNoContent},
		{name: "gm score update fails", route: gm, expect: expectGM(0, 50), err: assert.AnError, wantCode: http.StatusInternalServerError, wantBody: "failed to update"},
		{name: "gm score raw lookup fails without writing an adjustment", route: gm, expect: expectGMLookupFails, err: assert.AnError, wantCode: http.StatusInternalServerError, wantBody: "failed to read the current score"},
	})
}

func TestAdminVanityRoleRoutes(t *testing.T) {
	targetID := uuid.New()
	createReq := dto.CreateVanityRoleRequest{Label: "VIP", Color: "#ff0000"}
	updateReq := dto.UpdateVanityRoleRequest{Label: "VIP", Color: "#ff0000"}
	list := adminCtlRoute{name: "list vanity roles", perm: authz.PermManageVanityRoles, method: "GET", path: "/admin/vanity-roles"}
	create := adminCtlRoute{name: "create vanity role", perm: authz.PermManageVanityRoles, method: "POST", path: "/admin/vanity-roles", body: createReq}
	update := adminCtlRoute{name: "update vanity role", perm: authz.PermManageVanityRoles, method: "PUT", path: "/admin/vanity-roles/r1", body: updateReq}
	remove := adminCtlRoute{name: "delete vanity role", perm: authz.PermManageVanityRoles, method: "DELETE", path: "/admin/vanity-roles/r1"}
	users := adminCtlRoute{name: "vanity role users", perm: authz.PermManageVanityRoles, method: "GET", path: "/admin/vanity-roles/r1/users"}
	assign := adminCtlRoute{name: "assign vanity role", perm: authz.PermManageVanityRoles, method: "POST", path: "/admin/vanity-roles/r1/users", body: dto.AssignVanityRoleRequest{UserID: targetID.String()}}
	unassign := adminCtlRoute{name: "unassign vanity role", perm: authz.PermManageVanityRoles, method: "DELETE", path: "/admin/vanity-roles/r1/users/" + targetID.String()}

	adminCtlRunPermissionFailures(t, list, create, update, remove, users, assign, unassign)

	listRes := []dto.VanityRoleResponse{{ID: "r1", Label: "VIP", Color: "#ff0000"}}
	createdRes := &dto.VanityRoleResponse{ID: "r1", Label: "VIP", Color: "#ff0000"}
	usersRes := &dto.VanityRoleUsersResponse{Total: 2, Limit: 20, Offset: 0}
	sortedCreateReq := dto.CreateVanityRoleRequest{Label: "VIP", Color: "#ff0000", SortOrder: 1}
	sortedUpdateReq := dto.UpdateVanityRoleRequest{Label: "VIP", Color: "#ff0000", SortOrder: 2}
	expectList := func(res []dto.VanityRoleResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().ListVanityRoles(mock.Anything).Return(res, err)
		}
	}
	expectCreate := func(req dto.CreateVanityRoleRequest, res *dto.VanityRoleResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().CreateVanityRole(mock.Anything, m.actorID, req).Return(res, err)
		}
	}
	expectUpdate := func(req dto.UpdateVanityRoleRequest) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().UpdateVanityRole(mock.Anything, m.actorID, "r1", req).Return(err)
		}
	}
	expectDelete := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().DeleteVanityRole(mock.Anything, m.actorID, "r1").Return(err)
	}
	expectUsers := func(search string, page bounds.Page, res *dto.VanityRoleUsersResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().GetVanityRoleUsers(mock.Anything, "r1", search, page).Return(res, err)
		}
	}
	expectAssign := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().AssignVanityRole(mock.Anything, m.actorID, "r1", targetID).Return(err)
	}
	expectUnassign := func(m adminCtlMocks, err error) {
		m.admin.EXPECT().UnassignVanityRole(mock.Anything, m.actorID, "r1", targetID).Return(err)
	}

	adminCtlRun(t, []adminCtlCase{
		{name: "list vanity roles", route: list, expect: expectList(listRes), wantCode: http.StatusOK, wantJSON: listRes},
		{name: "list vanity roles internal error", route: list, expect: expectList(nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "create vanity role malformed body", route: create, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "create vanity role", route: create, body: sortedCreateReq, expect: expectCreate(sortedCreateReq, createdRes), wantCode: http.StatusCreated, wantJSON: createdRes},
		{name: "create vanity role internal error", route: create, expect: expectCreate(createReq, nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "update vanity role malformed body", route: update, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "update vanity role", route: update, body: sortedUpdateReq, expect: expectUpdate(sortedUpdateReq), wantCode: http.StatusOK},
		{name: "update vanity role not found", route: update, expect: expectUpdate(updateReq), err: adminsvc.ErrVanityRoleNotFound, wantCode: http.StatusNotFound},
		{name: "update vanity role system role", route: update, expect: expectUpdate(updateReq), err: adminsvc.ErrSystemRole, wantCode: http.StatusForbidden},
		{name: "update vanity role internal error", route: update, expect: expectUpdate(updateReq), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "delete vanity role", route: remove, expect: expectDelete, wantCode: http.StatusOK},
		{name: "delete vanity role not found", route: remove, expect: expectDelete, err: adminsvc.ErrVanityRoleNotFound, wantCode: http.StatusNotFound},
		{name: "delete vanity role system role", route: remove, expect: expectDelete, err: adminsvc.ErrSystemRole, wantCode: http.StatusForbidden},
		{name: "delete vanity role internal error", route: remove, expect: expectDelete, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "vanity role users with the default query", route: users, expect: expectUsers("", bounds.NewPage(20, 0), usersRes), wantCode: http.StatusOK, wantJSON: usersRes},
		{name: "vanity role users with a custom query", route: users, path: "/admin/vanity-roles/r1/users?search=beato&limit=5&offset=10", expect: expectUsers("beato", bounds.NewPage(5, 10), &dto.VanityRoleUsersResponse{}), wantCode: http.StatusOK},
		{name: "vanity role users internal error", route: users, expect: expectUsers("", bounds.NewPage(20, 0), nil), err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "assign vanity role malformed body", route: assign, malformedBody: true, wantCode: http.StatusBadRequest, wantBody: "invalid request body"},
		{name: "assign vanity role invalid user id", route: assign, body: dto.AssignVanityRoleRequest{UserID: "not-a-uuid"}, wantCode: http.StatusBadRequest, wantBody: "invalid user id"},
		{name: "assign vanity role", route: assign, expect: expectAssign, wantCode: http.StatusOK},
		{name: "assign vanity role not found", route: assign, expect: expectAssign, err: adminsvc.ErrVanityRoleNotFound, wantCode: http.StatusNotFound},
		{name: "assign vanity role system role", route: assign, expect: expectAssign, err: adminsvc.ErrSystemRole, wantCode: http.StatusForbidden},
		{name: "assign vanity role internal error", route: assign, expect: expectAssign, err: assert.AnError, wantCode: http.StatusInternalServerError},
		{name: "unassign vanity role invalid user id", route: unassign, path: "/admin/vanity-roles/r1/users/not-a-uuid", wantCode: http.StatusBadRequest, wantBody: "invalid userId"},
		{name: "unassign vanity role", route: unassign, expect: expectUnassign, wantCode: http.StatusOK},
		{name: "unassign vanity role not found", route: unassign, expect: expectUnassign, err: adminsvc.ErrVanityRoleNotFound, wantCode: http.StatusNotFound},
		{name: "unassign vanity role system role", route: unassign, expect: expectUnassign, err: adminsvc.ErrSystemRole, wantCode: http.StatusForbidden},
		{name: "unassign vanity role internal error", route: unassign, expect: expectUnassign, err: assert.AnError, wantCode: http.StatusInternalServerError},
	})
}

func TestAdminBannedGifRoutes(t *testing.T) {
	list := adminCtlRoute{name: "list banned gifs", perm: authz.PermManageSettings, method: "GET", path: "/admin/banned-gifs"}
	add := adminCtlRoute{name: "add banned gif", perm: authz.PermManageSettings, method: "POST", path: "/admin/banned-gifs", body: dto.AddBannedGiphyRequest{Input: "abc"}}
	remove := adminCtlRoute{name: "remove banned gif", perm: authz.PermManageSettings, method: "DELETE", path: "/admin/banned-gifs/gif/abc123"}

	adminCtlRunPermissionFailures(t, list, add, remove)

	entry := dto.BannedGiphyEntry{Kind: "gif", Value: "abc123"}
	urlReq := dto.AddBannedGiphyRequest{Input: "https://giphy.com/gifs/cat-abc123", Reason: "spam"}
	garbageReq := dto.AddBannedGiphyRequest{Input: "!!!"}
	mismatchReq := dto.AddBannedGiphyRequest{Input: "https://giphy.com/channel/foo", Kind: "gif"}
	expectAdd := func(req dto.AddBannedGiphyRequest, res *dto.AddBannedGiphyResponse) adminCtlExpect {
		return func(m adminCtlMocks, err error) {
			m.admin.EXPECT().AddBannedGif(mock.Anything, m.actorID, req).Return(res, err)
		}
	}

	adminCtlRun(t, []adminCtlCase{
		{
			name:  "list banned gifs",
			route: list,
			expect: func(m adminCtlMocks, err error) {
				m.admin.EXPECT().ListBannedGifs(mock.Anything).Return(&dto.BannedGiphyListResponse{Entries: []dto.BannedGiphyEntry{entry}}, err)
			},
			wantCode: http.StatusOK,
			wantBody: `"value":"abc123"`,
		},
		{name: "add banned gif", route: add, body: urlReq, expect: expectAdd(urlReq, &dto.AddBannedGiphyResponse{Entry: entry}), wantCode: http.StatusCreated, wantBody: `"value":"abc123"`},
		{name: "add banned gif unrecognised input", route: add, body: garbageReq, expect: expectAdd(garbageReq, nil), err: adminsvc.ErrBannedGiphyInvalidInput, wantCode: http.StatusBadRequest, wantBody: "could not recognise"},
		{name: "add banned gif kind mismatch", route: add, body: mismatchReq, expect: expectAdd(mismatchReq, nil), err: adminsvc.ErrBannedGiphyKindMismatch, wantCode: http.StatusBadRequest, wantBody: "does not match"},
		{
			name:  "remove banned gif",
			route: remove,
			expect: func(m adminCtlMocks, err error) {
				m.admin.EXPECT().RemoveBannedGif(mock.Anything, m.actorID, "gif", "abc123").Return(err)
			},
			wantCode: http.StatusOK,
		},
	})
}
