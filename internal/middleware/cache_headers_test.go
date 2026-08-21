package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func cacheHeaderApp(t *testing.T, privateMode bool, status int) *fiber.App {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingPrivateMode).Return(privateMode).Maybe()

	app := fiber.New()
	app.Use(CacheHeaders(settingsSvc))
	app.All("/*", func(ctx fiber.Ctx) error {
		return ctx.SendStatus(status)
	})

	return app
}

func cacheControlFor(t *testing.T, app *fiber.App, path string) string {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.Header.Get("Cache-Control")
}

func TestCacheHeaders_NeverLetsAnErrorIntoASharedCache(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
	}{
		{name: "private mode refusing an upload", path: "/uploads/stream-thumbnails/a_1.webp", status: fiber.StatusUnauthorized},
		{name: "dronebl refusing an upload", path: "/uploads/art/a_1.webp", status: fiber.StatusForbidden},
		{name: "a missing upload", path: "/uploads/art/gone.webp", status: fiber.StatusNotFound},
		{name: "an og image refusal", path: "/og-image/posts/a_1.jpg", status: fiber.StatusUnauthorized},
		{name: "an hls segment refusal", path: "/hls/stream-abc/0.ts", status: fiber.StatusUnauthorized},
		{name: "a hashed asset refusal", path: "/assets/index-a1b2c3.js", status: fiber.StatusForbidden},
		{name: "an api refusal", path: "/api/v1/posts", status: fiber.StatusUnauthorized},
		{name: "an origin error", path: "/uploads/art/a_1.webp", status: fiber.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a path whose success response is cacheable
			app := cacheHeaderApp(t, false, tc.status)

			// when the request is refused instead
			got := cacheControlFor(t, app, tc.path)

			// then cloudflare must never store one caller's rejection and serve it to everyone
			assert.NotContains(t, got, "public")
			assert.Contains(t, got, "no-store")
		})
	}
}

func TestCacheHeaders_KeepsGatedMediaOutOfTheSharedCache(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "uploaded media", path: "/uploads/art/a_1.webp"},
		{name: "the og image mirror", path: "/og-image/posts/a_1.jpg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a private site, where these paths answer differently per caller
			app := cacheHeaderApp(t, true, fiber.StatusOK)

			// when a member fetches one successfully
			got := cacheControlFor(t, app, tc.path)

			// then the edge must not keep a copy to hand to logged out strangers
			assert.NotContains(t, got, "public")
			assert.Contains(t, got, "private")
		})
	}
}

func TestCacheHeaders_StillCachesPublicMediaWhileTheSiteIsOpen(t *testing.T) {
	// given private mode off, which is the default
	app := cacheHeaderApp(t, false, fiber.StatusOK)

	// when
	got := cacheControlFor(t, app, "/uploads/art/a_1.webp")

	// then we keep the cloudflare offload, because the response is identical for everyone
	assert.Equal(t, "public, max-age=2592000", got)
}

func TestCacheHeaders_LeavesHashedAssetsImmutable(t *testing.T) {
	// given
	app := cacheHeaderApp(t, true, fiber.StatusOK)

	// when
	got := cacheControlFor(t, app, "/assets/index-a1b2c3.js")

	// then content hashed bundles are the same bytes for everyone, private mode or not
	assert.True(t, strings.Contains(got, "public") && strings.Contains(got, "immutable"), "got %q", got)
}
