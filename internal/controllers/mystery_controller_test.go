package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	mysterysvc "umineko_city_of_books/internal/mystery"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newMysteryHarness(t *testing.T) (*testutil.Harness, *mysterysvc.MockService) {
	h := testutil.NewHarness(t)
	ms := mysterysvc.NewMockService(t)

	s := &Service{
		MysteryService: ms,
		AuthSession:    h.SessionManager,
		AuthzService:   h.AuthzService,
	}
	for _, setup := range s.getAllMysteryRoutes() {
		setup(h.App)
	}
	return h, ms
}

func TestGetMystery_Anonymous_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	mysteryID := uuid.New()
	expected := &dto.MysteryDetailResponse{ID: mysteryID, Title: "Legend of the Gold"}
	ms.EXPECT().GetMystery(mock.Anything, mysteryID, uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/mysteries/"+mysteryID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.MysteryDetailResponse](t, body)
	assert.Equal(t, expected.ID, got.ID)
	assert.Equal(t, expected.Title, got.Title)
}

func TestGetMystery_AuthenticatedUser_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().GetMystery(mock.Anything, mysteryID, userID).Return(&dto.MysteryDetailResponse{ID: mysteryID}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries/"+mysteryID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetMystery_InvalidCookie_TreatedAsAnonymous(t *testing.T) {
	// given — OptionalAuth should fall through to uuid.Nil when the cookie is junk.
	h, ms := newMysteryHarness(t)
	mysteryID := uuid.New()
	h.ExpectInvalidSession("bogus")
	ms.EXPECT().GetMystery(mock.Anything, mysteryID, uuid.Nil).Return(&dto.MysteryDetailResponse{ID: mysteryID}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries/"+mysteryID.String()).
		WithCookie("bogus").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetMystery_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)

	// when
	status, body := h.NewRequest("GET", "/mysteries/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetMystery_NotFound(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	mysteryID := uuid.New()
	ms.EXPECT().GetMystery(mock.Anything, mysteryID, uuid.Nil).Return(nil, mysterysvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("GET", "/mysteries/"+mysteryID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetMystery_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	mysteryID := uuid.New()
	ms.EXPECT().GetMystery(mock.Anything, mysteryID, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/mysteries/"+mysteryID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestCreateMystery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries", dto.CreateMysteryRequest{Title: "A Mystery"})
}

func TestCreateMystery_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateMysteryRequest{Title: "A Mystery", Body: "body", Difficulty: "medium"}
	ms.EXPECT().CreateMystery(mock.Anything, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/mysteries").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateMystery_EmptyTitle_BadRequest(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateMysteryRequest{Title: ""}
	ms.EXPECT().CreateMystery(mock.Anything, userID, req).Return(uuid.Nil, mysterysvc.ErrEmptyTitle)

	// when
	status, body := h.NewRequest("POST", "/mysteries").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), mysterysvc.ErrEmptyTitle.Error())
}

func TestCreateMystery_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mysteries").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestDeleteMystery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "DELETE", "/mysteries/"+uuid.NewString(), nil)
}

func TestDeleteMystery_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteMystery_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteMystery(mock.Anything, mysteryID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteMystery_Forbidden(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteMystery(mock.Anything, mysteryID, userID).Return(errors.New("not owner"))

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
}

func TestListMysteries_Anonymous_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	expected := &dto.MysteryListResponse{Total: 0, Limit: 20, Offset: 0}
	ms.EXPECT().ListMysteries(mock.Anything, "new", (*bool)(nil), uuid.Nil, 20, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/mysteries").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.MysteryListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListMysteries_SolvedTrue_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().ListMysteries(mock.Anything, "new", mock.MatchedBy(func(p *bool) bool {
		return p != nil && *p
	}), uuid.Nil, 20, 0).Return(&dto.MysteryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries?solved=true").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListMysteries_SolvedFalse_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().ListMysteries(mock.Anything, "top", mock.MatchedBy(func(p *bool) bool {
		return p != nil && !*p
	}), uuid.Nil, 50, 10).Return(&dto.MysteryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries?sort=top&solved=false&limit=50&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListMysteries_Authenticated_PassesUserID(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().ListMysteries(mock.Anything, "new", (*bool)(nil), userID, 20, 0).Return(&dto.MysteryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListMysteries_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().ListMysteries(mock.Anything, "new", (*bool)(nil), uuid.Nil, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/mysteries").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list mysteries")
}

func TestMysteryLeaderboard_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	expected := &dto.MysteryLeaderboardResponse{}
	ms.EXPECT().GetLeaderboard(mock.Anything, 20).Return(expected, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries/leaderboard").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestMysteryLeaderboard_CustomLimit(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().GetLeaderboard(mock.Anything, 5).Return(&dto.MysteryLeaderboardResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries/leaderboard?limit=5").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestMysteryLeaderboard_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().GetLeaderboard(mock.Anything, 20).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/mysteries/leaderboard").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load leaderboard")
}

func TestGMLeaderboard_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().GetGMLeaderboard(mock.Anything, 20).Return(&dto.GMLeaderboardResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/mysteries/gm-leaderboard").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGMLeaderboard_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	ms.EXPECT().GetGMLeaderboard(mock.Anything, 20).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/mysteries/gm-leaderboard").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load gm leaderboard")
}

func TestListUserMysteries_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	targetUserID := uuid.New()
	ms.EXPECT().ListByUser(mock.Anything, targetUserID, 20, 0).Return(&dto.MysteryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/mysteries").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserMysteries_CustomPaging(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	targetUserID := uuid.New()
	ms.EXPECT().ListByUser(mock.Anything, targetUserID, 5, 10).Return(&dto.MysteryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/mysteries?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserMysteries_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/mysteries").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserMysteries_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	targetUserID := uuid.New()
	ms.EXPECT().ListByUser(mock.Anything, targetUserID, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/mysteries").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestUpdateMystery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "PUT", "/mysteries/"+uuid.NewString(), dto.CreateMysteryRequest{Title: "x"})
}

func TestUpdateMystery_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("PUT", "/mysteries/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateMysteryRequest{Title: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateMystery_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("PUT", "/mysteries/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateMystery_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateMysteryRequest{Title: "Updated", Body: "body"}
	ms.EXPECT().UpdateMystery(mock.Anything, mysteryID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/mysteries/"+mysteryID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUpdateMystery_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateMysteryRequest{Title: "Updated"}
	ms.EXPECT().UpdateMystery(mock.Anything, mysteryID, userID, req).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("PUT", "/mysteries/"+mysteryID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestCreateAttempt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries/"+uuid.NewString()+"/attempts", dto.CreateAttemptRequest{Body: "b"})
}

func TestCreateAttempt_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/attempts").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateAttemptRequest{Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateAttempt_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/attempts").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateAttempt_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateAttemptRequest{Body: "my guess"}
	ms.EXPECT().CreateAttempt(mock.Anything, mysteryID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/attempts").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateAttempt_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty body", mysterysvc.ErrEmptyBody, http.StatusBadRequest},
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound},
		{"already solved", mysterysvc.ErrAlreadySolved, http.StatusForbidden},
		{"cannot reply", mysterysvc.ErrCannotReply, http.StatusForbidden},
		{"paused", mysterysvc.ErrMysteryPaused, http.StatusForbidden},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateAttemptRequest{Body: "x"}
			ms.EXPECT().CreateAttempt(mock.Anything, mysteryID, userID, req).
				Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/attempts").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteAttempt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "DELETE", "/mystery-attempts/"+uuid.NewString(), nil)
}

func TestDeleteAttempt_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/mystery-attempts/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteAttempt_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	attemptID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteAttempt(mock.Anything, attemptID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/mystery-attempts/"+attemptID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteAttempt_Forbidden(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	attemptID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteAttempt(mock.Anything, attemptID, userID).Return(errors.New("not owner"))

	// when
	status, _ := h.NewRequest("DELETE", "/mystery-attempts/"+attemptID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
}

func TestVoteAttempt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mystery-attempts/"+uuid.NewString()+"/vote", dto.VoteRequest{Value: 1})
}

func TestVoteAttempt_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mystery-attempts/not-a-uuid/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestVoteAttempt_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mystery-attempts/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestVoteAttempt_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	attemptID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().VoteAttempt(mock.Anything, attemptID, userID, 1).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/mystery-attempts/"+attemptID.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestVoteAttempt_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		value    int
		svcErr   error
		wantCode int
	}{
		{"not found", 1, mysterysvc.ErrNotFound, http.StatusNotFound},
		{"invalid vote", 42, mysterysvc.ErrInvalidVote, http.StatusBadRequest},
		{"blocked", 1, block.ErrUserBlocked, http.StatusForbidden},
		{"internal", 1, errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			attemptID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ms.EXPECT().VoteAttempt(mock.Anything, attemptID, userID, tc.value).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mystery-attempts/"+attemptID.String()+"/vote").
				WithCookie("valid-cookie").
				WithJSONBody(dto.VoteRequest{Value: tc.value}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestMarkSolved_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries/"+uuid.NewString()+"/solve", map[string]string{"attempt_id": uuid.NewString()})
}

func TestMarkSolved_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/solve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"attempt_id": uuid.NewString()}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestMarkSolved_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/solve").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestMarkSolved_MissingAttemptID_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/solve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "attempt_id is required")
}

func TestMarkSolved_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	attemptID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().MarkSolved(mock.Anything, mysteryID, userID, attemptID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/solve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"attempt_id": attemptID.String()}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestMarkSolved_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			attemptID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ms.EXPECT().MarkSolved(mock.Anything, mysteryID, userID, attemptID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/solve").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]string{"attempt_id": attemptID.String()}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestAddClue_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries/"+uuid.NewString()+"/clues", dto.CreateClueRequest{Body: "b"})
}

func TestAddClue_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/clues").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateClueRequest{Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestAddClue_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/clues").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAddClue_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateClueRequest{Body: "clue", TruthType: "red"}
	ms.EXPECT().AddClue(mock.Anything, mysteryID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/clues").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestAddClue_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty body", mysterysvc.ErrEmptyBody, http.StatusBadRequest},
		{"not found", mysterysvc.ErrNotFound, http.StatusForbidden},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateClueRequest{Body: "x"}
			ms.EXPECT().AddClue(mock.Anything, mysteryID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/clues").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestCreateMysteryComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries/"+uuid.NewString()+"/comments", dto.CreateCommentRequest{Body: "hi"})
}

func TestCreateMysteryComment_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateMysteryComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateMysteryComment_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "hello"}
	ms.EXPECT().CreateComment(mock.Anything, mysteryID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateMysteryComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty body", mysterysvc.ErrEmptyBody, http.StatusBadRequest},
		{"not solved", mysterysvc.ErrNotSolved, http.StatusForbidden},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateCommentRequest{Body: "x"}
			ms.EXPECT().CreateComment(mock.Anything, mysteryID, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateMysteryComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "PUT", "/mystery-comments/"+uuid.NewString(), dto.UpdateCommentRequest{Body: "x"})
}

func TestUpdateMysteryComment_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/mystery-comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateMysteryComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("PUT", "/mystery-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateMysteryComment_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateCommentRequest{Body: "new"}
	ms.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/mystery-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateMysteryComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		svcErr   error
		wantCode int
	}{
		{"empty body", "", mysterysvc.ErrEmptyBody, http.StatusBadRequest},
		{"internal", "x", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateCommentRequest{Body: tc.body}
			ms.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/mystery-comments/"+commentID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteMysteryComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "DELETE", "/mystery-comments/"+uuid.NewString(), nil)
}

func TestDeleteMysteryComment_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/mystery-comments/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteMysteryComment_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/mystery-comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteMysteryComment_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/mystery-comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestLikeMysteryComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mystery-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeMysteryComment_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mystery-comments/not-a-uuid/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeMysteryComment_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/mystery-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeMysteryComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ms.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/mystery-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			if tc.wantBody != "" {
				assert.Contains(t, string(body), tc.wantBody)
			}
		})
	}
}

func TestUnlikeMysteryComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "DELETE", "/mystery-comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeMysteryComment_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/mystery-comments/not-a-uuid/like").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeMysteryComment_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/mystery-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeMysteryComment_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/mystery-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestUploadMysteryCommentMedia_NoCookie_Unauthorized(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)

	// when
	status, _ := h.NewRequest("POST", "/mystery-comments/"+uuid.NewString()+"/media").Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestUploadMysteryCommentMedia_InvalidSession_Unauthorized(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectInvalidSession("bogus")

	// when
	status, _ := h.NewRequest("POST", "/mystery-comments/"+uuid.NewString()+"/media").WithCookie("bogus").Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestUploadMysteryCommentMedia_BannedUser_Forbidden(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectBannedUser("banned-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mystery-comments/"+uuid.NewString()+"/media").WithCookie("banned-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
}

func TestUploadMysteryCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mystery-comments/not-a-uuid/media").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadMysteryCommentMedia_NoFile_BadRequest(t *testing.T) {
	// given — skip happy-path multipart test; only cover auth/UUID/no-file branches.
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mystery-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestUploadMysteryAttachment_NoCookie_Unauthorized(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/attachments").Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestUploadMysteryAttachment_InvalidSession_Unauthorized(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectInvalidSession("bogus")

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/attachments").WithCookie("bogus").Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestUploadMysteryAttachment_BannedUser_Forbidden(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectBannedUser("banned-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/attachments").WithCookie("banned-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
}

func TestUploadMysteryAttachment_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/attachments").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadMysteryAttachment_NoFile_BadRequest(t *testing.T) {
	// given — skip happy-path multipart test; cover auth/UUID/no-file branches only.
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/attachments").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no file provided")
}

func TestDeleteMysteryAttachment_NoCookie_Unauthorized(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+uuid.NewString()+"/attachments/1").Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestDeleteMysteryAttachment_InvalidSession_Unauthorized(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectInvalidSession("bogus")

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+uuid.NewString()+"/attachments/1").WithCookie("bogus").Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestDeleteMysteryAttachment_BannedUser_Forbidden(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectBannedUser("banned-cookie", userID)

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+uuid.NewString()+"/attachments/1").WithCookie("banned-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
}

func TestDeleteMysteryAttachment_InvalidMysteryID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/not-a-uuid/attachments/1").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteMysteryAttachment_InvalidAttachmentID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/"+uuid.NewString()+"/attachments/not-an-int").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid attachment id")
}

func TestDeleteMysteryAttachment_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteAttachment(mock.Anything, int64(42), mysteryID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()+"/attachments/42").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteMysteryAttachment_NotFound(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteAttachment(mock.Anything, int64(1), mysteryID, userID).Return(mysterysvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()+"/attachments/1").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestDeleteMysteryAttachment_NotAuthor_Forbidden(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteAttachment(mock.Anything, int64(1), mysteryID, userID).Return(mysterysvc.ErrNotAuthor)

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()+"/attachments/1").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), mysterysvc.ErrNotAuthor.Error())
}

func TestDeleteMysteryAttachment_InternalError(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().DeleteAttachment(mock.Anything, int64(1), mysteryID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()+"/attachments/1").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestToggleMysteryPause_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries/"+uuid.NewString()+"/pause", map[string]bool{"paused": true})
}

func TestToggleMysteryPause_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/pause").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"paused": true}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestToggleMysteryPause_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/pause").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestToggleMysteryPause_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().SetPaused(mock.Anything, mysteryID, userID, true).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/pause").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"paused": true}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestToggleMysteryPause_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ms.EXPECT().SetPaused(mock.Anything, mysteryID, userID, true).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/pause").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]bool{"paused": true}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestToggleMysteryGmAway_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "POST", "/mysteries/"+uuid.NewString()+"/away", map[string]bool{"away": true})
}

func TestToggleMysteryGmAway_InvalidID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/mysteries/not-a-uuid/away").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"away": true}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestToggleMysteryGmAway_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/mysteries/"+uuid.NewString()+"/away").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestToggleMysteryGmAway_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ms.EXPECT().SetGmAway(mock.Anything, mysteryID, userID, true).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/away").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"away": true}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestToggleMysteryGmAway_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not found", mysterysvc.ErrNotFound, http.StatusNotFound},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ms.EXPECT().SetGmAway(mock.Anything, mysteryID, userID, true).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/mysteries/"+mysteryID.String()+"/away").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]bool{"away": true}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteMysteryClue_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "DELETE", "/mysteries/"+uuid.NewString()+"/clues/1", nil)
}

func TestDeleteMysteryClue_MissingPermission(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/"+uuid.NewString()+"/clues/1").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "insufficient permissions")
}

func TestDeleteMysteryClue_InvalidMysteryID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/not-a-uuid/clues/1").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteMysteryClue_InvalidClueID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)

	// when
	status, body := h.NewRequest("DELETE", "/mysteries/"+uuid.NewString()+"/clues/not-an-int").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid clue id")
}

func TestDeleteMysteryClue_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	ms.EXPECT().DeleteClue(mock.Anything, mysteryID, 7, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()+"/clues/7").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteMysteryClue_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"not found", mysterysvc.ErrNotFound, http.StatusForbidden},
		{"not author", mysterysvc.ErrNotAuthor, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
			ms.EXPECT().DeleteClue(mock.Anything, mysteryID, 1, userID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("DELETE", "/mysteries/"+mysteryID.String()+"/clues/1").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateMysteryClue_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newMysteryHarness, "PUT", "/mysteries/"+uuid.NewString()+"/clues/1", map[string]string{"body": "x"})
}

func TestUpdateMysteryClue_MissingPermission(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	status, body := h.NewRequest("PUT", "/mysteries/"+uuid.NewString()+"/clues/1").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"body": "x"}).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "insufficient permissions")
}

func TestUpdateMysteryClue_InvalidMysteryID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)

	// when
	status, body := h.NewRequest("PUT", "/mysteries/not-a-uuid/clues/1").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"body": "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateMysteryClue_InvalidClueID(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)

	// when
	status, body := h.NewRequest("PUT", "/mysteries/"+uuid.NewString()+"/clues/not-an-int").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"body": "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid clue id")
}

func TestUpdateMysteryClue_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newMysteryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)

	// when
	status, body := h.NewRequest("PUT", "/mysteries/"+uuid.NewString()+"/clues/1").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestUpdateMysteryClue_OK(t *testing.T) {
	// given
	h, ms := newMysteryHarness(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	ms.EXPECT().UpdateClue(mock.Anything, mysteryID, 5, userID, "updated body").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/mysteries/"+mysteryID.String()+"/clues/5").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"body": "updated body"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUpdateMysteryClue_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		svcErr   error
		wantCode int
	}{
		{"not found", "x", mysterysvc.ErrNotFound, http.StatusBadRequest},
		{"not author", "x", mysterysvc.ErrNotAuthor, http.StatusBadRequest},
		{"empty body", "", mysterysvc.ErrEmptyBody, http.StatusBadRequest},
		{"internal", "x", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ms := newMysteryHarness(t)
			userID := uuid.New()
			mysteryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.AuthzService.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
			ms.EXPECT().UpdateClue(mock.Anything, mysteryID, 1, userID, tc.body).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/mysteries/"+mysteryID.String()+"/clues/1").
				WithCookie("valid-cookie").
				WithJSONBody(map[string]string{"body": tc.body}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}
