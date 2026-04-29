package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/sidebar"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type homeDeps struct {
	homeFeedSvc *homefeed.MockService
	sidebarSvc  *sidebar.MockService
}

func newHomeHarness(t *testing.T) (*testutil.Harness, homeDeps) {
	h := testutil.NewHarness(t)
	deps := homeDeps{
		homeFeedSvc: homefeed.NewMockService(t),
		sidebarSvc:  sidebar.NewMockService(t),
	}

	s := &Service{
		HomeFeedService: deps.homeFeedSvc,
		SidebarService:  deps.sidebarSvc,
		AuthSession:     h.SessionManager,
		AuthzService:    h.AuthzService,
	}
	for _, setup := range s.getAllHomeRoutes() {
		setup(h.App)
	}
	return h, deps
}

func TestGetSidebarActivity_OK(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	deps.homeFeedSvc.EXPECT().SidebarActivity(mock.Anything).Return(&dto.SidebarActivityResponse{
		Activity: map[string]string{
			"game_board_umineko": "2026-04-23T10:00:00Z",
			"rooms":              "2026-04-24T09:30:00Z",
		},
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/sidebar/activity").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SidebarActivityResponse](t, body)
	assert.Equal(t, "2026-04-23T10:00:00Z", got.Activity["game_board_umineko"])
	assert.Equal(t, "2026-04-24T09:30:00Z", got.Activity["rooms"])
}

func TestGetSidebarActivity_ServiceError(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	deps.homeFeedSvc.EXPECT().SidebarActivity(mock.Anything).Return(nil, errors.New("db down"))

	// when
	status, body := h.NewRequest("GET", "/sidebar/activity").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load sidebar activity")
}

func TestGetHomeActivity_OK(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	resp := &dto.HomeActivityResponse{
		OnlineCount:    3,
		RecentActivity: []dto.HomeActivityEntry{{Kind: "theory", URL: "/theory/abc"}},
		RecentMembers:  []dto.HomeMember{{Username: "u"}},
		PublicRooms:    []dto.HomePublicRoom{{Name: "Hangout"}},
		CornerActivity: []dto.HomeCornerActivity{{Corner: "umineko", PostCount: 7}},
	}
	deps.homeFeedSvc.EXPECT().HomeActivity(mock.Anything).Return(resp, nil)

	// when
	status, body := h.NewRequest("GET", "/home/activity").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.HomeActivityResponse](t, body)
	assert.Equal(t, 3, got.OnlineCount)
	assert.Len(t, got.RecentActivity, 1)
	assert.Equal(t, "/theory/abc", got.RecentActivity[0].URL)
}

func TestGetHomeActivity_ServiceError(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	deps.homeFeedSvc.EXPECT().HomeActivity(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/home/activity").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load home activity")
}

func TestGetSidebarLastVisited_OK(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	deps.sidebarSvc.EXPECT().ListVisited(mock.Anything, userID).Return(&dto.SidebarLastVisitedResponse{
		Visited: map[string]string{
			"mysteries": "2026-04-24T10:00:00Z",
			"rooms":     "2026-04-24T11:00:00Z",
		},
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/sidebar/last-visited").WithCookie("valid").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SidebarLastVisitedResponse](t, body)
	assert.Equal(t, "2026-04-24T10:00:00Z", got.Visited["mysteries"])
	assert.Equal(t, "2026-04-24T11:00:00Z", got.Visited["rooms"])
}

func TestGetSidebarLastVisited_ServiceError(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	deps.sidebarSvc.EXPECT().ListVisited(mock.Anything, userID).Return(nil, errors.New("db down"))

	// when
	status, body := h.NewRequest("GET", "/sidebar/last-visited").WithCookie("valid").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load sidebar last visited")
}

func TestGetSidebarLastVisited_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newHomeHarness, "GET", "/sidebar/last-visited", nil)
}

func TestMarkSidebarVisited_OK(t *testing.T) {
	// given
	h, deps := newHomeHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	deps.sidebarSvc.EXPECT().MarkVisited(mock.Anything, userID, "mysteries").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithJSONBody(dto.MarkSidebarVisitedRequest{Key: "mysteries"}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestMarkSidebarVisited_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty key", sidebar.ErrEmptyKey, http.StatusBadRequest, "key is required"},
		{"too long", sidebar.ErrKeyTooLong, http.StatusBadRequest, "key too long"},
		{"internal", errors.New("db down"), http.StatusInternalServerError, "failed to mark sidebar visited"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newHomeHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid", userID)
			deps.sidebarSvc.EXPECT().MarkVisited(mock.Anything, userID, "mysteries").Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/sidebar/last-visited").
				WithCookie("valid").
				WithJSONBody(dto.MarkSidebarVisitedRequest{Key: "mysteries"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestMarkSidebarVisited_InvalidBody(t *testing.T) {
	// given
	h, _ := newHomeHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)

	// when
	status, body := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithRawBody("not-json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestMarkSidebarVisited_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newHomeHarness, "POST", "/sidebar/last-visited",
		dto.MarkSidebarVisitedRequest{Key: "mysteries"})
}
