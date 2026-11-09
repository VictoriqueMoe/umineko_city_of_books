package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cspNonce = regexp.MustCompile(`'nonce-([A-Za-z0-9+/=_-]+)'`)
)

func securityHeadersOf(t *testing.T, app *fiber.App) http.Header {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.Header
}

func securityHeadersApp() *fiber.App {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/", func(ctx fiber.Ctx) error {
		return ctx.SendString("ok")
	})

	return app
}

func TestSecurityHeaders_EnforcesTheFullPolicy(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "scripts only come from known places", want: "script-src 'self' blob: https://challenges.cloudflare.com 'nonce-"},
		{name: "everything else defaults to this site", want: "default-src 'self'"},
		{name: "videos linked from any https site still play", want: "media-src 'self' blob: https:"},
		{name: "the site cannot be framed", want: "frame-ancestors 'none'"},
		{name: "plugins stay off", want: "object-src 'none'"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			headers := securityHeadersOf(t, securityHeadersApp())

			// then
			assert.Contains(t, headers.Get(fiber.HeaderContentSecurityPolicy), tc.want)
		})
	}
}

func TestSecurityHeaders_NeverAllowsInlineScriptsWholesale(t *testing.T) {
	// when
	policy := securityHeadersOf(t, securityHeadersApp()).Get(fiber.HeaderContentSecurityPolicy)

	// then
	match := regexp.MustCompile(`script-src[^;]*`).FindString(policy)
	require.NotEmpty(t, match)
	assert.NotContains(t, match, "unsafe-inline")
	assert.NotContains(t, match, "unsafe-eval")
}

func TestSecurityHeaders_UsesAFreshNonceForEveryResponse(t *testing.T) {
	// given
	app := securityHeadersApp()

	// when
	first := cspNonce.FindStringSubmatch(securityHeadersOf(t, app).Get(fiber.HeaderContentSecurityPolicy))
	second := cspNonce.FindStringSubmatch(securityHeadersOf(t, app).Get(fiber.HeaderContentSecurityPolicy))

	// then
	require.Len(t, first, 2)
	require.Len(t, second, 2)
	assert.NotEqual(t, first[1], second[1])
}

func TestSecurityHeaders_NoLongerSendsAReportOnlyPolicy(t *testing.T) {
	// when
	headers := securityHeadersOf(t, securityHeadersApp())

	// then
	assert.Empty(t, headers.Get(fiber.HeaderContentSecurityPolicyReportOnly))
}
