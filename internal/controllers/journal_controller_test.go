package controllers

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	journalsvc "umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/journal/params"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newJournalHarness(t *testing.T) (*testutil.Harness, *journalsvc.MockService) {
	h := testutil.NewHarness(t)
	js := journalsvc.NewMockService(t)

	s := &Service{
		JournalService: js,
		AuthSession:    h.SessionManager,
		AuthzService:   h.AuthzService,
	}
	for _, setup := range s.getAllJournalRoutes() {
		setup(h.App)
	}
	return h, js
}

func defaultJournalListParams() params.ListParams {
	return params.NewListParams("new", "", uuid.Nil, "", false, 20, 0)
}

func TestListJournals_Anonymous_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	expected := &dto.JournalListResponse{Total: 0, Limit: 20, Offset: 0}
	js.EXPECT().ListJournals(mock.Anything, defaultJournalListParams(), uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/journals").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.JournalListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListJournals_Authenticated_PassesUserID(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().ListJournals(mock.Anything, defaultJournalListParams(), userID).Return(&dto.JournalListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/journals").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListJournals_CustomParams(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	authorID := uuid.New()
	p := params.NewListParams("top", "umineko", authorID, "truth", true, 50, 10)
	js.EXPECT().ListJournals(mock.Anything, p, uuid.Nil).Return(&dto.JournalListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/journals?sort=top&work=umineko&author="+authorID.String()+"&search=truth&include_archived=true&limit=50&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListJournals_InvalidAuthor_BadRequest(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)

	// when
	status, body := h.NewRequest("GET", "/journals?author=not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid author ID")
}

func TestListJournals_InternalError(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	js.EXPECT().ListJournals(mock.Anything, defaultJournalListParams(), uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/journals").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list journals")
}

func TestListUserJournals_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	target := uuid.New()
	js.EXPECT().ListJournalsByUser(mock.Anything, target, uuid.Nil, 20, 0).Return(&dto.JournalListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+target.String()+"/journals").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserJournals_CustomPaging(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	target := uuid.New()
	js.EXPECT().ListJournalsByUser(mock.Anything, target, uuid.Nil, 5, 10).Return(&dto.JournalListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+target.String()+"/journals?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserJournals_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/journals").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserJournals_InternalError(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	target := uuid.New()
	js.EXPECT().ListJournalsByUser(mock.Anything, target, uuid.Nil, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+target.String()+"/journals").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list user journals")
}

func TestListUserFollowedJournals_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	target := uuid.New()
	js.EXPECT().ListFollowedByUser(mock.Anything, target, uuid.Nil, 20, 0).Return(&dto.JournalListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+target.String()+"/journal-follows").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserFollowedJournals_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/journal-follows").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserFollowedJournals_InternalError(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	target := uuid.New()
	js.EXPECT().ListFollowedByUser(mock.Anything, target, uuid.Nil, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+target.String()+"/journal-follows").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list followed journals")
}

func TestCreateJournal_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "POST", "/journals", dto.CreateJournalRequest{Title: "t", Body: "b"})
}

func TestCreateJournal_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateJournalRequest{Title: "t", Body: "b", Work: "umineko"}
	js.EXPECT().CreateJournal(mock.Anything, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/journals").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateJournal_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/journals").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateJournal_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"rate limited", journalsvc.ErrRateLimited, http.StatusTooManyRequests, "daily journal limit reached"},
		{"empty body", journalsvc.ErrEmptyBody, http.StatusBadRequest, "title and body are required"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create journal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateJournalRequest{Title: "t", Body: "b"}
			js.EXPECT().CreateJournal(mock.Anything, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/journals").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetJournal_Anonymous_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	id := uuid.New()
	js.EXPECT().GetJournalDetail(mock.Anything, id, uuid.Nil).Return(&dto.JournalDetailResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/journals/"+id.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetJournal_Authenticated_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().GetJournalDetail(mock.Anything, id, userID).Return(&dto.JournalDetailResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/journals/"+id.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetJournal_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)

	// when
	status, body := h.NewRequest("GET", "/journals/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetJournal_NotFound(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	id := uuid.New()
	js.EXPECT().GetJournalDetail(mock.Anything, id, uuid.Nil).Return(nil, journalsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/journals/"+id.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "journal not found")
}

func TestGetJournal_InternalError(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	id := uuid.New()
	js.EXPECT().GetJournalDetail(mock.Anything, id, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/journals/"+id.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get journal")
}

func TestUpdateJournal_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "PUT", "/journals/"+uuid.NewString(), dto.CreateJournalRequest{Title: "t", Body: "b"})
}

func TestUpdateJournal_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/journals/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateJournalRequest{Title: "t", Body: "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateJournal_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/journals/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateJournal_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateJournalRequest{Title: "updated", Body: "body"}
	js.EXPECT().UpdateJournal(mock.Anything, id, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/journals/"+id.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateJournal_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"empty body", journalsvc.ErrEmptyBody, http.StatusBadRequest, "title and body are required"},
		{"forbidden", errors.New("not owner"), http.StatusForbidden, "cannot update this journal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			id := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateJournalRequest{Title: "t", Body: "b"}
			js.EXPECT().UpdateJournal(mock.Anything, id, userID, req).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("PUT", "/journals/"+id.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteJournal_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "DELETE", "/journals/"+uuid.NewString(), nil)
}

func TestDeleteJournal_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/journals/not-a-uuid").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteJournal_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().DeleteJournal(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/journals/"+id.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteJournal_Forbidden(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().DeleteJournal(mock.Anything, id, userID).Return(errors.New("not owner"))

	// when
	status, body := h.NewRequest("DELETE", "/journals/"+id.String()).
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot delete this journal")
}

func TestFollowJournal_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "POST", "/journals/"+uuid.NewString()+"/follow", nil)
}

func TestFollowJournal_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/journals/not-a-uuid/follow").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestFollowJournal_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().FollowJournal(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/journals/"+id.String()+"/follow").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestFollowJournal_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"cannot follow own", journalsvc.ErrCannotFollowOwn, http.StatusBadRequest, "cannot follow your own journal"},
		{"not found", journalsvc.ErrNotFound, http.StatusNotFound, "journal not found"},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to follow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			id := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			js.EXPECT().FollowJournal(mock.Anything, id, userID).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/journals/"+id.String()+"/follow").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnfollowJournal_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "DELETE", "/journals/"+uuid.NewString()+"/follow", nil)
}

func TestUnfollowJournal_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/journals/not-a-uuid/follow").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnfollowJournal_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().UnfollowJournal(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/journals/"+id.String()+"/follow").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnfollowJournal_InternalError(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().UnfollowJournal(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/journals/"+id.String()+"/follow").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unfollow")
}

func TestCreateJournalComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "POST", "/journals/"+uuid.NewString()+"/comments", dto.CreateCommentRequest{Body: "b"})
}

func TestCreateJournalComment_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/journals/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "b"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateJournalComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/journals/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateJournalComment_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	journalID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "hello"}
	js.EXPECT().CreateComment(mock.Anything, journalID, userID, (*uuid.UUID)(nil), "hello").Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/journals/"+journalID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateJournalComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"empty body", journalsvc.ErrEmptyBody, http.StatusBadRequest, "body is required"},
		{"archived", journalsvc.ErrArchived, http.StatusForbidden, "journal is archived"},
		{"not found", journalsvc.ErrNotFound, http.StatusNotFound, "journal not found"},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			journalID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateCommentRequest{Body: "hi"}
			js.EXPECT().CreateComment(mock.Anything, journalID, userID, (*uuid.UUID)(nil), "hi").
				Return(uuid.Nil, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/journals/"+journalID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateJournalComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "PUT", "/journal-comments/"+uuid.NewString(), dto.UpdateCommentRequest{Body: "b"})
}

func TestUpdateJournalComment_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/journal-comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateJournalComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/journal-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateJournalComment_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().UpdateComment(mock.Anything, id, userID, "updated").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/journal-comments/"+id.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "updated"}).Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateJournalComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"empty body", journalsvc.ErrEmptyBody, http.StatusBadRequest, "body is required"},
		{"forbidden", errors.New("not owner"), http.StatusForbidden, "cannot update this comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			id := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			js.EXPECT().UpdateComment(mock.Anything, id, userID, "x").Return(tc.svcErr)

			// when
			status, body := h.NewRequest("PUT", "/journal-comments/"+id.String()).
				WithCookie("valid-cookie").
				WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteJournalComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "DELETE", "/journal-comments/"+uuid.NewString(), nil)
}

func TestDeleteJournalComment_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/journal-comments/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteJournalComment_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().DeleteComment(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/journal-comments/"+id.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteJournalComment_Forbidden(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().DeleteComment(mock.Anything, id, userID).Return(errors.New("not owner"))

	// when
	status, body := h.NewRequest("DELETE", "/journal-comments/"+id.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot delete this comment")
}

func TestLikeJournalComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "POST", "/journal-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeJournalComment_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/journal-comments/not-a-uuid/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeJournalComment_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().LikeComment(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/journal-comments/"+id.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeJournalComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"not found", journalsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to like comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			id := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			js.EXPECT().LikeComment(mock.Anything, id, userID).Return(tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/journal-comments/"+id.String()+"/like").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlikeJournalComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "DELETE", "/journal-comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeJournalComment_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/journal-comments/not-a-uuid/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeJournalComment_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().UnlikeComment(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/journal-comments/"+id.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeJournalComment_InternalError(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	js.EXPECT().UnlikeComment(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/journal-comments/"+id.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unlike comment")
}

func buildMultipart(t *testing.T, fieldName, filename, contentType string, content []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(map[string][]string)
	hdr["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + filename + `"`}
	if contentType != "" {
		hdr["Content-Type"] = []string{contentType}
	}
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func TestUploadJournalCommentMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newJournalHarness, "POST", "/journal-comments/"+uuid.NewString()+"/media", nil)
}

func TestUploadJournalCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/journal-comments/not-a-uuid/media").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadJournalCommentMedia_NoFile_BadRequest(t *testing.T) {
	// given
	h, _ := newJournalHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/journal-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestUploadJournalCommentMedia_OK(t *testing.T) {
	// given
	h, js := newJournalHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	expected := &dto.PostMediaResponse{MediaType: "image"}
	js.EXPECT().UploadCommentMedia(mock.Anything, commentID, userID, "image/png", mock.AnythingOfType("int64"), mock.Anything).
		Return(expected, nil)

	body, ct := buildMultipart(t, "media", "pic.png", "image/png", []byte("payload"))
	raw, err := io.ReadAll(body)
	require.NoError(t, err)

	// when
	status, respBody := h.NewRequest("POST", "/journal-comments/"+commentID.String()+"/media").
		WithCookie("valid-cookie").
		WithRawBody(string(raw), ct).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[dto.PostMediaResponse](t, respBody)
	assert.Equal(t, expected.MediaType, got.MediaType)
}

func TestUploadJournalCommentMedia_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"not author", journalsvc.ErrNotAuthor, http.StatusForbidden, "not the comment author"},
		{"not found", journalsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
		{"too large", errors.New("file too large: max 5MB"), http.StatusBadRequest, "too large"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to upload media"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := newJournalHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			js.EXPECT().UploadCommentMedia(mock.Anything, commentID, userID, "image/png", mock.AnythingOfType("int64"), mock.Anything).
				Return(nil, tc.svcErr)

			body, ct := buildMultipart(t, "media", "pic.png", "image/png", []byte("payload"))
			raw, err := io.ReadAll(body)
			require.NoError(t, err)

			// when
			status, respBody := h.NewRequest("POST", "/journal-comments/"+commentID.String()+"/media").
				WithCookie("valid-cookie").
				WithRawBody(string(raw), ct).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(respBody), tc.wantBody)
		})
	}
}
