package controllers

import (
	"errors"
	"net/http"
	"testing"

	authsvc "umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	mysterysvc "umineko_city_of_books/internal/mystery"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	usersvc "umineko_city_of_books/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type authDeps struct {
	authSvc    *authsvc.MockService
	userRepo   *repository.MockUserRepository
	mysterySvc *mysterysvc.MockService
	vanityRepo *repository.MockVanityRoleRepository
}

func newAuthHarness(t *testing.T) (*testutil.Harness, authDeps) {
	h := testutil.NewHarness(t)
	deps := authDeps{
		authSvc:    authsvc.NewMockService(t),
		userRepo:   repository.NewMockUserRepository(t),
		mysterySvc: mysterysvc.NewMockService(t),
		vanityRepo: repository.NewMockVanityRoleRepository(t),
	}

	h.SettingsService.EXPECT().Get(mock.Anything, mock.Anything).Return("").Maybe()
	h.SettingsService.EXPECT().GetBool(mock.Anything, mock.Anything).Return(false).Maybe()

	s := &Service{
		AuthService:     deps.authSvc,
		UserRepo:        deps.userRepo,
		MysteryService:  deps.mysterySvc,
		VanityRoleRepo:  deps.vanityRepo,
		SettingsService: h.SettingsService,
		AuthSession:     h.SessionManager,
		AuthzService:    h.AuthzService,
	}
	for _, setup := range s.getAllAuthRoutes() {
		setup(h.App)
	}
	return h, deps
}

func authFactory(t *testing.T) (*testutil.Harness, authDeps) {
	return newAuthHarness(t)
}

func TestRegister_OK(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	req := dto.RegisterRequest{
		LoginRequest: dto.LoginRequest{Username: "beato", Password: "goldenwitch"},
		DisplayName:  "Beatrice",
	}
	userID := uuid.New()
	user := &dto.UserResponse{ID: userID, Username: "beato", DisplayName: "Beatrice"}
	deps.authSvc.EXPECT().Register(mock.Anything, req).Return(user, "session-token", nil)
	deps.userRepo.EXPECT().UpdateIP(mock.Anything, userID, mock.Anything).Return(nil).Maybe()

	// when
	status, body := h.NewRequest("POST", "/auth/register").WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[dto.UserResponse](t, body)
	assert.Equal(t, "beato", got.Username)
}

func TestRegister_BadJSON(t *testing.T) {
	// given
	h, _ := newAuthHarness(t)

	// when
	status, body := h.NewRequest("POST", "/auth/register").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestRegister_MissingCredentials(t *testing.T) {
	cases := []struct {
		name string
		req  dto.RegisterRequest
	}{
		{"empty username", dto.RegisterRequest{LoginRequest: dto.LoginRequest{Username: "", Password: "pw"}}},
		{"empty password", dto.RegisterRequest{LoginRequest: dto.LoginRequest{Username: "beato", Password: ""}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newAuthHarness(t)

			// when
			status, body := h.NewRequest("POST", "/auth/register").WithJSONBody(tc.req).Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "username and password are required")
		})
	}
}

func TestRegister_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"invalid username", authsvc.ErrInvalidUsername, http.StatusBadRequest, "username must be"},
		{"registration disabled", authsvc.ErrRegistrationDisabled, http.StatusForbidden, "registration is currently disabled"},
		{"invite required", authsvc.ErrInviteRequired, http.StatusBadRequest, "invite code is required"},
		{"invalid invite", authsvc.ErrInvalidInvite, http.StatusBadRequest, "invalid or already used invite"},
		{"password too short", authsvc.ErrPasswordTooShort, http.StatusBadRequest, "password must be at least"},
		{"username taken", usersvc.ErrUsernameTaken, http.StatusConflict, "username already taken"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to register"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAuthHarness(t)
			req := dto.RegisterRequest{
				LoginRequest: dto.LoginRequest{Username: "beato", Password: "goldenwitch"},
			}
			deps.authSvc.EXPECT().Register(mock.Anything, req).Return(nil, "", tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/auth/register").WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestLogin_OK(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	req := dto.LoginRequest{Username: "beato", Password: "goldenwitch"}
	userID := uuid.New()
	user := &dto.UserResponse{ID: userID, Username: "beato"}
	deps.authSvc.EXPECT().Login(mock.Anything, req).Return(user, "session-token", nil)
	deps.userRepo.EXPECT().UpdateIP(mock.Anything, userID, mock.Anything).Return(nil).Maybe()

	// when
	status, body := h.NewRequest("POST", "/auth/login").WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.UserResponse](t, body)
	assert.Equal(t, "beato", got.Username)
}

func TestLogin_BadJSON(t *testing.T) {
	// given
	h, _ := newAuthHarness(t)

	// when
	status, body := h.NewRequest("POST", "/auth/login").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestLogin_MissingCredentials(t *testing.T) {
	cases := []struct {
		name string
		req  dto.LoginRequest
	}{
		{"empty username", dto.LoginRequest{Username: "", Password: "pw"}},
		{"empty password", dto.LoginRequest{Username: "beato", Password: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newAuthHarness(t)

			// when
			status, body := h.NewRequest("POST", "/auth/login").WithJSONBody(tc.req).Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "username and password are required")
		})
	}
}

func TestLogin_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"invalid credentials", usersvc.ErrInvalidCredentials, http.StatusUnauthorized, "invalid username or password"},
		{"banned", authsvc.ErrUserBanned, http.StatusForbidden, "your account has been banned"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to login"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAuthHarness(t)
			req := dto.LoginRequest{Username: "beato", Password: "goldenwitch"}
			deps.authSvc.EXPECT().Login(mock.Anything, req).Return(nil, "", tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/auth/login").WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestLogout_OK(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	deps.authSvc.EXPECT().Logout(mock.Anything, "some-cookie").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/auth/logout").WithCookie("some-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestLogout_NoCookie(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	deps.authSvc.EXPECT().Logout(mock.Anything, "").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/auth/logout").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestLogout_ServiceError(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	deps.authSvc.EXPECT().Logout(mock.Anything, "some-cookie").Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/auth/logout").WithCookie("some-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to logout")
}

func TestGetSession_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, authFactory, "GET", "/auth/session", nil)
}

func TestGetSession_OK(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Username: "beato"}, nil)

	// when
	status, body := h.NewRequest("GET", "/auth/session").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, "beato", got["username"])
}

func TestGetSession_ServiceErrors(t *testing.T) {
	cases := []struct {
		name    string
		user    *model.User
		repoErr error
	}{
		{"user not found", nil, nil},
		{"repo error", nil, errors.New("boom")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAuthHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(tc.user, tc.repoErr)

			// when
			status, body := h.NewRequest("GET", "/auth/session").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, http.StatusUnauthorized, status)
			assert.Contains(t, string(body), "not authenticated")
		})
	}
}

func TestSiteInfo_OK(t *testing.T) {
	// given
	h, deps := newAuthHarness(t)
	deps.mysterySvc.EXPECT().GetTopDetectiveIDs(mock.Anything).Return([]string{"det-1"}, nil)
	deps.mysterySvc.EXPECT().GetTopGMIDs(mock.Anything).Return([]string{"gm-1"}, nil)
	deps.vanityRepo.EXPECT().List(mock.Anything).Return([]repository.VanityRoleRow{
		{ID: "role-1", Label: "VIP", Color: "#fff", IsSystem: false, SortOrder: 1},
	}, nil)
	deps.vanityRepo.EXPECT().GetAllAssignments(mock.Anything).Return(map[string][]string{
		"user-1": {"role-1"},
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/site-info").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SiteInfoResponse](t, body)
	assert.Equal(t, []string{"det-1"}, got.TopDetectiveIDs)
	assert.Equal(t, []string{"gm-1"}, got.TopGMIDs)
	assert.Len(t, got.VanityRoles, 1)
	assert.Equal(t, "VIP", got.VanityRoles[0].Label)
	assert.Contains(t, got.VanityRoleAssignments["user-1"], "role-1")
	assert.Contains(t, got.VanityRoleAssignments["det-1"], "system_top_detective")
	assert.Contains(t, got.VanityRoleAssignments["gm-1"], "system_top_gm")
}

func TestSiteInfo_ServiceErrors(t *testing.T) {
	// given - all downstream errors are swallowed; response still 200 with zero values
	h, deps := newAuthHarness(t)
	deps.mysterySvc.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(nil, errors.New("boom"))
	deps.mysterySvc.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, errors.New("boom"))
	deps.vanityRepo.EXPECT().List(mock.Anything).Return(nil, errors.New("boom"))
	deps.vanityRepo.EXPECT().GetAllAssignments(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/site-info").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SiteInfoResponse](t, body)
	assert.Empty(t, got.TopDetectiveIDs)
	assert.Empty(t, got.VanityRoles)
}

func TestGetRules_OK(t *testing.T) {
	// given
	h, _ := newAuthHarness(t)

	// when
	status, body := h.NewRequest("GET", "/rules/theories").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, "theories", got["page"])
}

func TestGetRules_UnknownPage(t *testing.T) {
	// given
	h, _ := newAuthHarness(t)

	// when
	status, body := h.NewRequest("GET", "/rules/nonexistent").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "unknown page")
}
