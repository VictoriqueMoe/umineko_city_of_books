package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authzsvc "umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/session"
	settingssvc "umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// RunAuthFailureSuite runs the three standard auth failure cases against a
// protected route: missing cookie (401), invalid session cookie (401), and
// banned user (403). The factory is called fresh for each subtest so mocks
// are isolated. If body is non-nil it is sent as JSON.
func RunAuthFailureSuite[H any](t *testing.T, factory func(t *testing.T) (*Harness, H), method, path string, body any) {
	t.Helper()

	cases := []struct {
		name     string
		cookie   string
		setup    func(*Harness)
		wantCode int
		wantBody string
	}{
		{
			name:     "no cookie",
			cookie:   "",
			setup:    func(_ *Harness) {},
			wantCode: http.StatusUnauthorized,
			wantBody: "authentication required",
		},
		{
			name:     "invalid session",
			cookie:   "bogus",
			setup:    func(h *Harness) { h.ExpectInvalidSession("bogus") },
			wantCode: http.StatusUnauthorized,
			wantBody: "invalid or expired session",
		},
		{
			name:     "banned",
			cookie:   "banned-cookie",
			setup:    func(h *Harness) { h.ExpectBannedUser("banned-cookie", uuid.New()) },
			wantCode: http.StatusForbidden,
			wantBody: "banned",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := factory(t)
			tc.setup(h)

			// when
			req := h.NewRequest(method, path)
			if body != nil {
				req = req.WithJSONBody(body)
			}
			if tc.cookie != "" {
				req = req.WithCookie(tc.cookie)
			}
			status, respBody := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(respBody), tc.wantBody)
		})
	}
}

// Harness wires a real session.Manager to mocked dependencies so tests can
// exercise the full route chain (middleware + handler) against a fiber.App.
//
// The harness exposes the mocked repos/services so individual tests can
// customise expectations on top of the defaults set up by NewHarness.
type Harness struct {
	T               *testing.T
	App             *fiber.App
	SessionManager  *session.Manager
	SessionRepo     *repository.MockSessionRepository
	SettingsService *settingssvc.MockService
	AuthzService    *authzsvc.MockService
}

// NewHarness creates a harness with a fresh fiber app and default-mocked
// session infrastructure. Callers register routes on harness.App after
// constructing a controllers.Service that uses harness.SessionManager and
// harness.AuthzService.
func NewHarness(t *testing.T) *Harness {
	t.Helper()
	sessionRepo := repository.NewMockSessionRepository(t)
	settingsService := settingssvc.NewMockService(t)
	authzService := authzsvc.NewMockService(t)

	// session.Manager reads SettingSessionDurationDays during Create; stub a sane default.
	settingsService.EXPECT().GetInt(mock.Anything, mock.Anything).Return(30).Maybe()

	mgr := session.NewManager(sessionRepo, settingsService)

	return &Harness{
		T:               t,
		App:             fiber.New(),
		SessionManager:  mgr,
		SessionRepo:     sessionRepo,
		SettingsService: settingsService,
		AuthzService:    authzService,
	}
}

// ExpectValidSession registers a SessionRepo expectation that `cookie` maps to
// `userID` and IsBanned(userID) returns false. This is the happy path setup
// for any authenticated route under test.
func (h *Harness) ExpectValidSession(cookie string, userID uuid.UUID) {
	h.T.Helper()
	h.SessionRepo.EXPECT().
		GetUserID(mock.Anything, cookie).
		Return(userID, time.Now().Add(time.Hour), nil).
		Maybe()
	h.AuthzService.EXPECT().
		IsBanned(mock.Anything, userID).
		Return(false).
		Maybe()
}

// ExpectInvalidSession makes `cookie` lookup fail with a not-found-style error.
func (h *Harness) ExpectInvalidSession(cookie string) {
	h.T.Helper()
	h.SessionRepo.EXPECT().
		GetUserID(mock.Anything, cookie).
		Return(uuid.Nil, time.Time{}, errNotFound).
		Maybe()
}

// ExpectBannedUser makes `cookie` resolve to `userID` but the authz service
// reports the user as banned. This exercises the 403 branch in RequireAuth.
func (h *Harness) ExpectBannedUser(cookie string, userID uuid.UUID) {
	h.T.Helper()
	h.SessionRepo.EXPECT().
		GetUserID(mock.Anything, cookie).
		Return(userID, time.Now().Add(time.Hour), nil).
		Maybe()
	h.AuthzService.EXPECT().
		IsBanned(mock.Anything, userID).
		Return(true).
		Maybe()
	h.SessionRepo.EXPECT().
		Delete(mock.Anything, cookie).
		Return(nil).
		Maybe()
}

// ExpectHasPermission registers Can(userID, perm) → granted on the authz mock.
// Combine with ExpectValidSession to set up a fully-authorised user for routes
// gated by middleware.RequirePermission.
func (h *Harness) ExpectHasPermission(userID uuid.UUID, perm authzsvc.Permission, granted bool) {
	h.T.Helper()
	h.AuthzService.EXPECT().
		Can(mock.Anything, userID, perm).
		Return(granted).
		Maybe()
}

// RunPermissionFailureSuite runs the four standard failure cases against a
// permission-gated route: missing cookie (401), invalid session (401), banned
// user (403), and authenticated-but-lacking-permission (403).
func RunPermissionFailureSuite[H any](t *testing.T, factory func(t *testing.T) (*Harness, H), method, path string, body any, perm authzsvc.Permission) {
	t.Helper()

	cases := []struct {
		name     string
		cookie   string
		setup    func(*Harness)
		wantCode int
		wantBody string
	}{
		{
			name:     "no cookie",
			cookie:   "",
			setup:    func(_ *Harness) {},
			wantCode: http.StatusUnauthorized,
			wantBody: "authentication required",
		},
		{
			name:     "invalid session",
			cookie:   "bogus",
			setup:    func(h *Harness) { h.ExpectInvalidSession("bogus") },
			wantCode: http.StatusUnauthorized,
			wantBody: "invalid or expired session",
		},
		{
			name:     "banned",
			cookie:   "banned-cookie",
			setup:    func(h *Harness) { h.ExpectBannedUser("banned-cookie", uuid.New()) },
			wantCode: http.StatusForbidden,
			wantBody: "banned",
		},
		{
			name:   "insufficient permissions",
			cookie: "valid-cookie",
			setup: func(h *Harness) {
				userID := uuid.New()
				h.ExpectValidSession("valid-cookie", userID)
				h.ExpectHasPermission(userID, perm, false)
			},
			wantCode: http.StatusForbidden,
			wantBody: "insufficient permissions",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := factory(t)
			tc.setup(h)

			// when
			req := h.NewRequest(method, path)
			if body != nil {
				req = req.WithJSONBody(body)
			}
			if tc.cookie != "" {
				req = req.WithCookie(tc.cookie)
			}
			status, respBody := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(respBody), tc.wantBody)
		})
	}
}

// Request is a fluent builder for issuing HTTP requests against the harness app.
type Request struct {
	h       *Harness
	method  string
	path    string
	body    io.Reader
	cookie  string
	headers map[string]string
}

func (h *Harness) NewRequest(method, path string) *Request {
	return &Request{h: h, method: method, path: path, headers: map[string]string{}}
}

func (r *Request) WithCookie(value string) *Request {
	r.cookie = value
	return r
}

func (r *Request) WithJSONBody(v any) *Request {
	data, err := json.Marshal(v)
	require.NoError(r.h.T, err)
	r.body = bytes.NewReader(data)
	r.headers["Content-Type"] = "application/json"
	return r
}

func (r *Request) WithRawBody(s string, contentType string) *Request {
	r.body = bytes.NewReader([]byte(s))
	if contentType != "" {
		r.headers["Content-Type"] = contentType
	}
	return r
}

func (r *Request) WithHeader(k, v string) *Request {
	r.headers[k] = v
	return r
}

// Do executes the request and returns status code + body bytes.
func (r *Request) Do() (int, []byte) {
	r.h.T.Helper()
	req := httptest.NewRequest(r.method, r.path, r.body)
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	if r.cookie != "" {
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: r.cookie})
	}
	resp, err := r.h.App.Test(req)
	require.NoError(r.h.T, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(r.h.T, err)
	return resp.StatusCode, data
}

// UnmarshalJSON is a convenience to unmarshal the response body.
func UnmarshalJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(body, &v))
	return v
}

// sentinel used by ExpectInvalidSession so callers don't need to import
// database/sql or similar just to fabricate a "session missing" error.
type errNotFoundT struct{}

func (errNotFoundT) Error() string { return "session not found" }

var errNotFound error = errNotFoundT{}
