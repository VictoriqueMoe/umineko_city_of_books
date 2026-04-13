package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type userPrefsDeps struct {
	UserRepo *repository.MockUserRepository
}

func newUserPrefsHarness(t *testing.T) (*testutil.Harness, userPrefsDeps) {
	h := testutil.NewHarness(t)
	ur := repository.NewMockUserRepository(t)

	s := &Service{
		UserRepo:     ur,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllUserPreferencesRoutes() {
		setup(h.App)
	}
	return h, userPrefsDeps{UserRepo: ur}
}

func userPrefsFactory(t *testing.T) (*testutil.Harness, userPrefsDeps) {
	return newUserPrefsHarness(t)
}

func TestUpdateGameBoardSort_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, userPrefsFactory, "PUT", "/preferences/game-board-sort", map[string]string{"sort": "newest"})
}

func TestUpdateGameBoardSort_OK(t *testing.T) {
	// given
	h, deps := newUserPrefsHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.UserRepo.EXPECT().UpdateGameBoardSort(mock.Anything, userID, "newest").Return(nil)

	// when
	status, body := h.NewRequest("PUT", "/preferences/game-board-sort").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"sort": "newest"}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
	assert.Empty(t, body)
}

func TestUpdateGameBoardSort_BadJSON(t *testing.T) {
	// given
	h, _ := newUserPrefsHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("PUT", "/preferences/game-board-sort").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestUpdateGameBoardSort_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		repoErr  error
		wantCode int
		wantBody string
	}{
		{"internal repo error", errors.New("boom"), http.StatusInternalServerError, "failed to save"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newUserPrefsHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.UserRepo.EXPECT().UpdateGameBoardSort(mock.Anything, userID, "oldest").Return(tc.repoErr)

			// when
			status, body := h.NewRequest("PUT", "/preferences/game-board-sort").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]string{"sort": "oldest"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateAppearance_AuthFailures(t *testing.T) {
	body := map[string]any{"theme": "dark", "font": "serif", "wide_layout": true}
	testutil.RunAuthFailureSuite(t, userPrefsFactory, "PUT", "/preferences/appearance", body)
}

func TestUpdateAppearance_OK(t *testing.T) {
	// given
	h, deps := newUserPrefsHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.UserRepo.EXPECT().
		UpdateAppearance(mock.Anything, userID, "dark", "serif", true).
		Return(nil)

	// when
	status, body := h.NewRequest("PUT", "/preferences/appearance").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"theme": "dark", "font": "serif", "wide_layout": true}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
	assert.Empty(t, body)
}

func TestUpdateAppearance_DefaultValues(t *testing.T) {
	// given
	h, deps := newUserPrefsHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.UserRepo.EXPECT().
		UpdateAppearance(mock.Anything, userID, "", "", false).
		Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/preferences/appearance").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateAppearance_BadJSON(t *testing.T) {
	// given
	h, _ := newUserPrefsHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("PUT", "/preferences/appearance").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request")
}

func TestUpdateAppearance_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		repoErr  error
		wantCode int
		wantBody string
	}{
		{"internal repo error", errors.New("boom"), http.StatusInternalServerError, "failed to save"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newUserPrefsHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.UserRepo.EXPECT().
				UpdateAppearance(mock.Anything, userID, "light", "mono", false).
				Return(tc.repoErr)

			// when
			status, body := h.NewRequest("PUT", "/preferences/appearance").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]any{"theme": "light", "font": "mono", "wide_layout": false}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
