package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testSiteOrigin = "https://whentheycry.example"
)

func TestRejectForeignOrigins(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		origin  string
		baseURL string
		want    int
	}{
		{name: "a write from the site itself passes", method: http.MethodPost, origin: testSiteOrigin, baseURL: testSiteOrigin, want: http.StatusOK},
		{name: "a trailing slash on the base url changes nothing", method: http.MethodPost, origin: testSiteOrigin, baseURL: testSiteOrigin + "/", want: http.StatusOK},
		{name: "the native app passes", method: http.MethodPost, origin: "capacitor://localhost", baseURL: testSiteOrigin, want: http.StatusOK},
		{name: "a client with no origin passes", method: http.MethodPost, origin: "", baseURL: testSiteOrigin, want: http.StatusOK},
		{name: "a post from another site is refused", method: http.MethodPost, origin: "https://evil.example", baseURL: testSiteOrigin, want: http.StatusForbidden},
		{name: "a put from another site is refused", method: http.MethodPut, origin: "https://evil.example", baseURL: testSiteOrigin, want: http.StatusForbidden},
		{name: "a patch from another site is refused", method: http.MethodPatch, origin: "https://evil.example", baseURL: testSiteOrigin, want: http.StatusForbidden},
		{name: "a delete from another site is refused", method: http.MethodDelete, origin: "https://evil.example", baseURL: testSiteOrigin, want: http.StatusForbidden},
		{name: "a sandboxed page is refused", method: http.MethodPost, origin: "null", baseURL: testSiteOrigin, want: http.StatusForbidden},
		{name: "a lookalike subdomain is refused", method: http.MethodPost, origin: "https://whentheycry.example.evil.example", baseURL: testSiteOrigin, want: http.StatusForbidden},
		{name: "a read from another site passes", method: http.MethodGet, origin: "https://evil.example", baseURL: testSiteOrigin, want: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return(tc.baseURL).Maybe()

			app := fiber.New()
			app.Use(RejectForeignOrigins(settingsSvc))
			app.Add([]string{tc.method}, "/write", func(ctx fiber.Ctx) error {
				return ctx.SendString("ok")
			})

			req := httptest.NewRequest(tc.method, "/write", nil)
			if tc.origin != "" {
				req.Header.Set(fiber.HeaderOrigin, tc.origin)
			}

			// when
			resp, err := app.Test(req)

			// then
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.want, resp.StatusCode)
		})
	}
}
