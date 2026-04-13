package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	shipsvc "umineko_city_of_books/internal/ship"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newShipHarness(t *testing.T) (*testutil.Harness, *shipsvc.MockService) {
	h := testutil.NewHarness(t)
	ss := shipsvc.NewMockService(t)

	s := &Service{
		ShipService:  ss,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllShipRoutes() {
		setup(h.App)
	}
	return h, ss
}

func TestListShips_Anonymous_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	expected := &dto.ShipListResponse{Total: 0, Limit: 20, Offset: 0}
	ss.EXPECT().ListShips(mock.Anything, uuid.Nil, "new", false, "", "", 20, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/ships").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ShipListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListShips_CustomQuery_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	ss.EXPECT().ListShips(mock.Anything, uuid.Nil, "top", true, "umineko", "beato", 10, 5).
		Return(&dto.ShipListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/ships?sort=top&crackships=true&series=umineko&character=beato&limit=10&offset=5").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListShips_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().ListShips(mock.Anything, userID, "new", false, "", "", 20, 0).
		Return(&dto.ShipListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/ships").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListShips_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	ss.EXPECT().ListShips(mock.Anything, uuid.Nil, "new", false, "", "", 20, 0).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/ships").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list ships")
}

func TestGetShip_Anonymous_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	shipID := uuid.New()
	expected := &dto.ShipDetailResponse{ShipResponse: dto.ShipResponse{ID: shipID, Title: "BeatoXBattler"}}
	ss.EXPECT().GetShip(mock.Anything, shipID, uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/ships/"+shipID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ShipDetailResponse](t, body)
	assert.Equal(t, shipID, got.ID)
}

func TestGetShip_Authenticated_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	shipID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().GetShip(mock.Anything, shipID, userID).
		Return(&dto.ShipDetailResponse{ShipResponse: dto.ShipResponse{ID: shipID}}, nil)

	// when
	status, _ := h.NewRequest("GET", "/ships/"+shipID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetShip_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)

	// when
	status, body := h.NewRequest("GET", "/ships/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetShip_NotFound(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	shipID := uuid.New()
	ss.EXPECT().GetShip(mock.Anything, shipID, uuid.Nil).Return(nil, shipsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/ships/"+shipID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "ship not found")
}

func TestGetShip_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	shipID := uuid.New()
	ss.EXPECT().GetShip(mock.Anything, shipID, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/ships/"+shipID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get ship")
}

func TestCreateShip_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "POST", "/ships", dto.CreateShipRequest{Title: "x"})
}

func TestCreateShip_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/ships").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateShip_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateShipRequest{Title: "A Ship", Description: "desc"}
	ss.EXPECT().CreateShip(mock.Anything, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/ships").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateShip_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty title", shipsvc.ErrEmptyTitle, http.StatusBadRequest},
		{"too few characters", shipsvc.ErrTooFewCharacters, http.StatusBadRequest},
		{"duplicate characters", shipsvc.ErrDuplicateCharacters, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newShipHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateShipRequest{Title: "x"}
			ss.EXPECT().CreateShip(mock.Anything, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/ships").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateShip_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "PUT", "/ships/"+uuid.NewString(), dto.UpdateShipRequest{Title: "x"})
}

func TestUpdateShip_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/ships/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateShipRequest{Title: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateShip_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/ships/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateShip_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	shipID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateShipRequest{Title: "Updated"}
	ss.EXPECT().UpdateShip(mock.Anything, shipID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/ships/"+shipID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateShip_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty title", shipsvc.ErrEmptyTitle, http.StatusBadRequest},
		{"too few characters", shipsvc.ErrTooFewCharacters, http.StatusBadRequest},
		{"duplicate characters", shipsvc.ErrDuplicateCharacters, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newShipHarness(t)
			userID := uuid.New()
			shipID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateShipRequest{Title: "x"}
			ss.EXPECT().UpdateShip(mock.Anything, shipID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/ships/"+shipID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteShip_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "DELETE", "/ships/"+uuid.NewString(), nil)
}

func TestDeleteShip_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/ships/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteShip_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	shipID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().DeleteShip(mock.Anything, shipID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/ships/"+shipID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteShip_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	shipID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().DeleteShip(mock.Anything, shipID, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/ships/"+shipID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete ship")
}

func TestUploadShipImage_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "POST", "/ships/"+uuid.NewString()+"/image", nil)
}

func TestUploadShipImage_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ships/not-a-uuid/image").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadShipImage_NoFile_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ships/"+uuid.NewString()+"/image").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no image file provided")
}

func TestVoteShip_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "POST", "/ships/"+uuid.NewString()+"/vote", dto.VoteRequest{Value: 1})
}

func TestVoteShip_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ships/not-a-uuid/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestVoteShip_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/ships/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestVoteShip_InvalidValue_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ships/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 42}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "value must be 1, -1, or 0")
}

func TestVoteShip_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	shipID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().Vote(mock.Anything, userID, shipID, 1).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/ships/"+shipID.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestVoteShip_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newShipHarness(t)
			userID := uuid.New()
			shipID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ss.EXPECT().Vote(mock.Anything, userID, shipID, -1).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/ships/"+shipID.String()+"/vote").
				WithCookie("valid-cookie").
				WithJSONBody(dto.VoteRequest{Value: -1}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestCreateShipComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "POST", "/ships/"+uuid.NewString()+"/comments", dto.CreateCommentRequest{Body: "hi"})
}

func TestCreateShipComment_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ships/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateShipComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/ships/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateShipComment_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	shipID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "hi"}
	ss.EXPECT().CreateComment(mock.Anything, shipID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/ships/"+shipID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateShipComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"empty body", shipsvc.ErrEmptyBody, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newShipHarness(t)
			userID := uuid.New()
			shipID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateCommentRequest{Body: "hi"}
			ss.EXPECT().CreateComment(mock.Anything, shipID, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/ships/"+shipID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateShipComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "PUT", "/ship-comments/"+uuid.NewString(), dto.UpdateCommentRequest{Body: "x"})
}

func TestUpdateShipComment_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/ship-comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateShipComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/ship-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateShipComment_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateCommentRequest{Body: "edited"}
	ss.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/ship-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateShipComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty body", shipsvc.ErrEmptyBody, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newShipHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateCommentRequest{Body: "x"}
			ss.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/ship-comments/"+commentID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteShipComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "DELETE", "/ship-comments/"+uuid.NewString(), nil)
}

func TestDeleteShipComment_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/ship-comments/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteShipComment_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/ship-comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteShipComment_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/ship-comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete comment")
}

func TestLikeShipComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "POST", "/ship-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeShipComment_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ship-comments/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeShipComment_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/ship-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeShipComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ss := newShipHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ss.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/ship-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUnlikeShipComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "DELETE", "/ship-comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeShipComment_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/ship-comments/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeShipComment_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/ship-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeShipComment_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ss.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/ship-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unlike comment")
}

func TestUploadShipCommentMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newShipHarness, "POST", "/ship-comments/"+uuid.NewString()+"/media", nil)
}

func TestUploadShipCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ship-comments/not-a-uuid/media").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadShipCommentMedia_NoFile_BadRequest(t *testing.T) {
	// given
	h, _ := newShipHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/ship-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestListCharacters_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	expected := []dto.CharacterListEntry{{ID: "beato", Name: "Beatrice"}}
	ss.EXPECT().ListCharacters(mock.AnythingOfType("quotefinder.Series")).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/characters/umineko").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.CharacterListResponse](t, body)
	assert.Equal(t, "umineko", got.Series)
	require.Len(t, got.Characters, 1)
	assert.Equal(t, "beato", got.Characters[0].ID)
}

func TestListCharacters_InvalidSeries(t *testing.T) {
	// given
	h, _ := newShipHarness(t)

	// when
	status, body := h.NewRequest("GET", "/characters/notaseries").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "unsupported series")
}

func TestListCharacters_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	ss.EXPECT().ListCharacters(mock.AnythingOfType("quotefinder.Series")).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/characters/umineko").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list characters")
}

func TestListUserShips_OK(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	targetUserID := uuid.New()
	ss.EXPECT().ListShipsByUser(mock.Anything, targetUserID, uuid.Nil, 20, 0).Return(&dto.ShipListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/ships").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserShips_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	viewerID := uuid.New()
	targetUserID := uuid.New()
	h.ExpectValidSession("valid-cookie", viewerID)
	ss.EXPECT().ListShipsByUser(mock.Anything, targetUserID, viewerID, 20, 0).Return(&dto.ShipListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/ships").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserShips_CustomPaging(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	targetUserID := uuid.New()
	ss.EXPECT().ListShipsByUser(mock.Anything, targetUserID, uuid.Nil, 5, 10).Return(&dto.ShipListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/ships?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserShips_InvalidID(t *testing.T) {
	// given
	h, _ := newShipHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/ships").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserShips_InternalError(t *testing.T) {
	// given
	h, ss := newShipHarness(t)
	targetUserID := uuid.New()
	ss.EXPECT().ListShipsByUser(mock.Anything, targetUserID, uuid.Nil, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/ships").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list user ships")
}
