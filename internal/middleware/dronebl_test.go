package middleware

import (
	"net"
	"net/http/httptest"
	"net/netip"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dronebl"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type droneblCase struct {
	enabled   bool
	listed    bool
	allowlist string
	ignored   string
}

func droneblApp(t *testing.T, tc droneblCase) *fiber.App {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingDroneBLEnabled).Return(tc.enabled).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingDroneBLAllowlist).Return(tc.allowlist).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingDroneBLIgnoredClasses).Return(tc.ignored).Maybe()

	resolver := dronebl.NewMockResolver(t)
	if tc.listed {
		resolver.EXPECT().
			LookupNetIP(mock.Anything, "ip4", mock.Anything).
			Return([]netip.Addr{netip.AddrFrom4([4]byte{127, 0, 0, 13})}, nil).
			Maybe()
	} else {
		resolver.EXPECT().
			LookupNetIP(mock.Anything, "ip4", mock.Anything).
			Return(nil, &net.DNSError{Err: "no such host", IsNotFound: true}).
			Maybe()
	}

	checker := dronebl.New(settingsSvc, cache.NewManager(engines.NewInMemory(0)), resolver, nil)

	app := fiber.New()
	app.Use(func(ctx fiber.Ctx) error {
		ctx.Locals("client_ip", "190.114.41.165")

		return ctx.Next()
	})
	app.Use(RequireCleanIP(checker, nil))
	app.All("/*", func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	return app
}

func getAs(t *testing.T, app *fiber.App, path, accept string) (int, string) {
	t.Helper()

	req := httptest.NewRequest("GET", path, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.StatusCode, resp.Header.Get("Content-Type")
}

func TestRequireCleanIP_BlocksAListedAddress(t *testing.T) {
	// given a listed visitor with no session
	app := droneblApp(t, droneblCase{enabled: true, listed: true})

	// when they ask for a page
	status, contentType := getAs(t, app, "/game-board", "text/html")

	// then they get the blocked page, not the site
	assert.Equal(t, fiber.StatusForbidden, status)
	assert.Contains(t, contentType, "text/html")
}

func TestRequireCleanIP_AnsweresAPICallsWithJSON(t *testing.T) {
	// given the same visitor, but the request is from the app rather than a navigation
	app := droneblApp(t, droneblCase{enabled: true, listed: true})

	// when
	status, contentType := getAs(t, app, "/api/v1/posts", "text/html")

	// then an API caller must never be handed an HTML page to parse
	assert.Equal(t, fiber.StatusForbidden, status)
	assert.Contains(t, contentType, "application/json")
}

func TestRequireCleanIP_LetsACleanAddressThrough(t *testing.T) {
	// given
	app := droneblApp(t, droneblCase{enabled: true, listed: false})

	// when
	status, _ := getAs(t, app, "/game-board", "text/html")

	// then
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireCleanIP_DoesNothingWhileDisabled(t *testing.T) {
	// given the setting is off, which is the default
	app := droneblApp(t, droneblCase{enabled: false, listed: true})

	// when
	status, _ := getAs(t, app, "/game-board", "text/html")

	// then the blocklist is never consulted
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireCleanIP_HonoursTheAllowlist(t *testing.T) {
	// given an operator who rescued their own range
	app := droneblApp(t, droneblCase{enabled: true, listed: true, allowlist: "190.114.41.0/24"})

	// when
	status, _ := getAs(t, app, "/game-board", "text/html")

	// then
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireCleanIP_HonoursIgnoredClasses(t *testing.T) {
	// given class 13, the noisy one that catches recycled residential addresses
	app := droneblApp(t, droneblCase{enabled: true, listed: true, ignored: "13"})

	// when
	status, _ := getAs(t, app, "/game-board", "text/html")

	// then
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireCleanIP_LeavesAnEscapeRouteToSignIn(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "the login page itself", path: "/login"},
		{name: "the login endpoint", path: "/api/v1/auth/login"},
		{name: "the session endpoint", path: "/api/v1/auth/session"},
		{name: "site info, which the shell needs to boot", path: "/api/v1/site-info"},
		{name: "a javascript bundle", path: "/assets/index-a1b2c3.js"},
		{name: "a stylesheet", path: "/assets/index-a1b2c3.css"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a listed member whose session has expired
			app := droneblApp(t, droneblCase{enabled: true, listed: true})

			// when
			status, _ := getAs(t, app, tc.path, "text/html")

			// then they can still reach the login page and sign back in
			assert.Equal(t, fiber.StatusOK, status)
		})
	}
}

func TestRequireCleanIP_StillBlocksTheThingsWorthBlocking(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "registration, which is how a raid starts", path: "/api/v1/auth/register"},
		{name: "uploaded media, which is the bandwidth", path: "/uploads/posts/gore.webp"},
		{name: "an ordinary api call", path: "/api/v1/chat/rooms"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := droneblApp(t, droneblCase{enabled: true, listed: true})

			// when
			status, _ := getAs(t, app, tc.path, "text/html")

			// then
			assert.Equal(t, fiber.StatusForbidden, status)
		})
	}
}

func TestRequireCleanIP_NeverBlocksTheHealthcheck(t *testing.T) {
	tests := []string{"/livez", "/health"}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			// given a listed address, which the compose healthcheck could well be
			app := droneblApp(t, droneblCase{enabled: true, listed: true})

			// when
			status, _ := getAs(t, app, path, "")

			// then a blocked healthcheck would take the container down
			assert.Equal(t, fiber.StatusOK, status)
		})
	}
}

func TestRequireCleanIP_AllowsEverythingWhenUnwired(t *testing.T) {
	// given a misconfiguration, where refusing every request would take the site down
	app := fiber.New()
	app.Use(RequireCleanIP(nil, nil))
	app.Get("/*", func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	// when
	status, _ := getAs(t, app, "/game-board", "text/html")

	// then
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireCleanIP_SkipsWhenTheAddressIsUnknown(t *testing.T) {
	// given a request that never got a client_ip local
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetBool(mock.Anything, config.SettingDroneBLEnabled).Return(true).Maybe()
	checker := dronebl.New(settingsSvc, cache.NewManager(engines.NewInMemory(0)), dronebl.NewMockResolver(t), nil)

	app := fiber.New()
	app.Use(RequireCleanIP(checker, nil))
	app.Get("/*", func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	// when
	status, _ := getAs(t, app, "/game-board", "text/html")

	// then no resolver expectation was registered, so a lookup would fail the test
	assert.Equal(t, fiber.StatusOK, status)
}
