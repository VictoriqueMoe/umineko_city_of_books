package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRateLimitByAccount_KeysOnBodyFieldNotIP(t *testing.T) {
	cases := []struct {
		name       string
		bodyFor    func(call int) string
		ipFor      func(call int) string
		wantStatus int
	}{
		{"rotating IPs cannot escape one account's budget", func(int) string { return `{"username":"Beatrice","password":"x"}` }, func(call int) string { return fmt.Sprintf("203.0.113.%d", call) }, http.StatusTooManyRequests},
		{"case and whitespace do not make a new key", func(call int) string {
			if call%2 == 0 {
				return `{"username":" beatrice ","password":"x"}`
			}
			return `{"username":"BEATRICE","password":"x"}`
		}, func(call int) string { return fmt.Sprintf("203.0.113.%d", call) }, http.StatusTooManyRequests},
		{"different accounts keep separate budgets", func(call int) string { return fmt.Sprintf(`{"username":"user%d","password":"x"}`, call) }, func(int) string { return "203.0.113.7" }, http.StatusOK},
		{"unparseable body falls back to the client IP", func(int) string { return "not json" }, func(int) string { return "203.0.113.7" }, http.StatusTooManyRequests},
		{"missing field falls back to the client IP", func(int) string { return `{"password":"x"}` }, func(int) string { return "203.0.113.7" }, http.StatusTooManyRequests},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := fiber.New()
			call := 0
			app.Use(func(ctx fiber.Ctx) error {
				ctx.Locals("client_ip", tc.ipFor(call))
				return ctx.Next()
			})
			app.Post("/limited", RateLimitCredentialsByAccount("username"), func(ctx fiber.Ctx) error {
				return ctx.JSON(fiber.Map{"status": "ok"})
			})

			// when
			var lastStatus int
			for call = 0; call <= CredentialAttemptsPerMinute; call++ {
				req := httptest.NewRequest("POST", "/limited", strings.NewReader(tc.bodyFor(call)))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				require.NoError(t, err)
				lastStatus = resp.StatusCode
				resp.Body.Close()
			}

			// then
			assert.Equal(t, tc.wantStatus, lastStatus)
		})
	}
}
