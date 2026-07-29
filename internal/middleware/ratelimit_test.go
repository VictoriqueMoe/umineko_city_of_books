package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postLimited(t *testing.T, app *fiber.App) (int, []byte) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("POST", "/limited", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

func limitedApp(limit fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Post("/limited", limit, func(ctx fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"status": "ok"})
	})
	return app
}

func TestRateLimitByClientIP_AllowsBudgetThenRejects(t *testing.T) {
	cases := []struct {
		name    string
		limit   func() fiber.Handler
		allowed int
	}{
		{"credentials", RateLimitCredentials, CredentialAttemptsPerMinute},
		{"mail", RateLimitMail, MailAttemptsPerHour},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := limitedApp(tc.limit())

			// when
			allowedStatuses := make([]int, 0, tc.allowed)
			for range tc.allowed {
				status, _ := postLimited(t, app)
				allowedStatuses = append(allowedStatuses, status)
			}
			overflowStatus, overflowBody := postLimited(t, app)

			// then
			for _, status := range allowedStatuses {
				assert.Equal(t, http.StatusOK, status)
			}
			require.Equal(t, http.StatusTooManyRequests, overflowStatus)

			var payload map[string]string
			require.NoError(t, json.Unmarshal(overflowBody, &payload))
			assert.Equal(t, "too many requests, please try again later", payload["error"])
		})
	}
}

func TestRateLimitByClientIP_KeysOnResolvedClientIP(t *testing.T) {
	cases := []struct {
		name       string
		ipForCall  func(call int) string
		wantStatus int
	}{
		{"one client burns its own budget", func(int) string { return "203.0.113.7" }, http.StatusTooManyRequests},
		{"separate clients keep separate budgets", func(call int) string { return fmt.Sprintf("203.0.113.%d", call) }, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			call := 0
			app := fiber.New()
			app.Use(func(ctx fiber.Ctx) error {
				call++
				ctx.Locals("client_ip", tc.ipForCall(call))
				return ctx.Next()
			})
			app.Post("/limited", RateLimitMail(), func(ctx fiber.Ctx) error {
				return ctx.JSON(fiber.Map{"status": "ok"})
			})

			// when
			var lastStatus int
			for range MailAttemptsPerHour + 1 {
				lastStatus, _ = postLimited(t, app)
			}

			// then
			assert.Equal(t, tc.wantStatus, lastStatus)
		})
	}
}
