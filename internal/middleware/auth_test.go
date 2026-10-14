package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"umineko_city_of_books/internal/auth"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testToken = "session-token"
)

func authRequest(t *testing.T, app *fiber.App, method, path, token string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: token})
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(body)
}

func liveState(userID uuid.UUID) model.SessionAuthState {
	return model.SessionAuthState{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}
}

func TestRequireAuth(t *testing.T) {
	userID := uuid.New()
	locked := liveState(userID)
	locked.Locked = true
	unverified := liveState(userID)
	unverified.RequiresEmailVerification = true
	lockedAndUnverified := locked
	lockedAndUnverified.RequiresEmailVerification = true

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		state      model.SessionAuthState
		authErr    error
		wantStatus int
		wantBody   string
	}{
		{name: "no session at all", method: fiber.MethodGet, path: "/api/v1/posts", wantStatus: fiber.StatusUnauthorized, wantBody: `{"error":"authentication required"}`},
		{name: "an invalid session", method: fiber.MethodGet, path: "/api/v1/posts", token: testToken, authErr: auth.ErrInvalidSession, wantStatus: fiber.StatusUnauthorized, wantBody: `{"error":"invalid or expired session"}`},
		{name: "a banned account", method: fiber.MethodGet, path: "/api/v1/posts", token: testToken, authErr: auth.ErrUserBanned, wantStatus: fiber.StatusForbidden, wantBody: `{"error":"your account has been banned"}`},
		{name: "a locked account can still read", method: fiber.MethodGet, path: "/api/v1/posts", token: testToken, state: locked, wantStatus: fiber.StatusOK, wantBody: userID.String()},
		{name: "a locked account cannot write", method: fiber.MethodPost, path: "/api/v1/posts", token: testToken, state: locked, wantStatus: fiber.StatusForbidden, wantBody: `{"error":"your account is locked"}`},
		{name: "a locked account can still mark notifications read", method: fiber.MethodPost, path: "/api/v1/notifications/read", token: testToken, state: locked, wantStatus: fiber.StatusOK, wantBody: userID.String()},
		{name: "a locked account can still send chat messages", method: fiber.MethodPost, path: "/api/v1/chat/rooms/abc/messages", token: testToken, state: locked, wantStatus: fiber.StatusOK, wantBody: userID.String()},
		{name: "an unverified account can still read", method: fiber.MethodGet, path: "/api/v1/posts", token: testToken, state: unverified, wantStatus: fiber.StatusOK, wantBody: userID.String()},
		{name: "an unverified account cannot write", method: fiber.MethodDelete, path: "/api/v1/posts/abc", token: testToken, state: unverified, wantStatus: fiber.StatusForbidden, wantBody: `{"code":"email_unverified","error":"please verify your email address to continue"}`},
		{name: "an unverified account can still verify its email", method: fiber.MethodPost, path: "/api/v1/auth/verify-email", token: testToken, state: unverified, wantStatus: fiber.StatusOK, wantBody: userID.String()},
		{name: "a lock is reported before a missing verification", method: fiber.MethodPost, path: "/api/v1/posts", token: testToken, state: lockedAndUnverified, wantStatus: fiber.StatusForbidden, wantBody: `{"error":"your account is locked"}`},
		{name: "a member in good standing writes", method: fiber.MethodPut, path: "/api/v1/posts/abc", token: testToken, state: liveState(userID), wantStatus: fiber.StatusOK, wantBody: userID.String()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			authenticator := NewMockAuthenticator(t)
			if tc.token != "" {
				authenticator.EXPECT().Authenticate(mock.Anything, tc.token).Return(tc.state, tc.authErr).Once()
			}

			app := fiber.New()
			app.All("/*", RequireAuth(authenticator), func(ctx fiber.Ctx) error {
				userID, _ := ctx.Locals("userID").(uuid.UUID)

				return ctx.SendString(userID.String())
			})

			// when
			status, body := authRequest(t, app, tc.method, tc.path, tc.token)

			// then
			assert.Equal(t, tc.wantStatus, status)
			if tc.wantStatus == fiber.StatusOK {
				assert.Equal(t, tc.wantBody, body)
				return
			}
			assert.JSONEq(t, tc.wantBody, body)
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	userID := uuid.New()
	restricted := liveState(userID)
	restricted.Locked = true
	restricted.RequiresEmailVerification = true

	tests := []struct {
		name       string
		token      string
		state      model.SessionAuthState
		authErr    error
		wantUserID uuid.UUID
	}{
		{name: "a stranger browses anonymously", wantUserID: uuid.Nil},
		{name: "an invalid session browses anonymously", token: testToken, authErr: auth.ErrInvalidSession, wantUserID: uuid.Nil},
		{name: "a banned account browses anonymously", token: testToken, authErr: auth.ErrUserBanned, wantUserID: uuid.Nil},
		{name: "a locked and unverified account still reads as itself", token: testToken, state: restricted, wantUserID: userID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			authenticator := NewMockAuthenticator(t)
			if tc.token != "" {
				authenticator.EXPECT().Authenticate(mock.Anything, tc.token).Return(tc.state, tc.authErr).Once()
			}

			app := fiber.New()
			app.Post("/*", OptionalAuth(authenticator), func(ctx fiber.Ctx) error {
				userID, _ := ctx.Locals("userID").(uuid.UUID)

				return ctx.SendString(userID.String())
			})

			// when
			status, body := authRequest(t, app, fiber.MethodPost, "/api/v1/posts", tc.token)

			// then
			assert.Equal(t, fiber.StatusOK, status)
			assert.Equal(t, tc.wantUserID.String(), body)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name       string
		granted    bool
		wantStatus int
	}{
		{name: "a granted permission passes", granted: true, wantStatus: fiber.StatusOK},
		{name: "a missing permission is refused", granted: false, wantStatus: fiber.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			userID := uuid.New()
			authenticator := NewMockAuthenticator(t)
			authenticator.EXPECT().Authenticate(mock.Anything, testToken).Return(liveState(userID), nil).Once()
			authzSvc := authz.NewMockService(t)
			authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermManageSettings).Return(tc.granted).Once()

			app := fiber.New()
			app.Get("/*", RequirePermission(authenticator, authzSvc, authz.PermManageSettings), func(ctx fiber.Ctx) error {
				return ctx.SendString("ok")
			})

			// when
			status, _ := authRequest(t, app, fiber.MethodGet, "/api/v1/admin/settings", testToken)

			// then
			assert.Equal(t, tc.wantStatus, status)
		})
	}
}

func TestAuthenticate_RunsOncePerRequest(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		path       string
		state      model.SessionAuthState
		authErr    error
		route      func(Authenticator) fiber.Handler
		wantStatus int
		wantBody   string
	}{
		{name: "the private gate and the route share an accepted session", path: "/api/v1/posts", state: liveState(userID), route: RequireAuth, wantStatus: fiber.StatusOK, wantBody: userID.String()},
		{name: "a refused session is not looked up again", path: "/api/v1/auth/session", authErr: auth.ErrInvalidSession, route: OptionalAuth, wantStatus: fiber.StatusOK, wantBody: uuid.Nil.String()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			authenticator := NewMockAuthenticator(t)
			authenticator.EXPECT().Authenticate(mock.Anything, testToken).Return(tc.state, tc.authErr).Once()
			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingPrivateMode).Return(true).Once()

			app := fiber.New()
			app.Use(RequireLogin(settingsSvc, authenticator))
			app.Get(tc.path, tc.route(authenticator), func(ctx fiber.Ctx) error {
				id, _ := ctx.Locals("userID").(uuid.UUID)

				return ctx.SendString(id.String())
			})

			// when
			status, body := authRequest(t, app, fiber.MethodGet, tc.path, testToken)

			// then
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantBody, body)
		})
	}
}
