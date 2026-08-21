package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func privateModeApp(t *testing.T, enabled bool) *fiber.App {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingPrivateMode).Return(enabled).Maybe()

	app := fiber.New()
	app.Use(RequireLogin(settingsSvc, nil))
	app.All("/*", func(ctx fiber.Ctx) error {
		if PrivateGated(ctx) {
			return ctx.SendString("gated")
		}

		return ctx.SendString("handler reached")
	})

	return app
}

func statusOf(t *testing.T, app *fiber.App, path string) (int, string) {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	body := make([]byte, 64)
	n, _ := resp.Body.Read(body)

	return resp.StatusCode, string(body[:n])
}

func TestRequireLogin_DoesNothingWhileDisabled(t *testing.T) {
	// given private mode off, which is the default
	app := privateModeApp(t, false)

	// when
	status, body := statusOf(t, app, "/game-board")

	// then
	assert.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "handler reached", body)
}

func TestRequireLogin_RefusesTheApiToAStranger(t *testing.T) {
	tests := []string{
		"/api/v1/posts",
		"/api/v1/users",
		"/api/v1/search",
		"/api/v1/theories",
		"/api/v1/chat/rooms/public",
		"/api/v1/streams/live",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			// given a site locked to members
			app := privateModeApp(t, true)

			// when
			status, _ := statusOf(t, app, path)

			// then every content read is closed, not just the obvious ones
			assert.Equal(t, fiber.StatusUnauthorized, status)
		})
	}
}

func TestRequireLogin_LeavesSigningInPossible(t *testing.T) {
	tests := []string{
		"/api/v1/site-info",
		"/api/v1/auth/session",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
		"/api/v1/auth/verify-email",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			// given
			app := privateModeApp(t, true)

			// when
			status, _ := statusOf(t, app, path)

			// then a locked out member must still be able to get back in
			assert.Equal(t, fiber.StatusOK, status)
		})
	}
}

func TestRequireLogin_ServesTheShellSoTheLoginPageRenders(t *testing.T) {
	tests := []string{"/", "/login", "/reset-password", "/assets/index-a1b2c3.js", "/favicon.ico"}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			// given
			app := privateModeApp(t, true)

			// when
			status, body := statusOf(t, app, path)

			// then the page loads, but marked so nothing leaks into it
			assert.Equal(t, fiber.StatusOK, status)
			assert.Equal(t, "gated", body)
		})
	}
}

func TestRequireLogin_ClosesTheCrawlerSurface(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "the member roster sitemap", path: "/sitemap-users.xml"},
		{name: "the post sitemap", path: "/sitemap-posts.xml"},
		{name: "the sitemap index", path: "/sitemap.xml"},
		{name: "uploaded media, including dm attachments", path: "/uploads/chat/a_1.webp"},
		{name: "the og image mirror of the same tree", path: "/og-image/posts/a_1.jpg"},
		{name: "stream segments", path: "/hls/stream-abc/live.m3u8"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := privateModeApp(t, true)

			// when
			status, _ := statusOf(t, app, tc.path)

			// then a private site must not still be serving content to strangers
			assert.Equal(t, fiber.StatusUnauthorized, status)
		})
	}
}

func privateModeAppWithSession(t *testing.T, validToken string) *fiber.App {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingPrivateMode).Return(true).Maybe()

	sessionRepo := repository.NewMockSessionRepository(t)
	sessionRepo.EXPECT().GetUserID(mock.Anything, validToken).Return(uuid.New(), time.Now().Add(time.Hour), nil).Maybe()
	sessionRepo.EXPECT().GetUserID(mock.Anything, mock.Anything).Return(uuid.Nil, time.Time{}, errors.New("no session")).Maybe()

	app := fiber.New()
	app.Use(RequireLogin(settingsSvc, session.NewManager(sessionRepo, settingsSvc)))
	app.All("/*", func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	return app
}

func TestRequireLogin_LetsTheNativeAppOpenItsWebsocket(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "a websocket cannot send an Authorization header, so the native app authenticates by query token", path: "/api/v1/ws?token=good", want: fiber.StatusOK},
		{name: "a stranger with no token at all is still refused", path: "/api/v1/ws", want: fiber.StatusUnauthorized},
		{name: "an invalid query token is refused", path: "/api/v1/ws?token=bad", want: fiber.StatusUnauthorized},
		{name: "a query token must not unlock the rest of the api", path: "/api/v1/posts?token=good", want: fiber.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a private site and a session manager that knows one good token
			app := privateModeAppWithSession(t, "good")

			// when
			status, _ := statusOf(t, app, tc.path)

			// then
			assert.Equal(t, tc.want, status)
		})
	}
}

func TestRequireLogin_LeavesStreamingWorking(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "livekit calling back when a stream starts, signed with its own hmac", path: "/api/v1/livekit/webhook"},
		{name: "the obs browser source, which carries its own token and no cookie", path: "/api/v1/overlay"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a private site
			app := privateModeApp(t, true)

			// when
			status, _ := statusOf(t, app, tc.path)

			// then neither has a session to present, and both authenticate themselves
			assert.Equal(t, fiber.StatusOK, status)
		})
	}
}

func TestRequireLogin_NeverBlocksTheHealthcheck(t *testing.T) {
	tests := []string{"/livez", "/health"}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			// given
			app := privateModeApp(t, true)

			// when
			status, _ := statusOf(t, app, path)

			// then a blocked healthcheck would take the container down
			assert.Equal(t, fiber.StatusOK, status)
		})
	}
}

func TestRequireLogin_AnswersTheApiWithJSON(t *testing.T) {
	// given
	app := privateModeApp(t, true)

	// when
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/posts", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	// then an api caller must not be handed html to parse
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}
