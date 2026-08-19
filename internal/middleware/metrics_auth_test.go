package middleware

import (
	"net/http/httptest"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func metricsApp(t *testing.T, configured string) *fiber.App {
	t.Helper()

	svc := settings.NewMockService(t)
	svc.EXPECT().Get(mock.Anything, config.SettingMetricsToken).Return(configured).Maybe()

	app := fiber.New()
	app.Get("/metrics", RequireMetricsToken(svc), func(ctx fiber.Ctx) error {
		return ctx.SendString("go_goroutines 12")
	})

	return app
}

func getMetrics(t *testing.T, app *fiber.App, header, value string) int {
	t.Helper()

	req := httptest.NewRequest("GET", "/metrics", nil)
	if header != "" {
		req.Header.Set(header, value)
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.StatusCode
}

func TestRequireMetricsToken_OpenWhenNoTokenIsConfigured(t *testing.T) {
	// given scraping must keep working until an operator opts in
	app := metricsApp(t, "")

	// when
	status := getMetrics(t, app, "", "")

	// then
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireMetricsToken_AcceptsTheConfiguredToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{name: "bearer authorization", header: "Authorization", value: "Bearer s3cret"},
		{name: "explicit metrics header", header: "X-Metrics-Token", value: "s3cret"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := metricsApp(t, "s3cret")

			// when
			status := getMetrics(t, app, tc.header, tc.value)

			// then
			assert.Equal(t, fiber.StatusOK, status)
		})
	}
}

func TestRequireMetricsToken_HidesTheEndpointWithoutTheToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{name: "no credentials at all", header: "", value: ""},
		{name: "wrong token", header: "X-Metrics-Token", value: "guess"},
		{name: "wrong bearer", header: "Authorization", value: "Bearer guess"},
		{name: "empty bearer", header: "Authorization", value: "Bearer "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := metricsApp(t, "s3cret")

			// when
			status := getMetrics(t, app, tc.header, tc.value)

			// then a 404 rather than a 401, so the endpoint does not advertise that it exists
			assert.Equal(t, fiber.StatusNotFound, status)
		})
	}
}
