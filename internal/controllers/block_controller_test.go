package controllers

import (
	"errors"
	"net/http"
	"testing"

	blocksvc "umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newBlockHarness(t *testing.T) (*testutil.Harness, *blocksvc.MockService) {
	h := testutil.NewHarness(t)
	bs := blocksvc.NewMockService(t)

	s := &Service{
		BlockService: bs,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllBlockRoutes() {
		setup(h.App)
	}
	return h, bs
}

func TestBlockUser_OK(t *testing.T) {
	// given
	h, bs := newBlockHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	bs.EXPECT().Block(mock.Anything, userID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/users/"+targetID.String()+"/block").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestBlockUser_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newBlockHarness, "POST", "/users/"+uuid.New().String()+"/block", nil)
}

func TestBlockUser_InvalidID(t *testing.T) {
	// given
	h, _ := newBlockHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/users/not-a-uuid/block").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestBlockUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"cannot block self", blocksvc.ErrCannotBlockSelf, http.StatusBadRequest, blocksvc.ErrCannotBlockSelf.Error()},
		{"cannot block staff", blocksvc.ErrCannotBlockStaff, http.StatusForbidden, blocksvc.ErrCannotBlockStaff.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to block user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, bs := newBlockHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			bs.EXPECT().Block(mock.Anything, userID, targetID).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/users/"+targetID.String()+"/block").
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnblockUser_OK(t *testing.T) {
	// given
	h, bs := newBlockHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	bs.EXPECT().Unblock(mock.Anything, userID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/users/"+targetID.String()+"/block").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnblockUser_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newBlockHarness, "DELETE", "/users/"+uuid.New().String()+"/block", nil)
}

func TestUnblockUser_InvalidID(t *testing.T) {
	// given
	h, _ := newBlockHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("DELETE", "/users/not-a-uuid/block").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnblockUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to unblock user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, bs := newBlockHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			bs.EXPECT().Unblock(mock.Anything, userID, targetID).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("DELETE", "/users/"+targetID.String()+"/block").
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetBlockStatus_Anonymous_ReturnsFalse(t *testing.T) {
	// given
	h, _ := newBlockHarness(t)
	targetID := uuid.New()

	// when
	status, body := h.NewRequest("GET", "/users/"+targetID.String()+"/block-status").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]bool](t, body)
	assert.False(t, got["blocking"])
	assert.False(t, got["blocked_by"])
}

func TestGetBlockStatus_InvalidCookie_TreatedAsAnonymous(t *testing.T) {
	// given
	h, _ := newBlockHarness(t)
	targetID := uuid.New()
	h.ExpectInvalidSession("bogus")

	// when
	status, body := h.NewRequest("GET", "/users/"+targetID.String()+"/block-status").
		WithCookie("bogus").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]bool](t, body)
	assert.False(t, got["blocking"])
	assert.False(t, got["blocked_by"])
}

func TestGetBlockStatus_Authenticated_OK(t *testing.T) {
	// given
	h, bs := newBlockHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	bs.EXPECT().IsBlocked(mock.Anything, userID, targetID).Return(true, nil)
	bs.EXPECT().IsBlocked(mock.Anything, targetID, userID).Return(false, nil)

	// when
	status, body := h.NewRequest("GET", "/users/"+targetID.String()+"/block-status").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]bool](t, body)
	assert.True(t, got["blocking"])
	assert.False(t, got["blocked_by"])
}

func TestGetBlockStatus_InvalidID(t *testing.T) {
	// given
	h, _ := newBlockHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/block-status").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetBlockStatus_ServiceErrors(t *testing.T) {
	cases := []struct {
		name           string
		blockingResult bool
		blockingErr    error
		expectSecond   bool
		blockedByErr   error
		wantCode       int
	}{
		{"first call fails", false, errors.New("boom"), false, nil, http.StatusInternalServerError},
		{"second call fails", false, nil, true, errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, bs := newBlockHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			bs.EXPECT().IsBlocked(mock.Anything, userID, targetID).Return(tc.blockingResult, tc.blockingErr)
			if tc.expectSecond {
				bs.EXPECT().IsBlocked(mock.Anything, targetID, userID).Return(false, tc.blockedByErr)
			}

			// when
			status, body := h.NewRequest("GET", "/users/"+targetID.String()+"/block-status").
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), "failed to check block status")
		})
	}
}

func TestListBlockedUsers_OK(t *testing.T) {
	// given
	h, bs := newBlockHarness(t)
	userID := uuid.New()
	blockedID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	users := []repository.BlockedUser{
		{
			ID:          blockedID,
			Username:    "kinzo",
			DisplayName: "Kinzo Ushiromiya",
			AvatarURL:   "/avatar.png",
			BlockedAt:   "2025-01-01T00:00:00Z",
		},
	}
	bs.EXPECT().GetBlockedUsers(mock.Anything, userID).Return(users, nil)

	// when
	status, body := h.NewRequest("GET", "/blocked-users").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string][]map[string]any](t, body)
	require.Len(t, got["users"], 1)
	assert.Equal(t, blockedID.String(), got["users"][0]["id"])
	assert.Equal(t, "kinzo", got["users"][0]["username"])
	assert.Equal(t, "Kinzo Ushiromiya", got["users"][0]["display_name"])
}

func TestListBlockedUsers_EmptyList_ReturnsEmptyArray(t *testing.T) {
	// given
	h, bs := newBlockHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	bs.EXPECT().GetBlockedUsers(mock.Anything, userID).Return(nil, nil)

	// when
	status, body := h.NewRequest("GET", "/blocked-users").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), `"users":[]`)
}

func TestListBlockedUsers_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newBlockHarness, "GET", "/blocked-users", nil)
}

func TestListBlockedUsers_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to list blocked users"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, bs := newBlockHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			bs.EXPECT().GetBlockedUsers(mock.Anything, userID).Return(nil, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/blocked-users").
				WithCookie("valid-cookie").
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
