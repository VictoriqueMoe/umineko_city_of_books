package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	theorysvc "umineko_city_of_books/internal/theory"
	"umineko_city_of_books/internal/theory/params"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTheoryHarness(t *testing.T) (*testutil.Harness, *theorysvc.MockService) {
	h := testutil.NewHarness(t)
	ts := theorysvc.NewMockService(t)

	s := &Service{
		TheoryService: ts,
		AuthSession:   h.SessionManager,
		AuthzService:  h.AuthzService,
	}
	for _, setup := range s.getAllTheoryRoutes() {
		setup(h.App)
	}
	return h, ts
}

func defaultListParams() params.ListParams {
	return params.NewListParams("new", 0, uuid.Nil, "", "umineko", 20, 0)
}

func TestListTheories_Anonymous_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	expected := &dto.TheoryListResponse{Total: 0, Limit: 20, Offset: 0}
	ts.EXPECT().ListTheories(mock.Anything, defaultListParams(), uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/theories").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.TheoryListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
	assert.Equal(t, expected.Limit, got.Limit)
}

func TestListTheories_Authenticated_PassesUserID(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().ListTheories(mock.Anything, defaultListParams(), userID).Return(&dto.TheoryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/theories").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListTheories_CustomQuery_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	authorID := uuid.New()
	expected := params.NewListParams("top", 3, authorID, "beatrice", "higurashi", 50, 10)
	ts.EXPECT().ListTheories(mock.Anything, expected, uuid.Nil).Return(&dto.TheoryListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/theories?sort=top&episode=3&author="+authorID.String()+"&search=beatrice&series=higurashi&limit=50&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListTheories_InvalidAuthorID_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)

	// when
	status, body := h.NewRequest("GET", "/theories?author=not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid author ID")
}

func TestListTheories_InternalError(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	ts.EXPECT().ListTheories(mock.Anything, defaultListParams(), uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/theories").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list theories")
}

func TestCreateTheory_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "POST", "/theories", dto.CreateTheoryRequest{Title: "t", Body: "b"})
}

func TestCreateTheory_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateTheoryRequest{Title: "My Theory", Body: "because", Episode: 1, Series: "umineko"}
	ts.EXPECT().CreateTheory(mock.Anything, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/theories").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateTheory_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/theories").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateTheory_MissingFields_BadRequest(t *testing.T) {
	cases := []struct {
		name string
		req  dto.CreateTheoryRequest
	}{
		{"empty title", dto.CreateTheoryRequest{Title: "", Body: "body"}},
		{"empty body", dto.CreateTheoryRequest{Title: "title", Body: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newTheoryHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)

			// when
			status, body := h.NewRequest("POST", "/theories").
				WithCookie("valid-cookie").
				WithJSONBody(tc.req).
				Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "title and body are required")
		})
	}
}

func TestCreateTheory_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"rate limited", theorysvc.ErrRateLimited, http.StatusTooManyRequests, "daily theory limit reached"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create theory"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ts := newTheoryHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateTheoryRequest{Title: "t", Body: "b", Series: "umineko"}
			ts.EXPECT().CreateTheory(mock.Anything, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/theories").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetTheory_Anonymous_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	theoryID := uuid.New()
	expected := &dto.TheoryDetailResponse{ID: theoryID, Title: "A Theory"}
	ts.EXPECT().GetTheoryDetail(mock.Anything, theoryID, uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/theories/"+theoryID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.TheoryDetailResponse](t, body)
	assert.Equal(t, expected.ID, got.ID)
	assert.Equal(t, expected.Title, got.Title)
}

func TestGetTheory_Authenticated_PassesUserID(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().GetTheoryDetail(mock.Anything, theoryID, userID).Return(&dto.TheoryDetailResponse{ID: theoryID}, nil)

	// when
	status, _ := h.NewRequest("GET", "/theories/"+theoryID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetTheory_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)

	// when
	status, body := h.NewRequest("GET", "/theories/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetTheory_NotFound(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	theoryID := uuid.New()
	ts.EXPECT().GetTheoryDetail(mock.Anything, theoryID, uuid.Nil).Return(nil, nil)

	// when
	status, body := h.NewRequest("GET", "/theories/"+theoryID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "theory not found")
}

func TestGetTheory_InternalError(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	theoryID := uuid.New()
	ts.EXPECT().GetTheoryDetail(mock.Anything, theoryID, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/theories/"+theoryID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get theory")
}

func TestUpdateTheory_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "PUT", "/theories/"+uuid.NewString(), dto.CreateTheoryRequest{Title: "x", Body: "y"})
}

func TestUpdateTheory_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/theories/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateTheoryRequest{Title: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateTheory_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/theories/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateTheory_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateTheoryRequest{Title: "Updated", Body: "body"}
	ts.EXPECT().UpdateTheory(mock.Anything, theoryID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/theories/"+theoryID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUpdateTheory_Forbidden(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateTheoryRequest{Title: "Updated", Body: "body"}
	ts.EXPECT().UpdateTheory(mock.Anything, theoryID, userID, req).Return(errors.New("not owner"))

	// when
	status, body := h.NewRequest("PUT", "/theories/"+theoryID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot update this theory")
}

func TestDeleteTheory_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "DELETE", "/theories/"+uuid.NewString(), nil)
}

func TestDeleteTheory_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/theories/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteTheory_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().DeleteTheory(mock.Anything, theoryID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/theories/"+theoryID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteTheory_Forbidden(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().DeleteTheory(mock.Anything, theoryID, userID).Return(errors.New("not owner"))

	// when
	status, body := h.NewRequest("DELETE", "/theories/"+theoryID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot delete this theory")
}

func TestVoteTheory_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "POST", "/theories/"+uuid.NewString()+"/vote", dto.VoteRequest{Value: 1})
}

func TestVoteTheory_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/theories/not-a-uuid/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestVoteTheory_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/theories/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestVoteTheory_InvalidValue_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/theories/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 42}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "value must be 1, -1, or 0")
}

func TestVoteTheory_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().VoteTheory(mock.Anything, userID, theoryID, 1).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/theories/"+theoryID.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestVoteTheory_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to vote"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ts := newTheoryHarness(t)
			userID := uuid.New()
			theoryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ts.EXPECT().VoteTheory(mock.Anything, userID, theoryID, 1).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/theories/"+theoryID.String()+"/vote").
				WithCookie("valid-cookie").
				WithJSONBody(dto.VoteRequest{Value: 1}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateResponse_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "POST", "/theories/"+uuid.NewString()+"/responses", dto.CreateResponseRequest{Side: "with_love", Body: "yes"})
}

func TestCreateResponse_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/theories/not-a-uuid/responses").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateResponseRequest{Side: "with_love", Body: "yes"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateResponse_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/theories/"+uuid.NewString()+"/responses").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateResponse_InvalidSide_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/theories/"+uuid.NewString()+"/responses").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateResponseRequest{Side: "maybe", Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "side must be 'with_love' or 'without_love'")
}

func TestCreateResponse_EmptyBody_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/theories/"+uuid.NewString()+"/responses").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateResponseRequest{Side: "with_love", Body: ""}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "body is required")
}

func TestCreateResponse_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	theoryID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateResponseRequest{Side: "with_love", Body: "I agree"}
	ts.EXPECT().CreateResponse(mock.Anything, theoryID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/theories/"+theoryID.String()+"/responses").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateResponse_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"own theory", theorysvc.ErrCannotRespondToOwnTheory, http.StatusForbidden, theorysvc.ErrCannotRespondToOwnTheory.Error()},
		{"rate limited", theorysvc.ErrRateLimited, http.StatusTooManyRequests, "daily response limit reached"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ts := newTheoryHarness(t)
			userID := uuid.New()
			theoryID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateResponseRequest{Side: "without_love", Body: "x"}
			ts.EXPECT().CreateResponse(mock.Anything, theoryID, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/theories/"+theoryID.String()+"/responses").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteResponse_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "DELETE", "/responses/"+uuid.NewString(), nil)
}

func TestDeleteResponse_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/responses/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteResponse_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	responseID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().DeleteResponse(mock.Anything, responseID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/responses/"+responseID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteResponse_Forbidden(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	responseID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().DeleteResponse(mock.Anything, responseID, userID).Return(errors.New("not owner"))

	// when
	status, body := h.NewRequest("DELETE", "/responses/"+responseID.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot delete this response")
}

func TestVoteResponse_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newTheoryHarness, "POST", "/responses/"+uuid.NewString()+"/vote", dto.VoteRequest{Value: 1})
}

func TestVoteResponse_InvalidID(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/responses/not-a-uuid/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestVoteResponse_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/responses/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestVoteResponse_InvalidValue_BadRequest(t *testing.T) {
	// given
	h, _ := newTheoryHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/responses/"+uuid.NewString()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 99}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "value must be 1, -1, or 0")
}

func TestVoteResponse_OK(t *testing.T) {
	// given
	h, ts := newTheoryHarness(t)
	userID := uuid.New()
	responseID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ts.EXPECT().VoteResponse(mock.Anything, userID, responseID, -1).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/responses/"+responseID.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: -1}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestVoteResponse_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to vote"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, ts := newTheoryHarness(t)
			userID := uuid.New()
			responseID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			ts.EXPECT().VoteResponse(mock.Anything, userID, responseID, 0).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/responses/"+responseID.String()+"/vote").
				WithCookie("valid-cookie").
				WithJSONBody(dto.VoteRequest{Value: 0}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
