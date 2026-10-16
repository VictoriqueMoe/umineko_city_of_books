package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

type (
	limitedResponse struct {
		status     int
		body       []byte
		retryAfter string
	}
)

func newTestRateLimiter() *RateLimiter {
	return NewRateLimiter(cache.NewManager(engines.NewInMemory(0)))
}

func postLimited(t *testing.T, app *fiber.App, path string) limitedResponse {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest("POST", path, nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return limitedResponse{status: resp.StatusCode, body: body, retryAfter: resp.Header.Get(fiber.HeaderRetryAfter)}
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
		name        string
		policy      RateLimitPolicy
		allowed     int
		window      time.Duration
		wantMessage string
	}{
		{"credentials", CredentialAttempts, CredentialAttemptsPerMinute, time.Minute, "too many requests, please try again later"},
		{"mail", MailAttempts, MailAttemptsPerHour, time.Hour, "too many requests, please try again later"},
		{"chat sends speak to the sender", ChatSends, ChatSendsPerWindow, 5 * time.Second, "You're sending messages too quickly. Wait a moment and try again."},
		{"link previews", LinkPreviews, LinkPreviewsPerMinute, time.Minute, "too many requests, please try again later"},
		{"stream viewer tokens", StreamViewerTokens, StreamViewerTokensPerMinute, time.Minute, "too many requests, please try again later"},
		{"stream thumbnails", StreamThumbnails, StreamThumbnailsPerMinute, time.Minute, "too many requests, please try again later"},
		{"stream credential resets", StreamCredentialsResets, StreamCredentialsResetsPerMinute, time.Minute, "too many requests, please try again later"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := limitedApp(newTestRateLimiter().Limit(tc.policy, ByClientIP))

			// when
			allowedStatuses := make([]int, 0, tc.allowed)
			for range tc.allowed {
				allowedStatuses = append(allowedStatuses, postLimited(t, app, "/limited").status)
			}
			overflow := postLimited(t, app, "/limited")

			// then
			for _, status := range allowedStatuses {
				assert.Equal(t, http.StatusOK, status)
			}
			require.Equal(t, http.StatusTooManyRequests, overflow.status)

			var payload map[string]string
			require.NoError(t, json.Unmarshal(overflow.body, &payload))
			assert.Equal(t, tc.wantMessage, payload["error"])

			retryAfter, err := strconv.Atoi(overflow.retryAfter)
			require.NoError(t, err)
			assert.Positive(t, retryAfter)
			assert.LessOrEqual(t, retryAfter, int(tc.window/time.Second))
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
				ctx.SetLocal("client_ip", tc.ipForCall(call))
				return ctx.Next()
			})
			app.Post("/limited", newTestRateLimiter().Limit(MailAttempts, ByClientIP), func(ctx fiber.Ctx) error {
				return ctx.JSON(fiber.Map{"status": "ok"})
			})

			// when
			var lastStatus int
			for range MailAttemptsPerHour + 1 {
				lastStatus = postLimited(t, app, "/limited").status
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
		{"an oversized account value is still limited", func(int) string { return `{"username":"` + strings.Repeat("a", 4096) + `","password":"x"}` }, func(call int) string { return fmt.Sprintf("203.0.113.%d", call) }, http.StatusTooManyRequests},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := fiber.New()
			call := 0
			app.Use(func(ctx fiber.Ctx) error {
				ctx.SetLocal("client_ip", tc.ipFor(call))
				return ctx.Next()
			})
			app.Post("/limited", newTestRateLimiter().Limit(CredentialAttempts, ByAccount("username")), func(ctx fiber.Ctx) error {
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

func TestRateLimit_RouteScope(t *testing.T) {
	cases := []struct {
		name       string
		policy     RateLimitPolicy
		subject    RateLimitSubject
		wantStatus int
	}{
		{"a per route policy gives every route its own budget", CredentialAttempts, ByClientIP, http.StatusOK},
		{"a shared policy spends one budget across routes", ChatSends, ByUser, http.StatusTooManyRequests},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			userID := uuid.New()
			limit := newTestRateLimiter().Limit(tc.policy, tc.subject)

			app := fiber.New()
			app.Use(func(ctx fiber.Ctx) error {
				ctx.SetLocal("client_ip", "203.0.113.7")
				ctx.SetLocal("userID", userID)
				return ctx.Next()
			})
			for _, path := range []string{"/first", "/second"} {
				app.Post(path, limit, func(ctx fiber.Ctx) error {
					return ctx.JSON(fiber.Map{"status": "ok"})
				})
			}

			// when
			for range tc.policy.Limit {
				require.Equal(t, http.StatusOK, postLimited(t, app, "/first").status)
			}
			got := postLimited(t, app, "/second")

			// then
			assert.Equal(t, tc.wantStatus, got.status)
		})
	}
}

func TestRateLimitByUser_RefusesToRunWithoutAUser(t *testing.T) {
	// given
	reached := false
	app := fiber.New()
	app.Post("/limited", newTestRateLimiter().Limit(ChatSends, ByUser), func(ctx fiber.Ctx) error {
		reached = true
		return ctx.JSON(fiber.Map{"status": "ok"})
	})

	// when
	got := postLimited(t, app, "/limited")

	// then a mis-mounted limiter is a loud wiring error, never one shared anonymous budget
	assert.Equal(t, http.StatusInternalServerError, got.status)
	assert.False(t, reached)
}

func TestRateLimit_FailsOpenWhenTheStoreFails(t *testing.T) {
	cases := []struct {
		name         string
		manager      func(t *testing.T) *cache.Manager
		wantOverflow int
	}{
		{
			name: "valkey failing lets that request through and the fallback counts the rest",
			manager: func(t *testing.T) *cache.Manager {
				client := valkeymock.NewClient(gomock.NewController(t))
				client.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(valkeymock.ErrorResult(errors.New("connection refused"))).
					Times(1)

				return cache.NewManager(engines.NewValkeyWithClient(client), engines.NewInMemory(0))
			},
			wantOverflow: http.StatusTooManyRequests,
		},
		{
			name: "no engine at all lets every request through",
			manager: func(*testing.T) *cache.Manager {
				return cache.NewManager()
			},
			wantOverflow: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := limitedApp(NewRateLimiter(tc.manager(t)).Limit(MailAttempts, ByClientIP))

			// when
			allowedStatuses := make([]int, 0, MailAttemptsPerHour+1)
			for range MailAttemptsPerHour + 1 {
				allowedStatuses = append(allowedStatuses, postLimited(t, app, "/limited").status)
			}
			overflow := postLimited(t, app, "/limited")

			// then
			for _, status := range allowedStatuses {
				assert.Equal(t, http.StatusOK, status)
			}
			assert.Equal(t, tc.wantOverflow, overflow.status)
		})
	}
}

func TestBoundSubject(t *testing.T) {
	long := "account:username:" + strings.Repeat("a", maxSubjectLength)

	cases := []struct {
		name       string
		subject    string
		wantKept   bool
		wantHashed bool
	}{
		{"a normal subject is kept verbatim", "ip:203.0.113.7", true, false},
		{"a subject at the limit is kept verbatim", strings.Repeat("a", maxSubjectLength), true, false},
		{"an oversized subject is hashed", long, false, true},
		{"a subject claiming the hashed prefix is hashed", hashedSubjectPrefix + "deadbeef", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := boundSubject(tc.subject)

			// then
			assert.LessOrEqual(t, len(got), maxSubjectLength)
			assert.Equal(t, tc.wantKept, got == tc.subject)
			assert.Equal(t, tc.wantHashed, strings.HasPrefix(got, hashedSubjectPrefix))
		})
	}
}

func TestBoundSubject_DistinctOversizedSubjectsStayDistinct(t *testing.T) {
	// given
	first := strings.Repeat("a", 4096)
	second := strings.Repeat("a", 4095) + "b"

	// when
	firstBound := boundSubject(first)
	secondBound := boundSubject(second)

	// then
	assert.NotEqual(t, firstBound, secondBound)
	assert.Equal(t, firstBound, boundSubject(first))
}

func TestRetryAfterSeconds(t *testing.T) {
	cases := []struct {
		name      string
		remaining time.Duration
		want      int64
	}{
		{"an elapsed window still asks for a second", 0, 1},
		{"a millisecond rounds up", time.Millisecond, 1},
		{"a whole second", time.Second, 1},
		{"just over a second rounds up", time.Second + time.Millisecond, 2},
		{"a full minute", time.Minute, 60},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, retryAfterSeconds(tc.remaining))
		})
	}
}
