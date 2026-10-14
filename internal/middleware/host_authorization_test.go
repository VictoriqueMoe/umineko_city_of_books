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

func TestHostAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		host       string
		wantStatus int
	}{
		{name: "matching host", baseURL: "https://whentheycry.social", host: "whentheycry.social", wantStatus: http.StatusOK},
		{name: "other host", baseURL: "https://whentheycry.social", host: "evil.example", wantStatus: http.StatusForbidden},
		{name: "loopback ip against a public base url", baseURL: "https://whentheycry.social", host: "127.0.0.1", wantStatus: http.StatusForbidden},
		{name: "localhost", baseURL: "http://localhost:4323", host: "localhost:4323", wantStatus: http.StatusOK},
		{name: "ipv4 loopback against localhost", baseURL: "http://localhost:4323", host: "127.0.0.1:4323", wantStatus: http.StatusOK},
		{name: "ipv6 loopback against localhost", baseURL: "http://localhost:4323", host: "[::1]:4323", wantStatus: http.StatusOK},
		{name: "lan ip against localhost", baseURL: "http://localhost:4323", host: "192.168.1.20:4323", wantStatus: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return(tc.baseURL).Maybe()
			app := fiber.New()
			app.Use(HostAuthorization(settingsSvc))
			app.Get("/page", func(ctx fiber.Ctx) error {
				return ctx.SendStatus(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/page", nil)
			req.Host = tc.host

			// when
			resp, err := app.Test(req)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}
