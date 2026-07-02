package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/overlay"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newOverlayHarness(t *testing.T) (*testutil.Harness, *overlay.MockService) {
	h := testutil.NewHarness(t)
	svc := overlay.NewMockService(t)
	NewOverlayHandler(svc, h.SessionManager, h.AuthzService).Register(h.App)
	return h, svc
}

func TestOverlayToken_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newOverlayHarness, "GET", "/api/v1/overlay/token", nil)
}

func TestOverlayGetToken_OK(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().Token(mock.Anything, userID).Return("tok_123", nil)
	svc.EXPECT().ConnectURL(mock.Anything, userID).Return("wss://site/api/v1/overlay?token=tok_123", nil)
	svc.EXPECT().IsConnected(userID).Return(true)

	// when
	status, body := h.NewRequest("GET", "/api/v1/overlay/token").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	resp := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.Equal(t, "tok_123", resp["token"])
	assert.Equal(t, "wss://site/api/v1/overlay?token=tok_123", resp["connect_url"])
	assert.Equal(t, true, resp["connected"])
}

func TestOverlayGetToken_TokenError(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().Token(mock.Anything, userID).Return("", errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/api/v1/overlay/token").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestOverlayResetToken_OK(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().ResetToken(mock.Anything, userID).Return("tok_new", nil)
	svc.EXPECT().Token(mock.Anything, userID).Return("tok_new", nil)
	svc.EXPECT().ConnectURL(mock.Anything, userID).Return("wss://site/api/v1/overlay?token=tok_new", nil)
	svc.EXPECT().IsConnected(userID).Return(false)

	// when
	status, body := h.NewRequest("POST", "/api/v1/overlay/token/reset").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	resp := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.Equal(t, "tok_new", resp["token"])
	assert.Equal(t, false, resp["connected"])
}

func TestOverlayResetToken_Error(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().ResetToken(mock.Anything, userID).Return("", errors.New("boom"))

	// when
	status, _ := h.NewRequest("POST", "/api/v1/overlay/token/reset").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestOverlayDownloadSEF_OK(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().BuildSEF(mock.Anything, userID).Return("[extension_name]\nUmineko Overlay", nil)

	// when
	status, body := h.NewRequest("GET", "/api/v1/overlay/connector.sef").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "[extension_name]")
}

func TestOverlayDownloadSEF_Error(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().BuildSEF(mock.Anything, userID).Return("", errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/api/v1/overlay/connector.sef").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestOverlayTest_OK(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().TestFire(userID).Return(nil)

	// when
	status, body := h.NewRequest("POST", "/api/v1/overlay/test").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	resp := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.Equal(t, true, resp["ok"])
}

func TestOverlayTest_NotConnected(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().TestFire(userID).Return(overlay.ErrNotConnected)

	// when
	status, body := h.NewRequest("POST", "/api/v1/overlay/test").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusConflict, status)
	assert.Contains(t, string(body), "no overlay connection")
}

func TestOverlayTest_Error(t *testing.T) {
	// given
	h, svc := newOverlayHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	svc.EXPECT().TestFire(userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("POST", "/api/v1/overlay/test").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}
