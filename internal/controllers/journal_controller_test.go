package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	journalsvc "umineko_city_of_books/internal/journal"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	journalCtlNotOwned = errors.Join(errors.New("not found or not owned"), dao.ErrNotFound)
)

type (
	journalCtlOutcome struct {
		name     string
		newID    uuid.UUID
		svcErr   error
		wantCode int
		wantBody string
	}
)

func journalCtlHarness(t *testing.T) (*testutil.Harness, *journalsvc.MockService) {
	t.Helper()

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

func journalCtlSignedIn(t *testing.T) (*testutil.Harness, *journalsvc.MockService, uuid.UUID) {
	t.Helper()

	h, js := journalCtlHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	return h, js, userID
}

func TestJournalRoutes_AuthFailures(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "create journal", method: "POST", path: "/journals", body: dto.CreateJournalRequest{Title: "t"}},
		{name: "update journal", method: "PUT", path: "/journals/" + uuid.NewString(), body: dto.CreateJournalRequest{Title: "t"}},
		{name: "delete journal", method: "DELETE", path: "/journals/" + uuid.NewString()},
		{name: "follow journal", method: "POST", path: "/journals/" + uuid.NewString() + "/follow"},
		{name: "unfollow journal", method: "DELETE", path: "/journals/" + uuid.NewString() + "/follow"},
		{name: "create comment", method: "POST", path: "/journals/" + uuid.NewString() + "/comments", body: dto.CreateCommentRequest{Body: "b"}},
		{name: "update comment", method: "PUT", path: "/journal-comments/" + uuid.NewString(), body: dto.UpdateCommentRequest{Body: "b"}},
		{name: "delete comment", method: "DELETE", path: "/journal-comments/" + uuid.NewString()},
		{name: "like comment", method: "POST", path: "/journal-comments/" + uuid.NewString() + "/like"},
		{name: "unlike comment", method: "DELETE", path: "/journal-comments/" + uuid.NewString() + "/like"},
		{name: "upload comment media", method: "POST", path: "/journal-comments/" + uuid.NewString() + "/media"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, journalCtlHarness, tc.method, tc.path, tc.body)
		})
	}
}

func TestJournalRoutes_MalformedRequests(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		path     string
		signedIn bool
		jsonBody any
		rawBody  string
		wantBody string
	}{
		{name: "list journals by a non-uuid author", method: "GET", path: "/journals?author=not-a-uuid", wantBody: "invalid author ID"},
		{name: "list user journals for a non-uuid user", method: "GET", path: "/users/not-a-uuid/journals", wantBody: "invalid id"},
		{name: "list followed journals for a non-uuid user", method: "GET", path: "/users/not-a-uuid/journal-follows", wantBody: "invalid id"},
		{name: "get a non-uuid journal", method: "GET", path: "/journals/not-a-uuid", wantBody: "invalid id"},
		{name: "update a non-uuid journal", method: "PUT", path: "/journals/not-a-uuid", signedIn: true, jsonBody: dto.CreateJournalRequest{Title: "t"}, wantBody: "invalid id"},
		{name: "delete a non-uuid journal", method: "DELETE", path: "/journals/not-a-uuid", signedIn: true, wantBody: "invalid id"},
		{name: "follow a non-uuid journal", method: "POST", path: "/journals/not-a-uuid/follow", signedIn: true, wantBody: "invalid id"},
		{name: "unfollow a non-uuid journal", method: "DELETE", path: "/journals/not-a-uuid/follow", signedIn: true, wantBody: "invalid id"},
		{name: "comment on a non-uuid journal", method: "POST", path: "/journals/not-a-uuid/comments", signedIn: true, jsonBody: dto.CreateCommentRequest{Body: "b"}, wantBody: "invalid id"},
		{name: "update a non-uuid comment", method: "PUT", path: "/journal-comments/not-a-uuid", signedIn: true, jsonBody: dto.UpdateCommentRequest{Body: "x"}, wantBody: "invalid id"},
		{name: "delete a non-uuid comment", method: "DELETE", path: "/journal-comments/not-a-uuid", signedIn: true, wantBody: "invalid id"},
		{name: "like a non-uuid comment", method: "POST", path: "/journal-comments/not-a-uuid/like", signedIn: true, wantBody: "invalid id"},
		{name: "unlike a non-uuid comment", method: "DELETE", path: "/journal-comments/not-a-uuid/like", signedIn: true, wantBody: "invalid id"},
		{name: "upload media to a non-uuid comment", method: "POST", path: "/journal-comments/not-a-uuid/media", signedIn: true, wantBody: "invalid id"},
		{name: "create a journal from malformed json", method: "POST", path: "/journals", signedIn: true, rawBody: "not json", wantBody: "invalid request body"},
		{name: "update a journal from malformed json", method: "PUT", path: "/journals/" + uuid.NewString(), signedIn: true, rawBody: "not json", wantBody: "invalid request body"},
		{name: "create a comment from malformed json", method: "POST", path: "/journals/" + uuid.NewString() + "/comments", signedIn: true, rawBody: "not json", wantBody: "invalid request body"},
		{name: "update a comment from malformed json", method: "PUT", path: "/journal-comments/" + uuid.NewString(), signedIn: true, rawBody: "not json", wantBody: "invalid request body"},
		{name: "upload comment media without a file", method: "POST", path: "/journal-comments/" + uuid.NewString() + "/media", signedIn: true, wantBody: "no media file provided"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := journalCtlHarness(t)

			req := h.NewRequest(tc.method, tc.path)
			if tc.signedIn {
				h.ExpectValidSession("valid-cookie", uuid.New())
				req = req.WithCookie("valid-cookie")
			}
			if tc.jsonBody != nil {
				req = req.WithJSONBody(tc.jsonBody)
			}
			if tc.rawBody != "" {
				req = req.WithRawBody(tc.rawBody, "application/json")
			}

			// when
			status, body := req.Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestJournalRoutes_IDActions(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		path     string
		expect   func(js *journalsvc.MockService, id, userID uuid.UUID, err error)
		outcomes []journalCtlOutcome
	}{
		{
			name:   "delete journal",
			method: "DELETE",
			path:   "/journals/:id",
			expect: func(js *journalsvc.MockService, id, userID uuid.UUID, err error) {
				js.EXPECT().DeleteJournal(mock.Anything, id, userID).Return(err)
			},
			outcomes: []journalCtlOutcome{
				{name: "deleted", wantCode: http.StatusNoContent},
				{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "journal not found"},
				{name: "not owned", svcErr: journalCtlNotOwned, wantCode: http.StatusForbidden, wantBody: "cannot delete this journal"},
				{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to delete journal"},
			},
		},
		{
			name:   "follow journal",
			method: "POST",
			path:   "/journals/:id/follow",
			expect: func(js *journalsvc.MockService, id, userID uuid.UUID, err error) {
				js.EXPECT().FollowJournal(mock.Anything, id, userID).Return(err)
			},
			outcomes: []journalCtlOutcome{
				{name: "followed", wantCode: http.StatusNoContent},
				{name: "cannot follow own", svcErr: journalsvc.ErrCannotFollowOwn, wantCode: http.StatusBadRequest, wantBody: "cannot follow your own journal"},
				{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "journal not found"},
				{name: "blocked", svcErr: block.ErrUserBlocked, wantCode: http.StatusForbidden, wantBody: "user is blocked"},
				{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to follow"},
			},
		},
		{
			name:   "unfollow journal",
			method: "DELETE",
			path:   "/journals/:id/follow",
			expect: func(js *journalsvc.MockService, id, userID uuid.UUID, err error) {
				js.EXPECT().UnfollowJournal(mock.Anything, id, userID).Return(err)
			},
			outcomes: []journalCtlOutcome{
				{name: "unfollowed", wantCode: http.StatusNoContent},
				{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to unfollow"},
			},
		},
		{
			name:   "delete comment",
			method: "DELETE",
			path:   "/journal-comments/:id",
			expect: func(js *journalsvc.MockService, id, userID uuid.UUID, err error) {
				js.EXPECT().DeleteComment(mock.Anything, id, userID).Return(err)
			},
			outcomes: []journalCtlOutcome{
				{name: "deleted", wantCode: http.StatusNoContent},
				{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "comment not found"},
				{name: "not owned", svcErr: journalCtlNotOwned, wantCode: http.StatusForbidden, wantBody: "cannot delete this comment"},
				{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to delete comment"},
			},
		},
		{
			name:   "like comment",
			method: "POST",
			path:   "/journal-comments/:id/like",
			expect: func(js *journalsvc.MockService, id, userID uuid.UUID, err error) {
				js.EXPECT().LikeComment(mock.Anything, id, userID).Return(err)
			},
			outcomes: []journalCtlOutcome{
				{name: "liked", wantCode: http.StatusNoContent},
				{name: "blocked", svcErr: block.ErrUserBlocked, wantCode: http.StatusForbidden, wantBody: "user is blocked"},
				{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "comment not found"},
				{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to like comment"},
			},
		},
		{
			name:   "unlike comment",
			method: "DELETE",
			path:   "/journal-comments/:id/like",
			expect: func(js *journalsvc.MockService, id, userID uuid.UUID, err error) {
				js.EXPECT().UnlikeComment(mock.Anything, id, userID).Return(err)
			},
			outcomes: []journalCtlOutcome{
				{name: "unliked", wantCode: http.StatusNoContent},
				{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to unlike comment"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, outcome := range tc.outcomes {
				t.Run(outcome.name, func(t *testing.T) {
					// given
					h, js, userID := journalCtlSignedIn(t)
					id := uuid.New()
					tc.expect(js, id, userID, outcome.svcErr)

					// when
					status, body := h.NewRequest(tc.method, strings.ReplaceAll(tc.path, ":id", id.String())).WithCookie("valid-cookie").Do()

					// then
					require.Equal(t, outcome.wantCode, status)
					assert.Contains(t, string(body), outcome.wantBody)
				})
			}
		})
	}
}

func TestListJournals(t *testing.T) {
	authorID := uuid.New()
	defaults := params.NewListParams("new", "", uuid.Nil, "", false, 20, 0)
	listed := &dto.JournalListResponse{Total: 7, Limit: 20, Offset: 0}
	cases := []struct {
		name     string
		signedIn bool
		query    string
		want     params.ListParams
		resp     *dto.JournalListResponse
		svcErr   error
		wantCode int
		wantBody string
	}{
		{name: "anonymous viewers get the default listing", want: defaults, resp: listed, wantCode: http.StatusOK, wantBody: `"total":7`},
		{name: "signed-in viewers are passed through as the viewer", signedIn: true, want: defaults, resp: listed, wantCode: http.StatusOK, wantBody: `"total":7`},
		{
			name:     "every query parameter reaches the service",
			query:    "?sort=top&work=umineko&author=" + authorID.String() + "&search=truth&include_archived=true&limit=50&offset=10",
			want:     params.NewListParams("top", "umineko", authorID, "truth", true, 50, 10),
			resp:     listed,
			wantCode: http.StatusOK,
			wantBody: `"total":7`,
		},
		{name: "internal", want: defaults, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list journals"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := journalCtlHarness(t)

			viewerID := uuid.Nil
			req := h.NewRequest("GET", "/journals"+tc.query)
			if tc.signedIn {
				viewerID = uuid.New()
				h.ExpectValidSession("valid-cookie", viewerID)
				req = req.WithCookie("valid-cookie")
			}

			js.EXPECT().ListJournals(mock.Anything, tc.want, viewerID).Return(tc.resp, tc.svcErr)

			// when
			status, body := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListUserJournals(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
		resp       *dto.JournalListResponse
		svcErr     error
		wantCode   int
		wantBody   string
	}{
		{name: "paging defaults to 20 from 0", wantLimit: 20, wantOffset: 0, resp: &dto.JournalListResponse{}, wantCode: http.StatusOK},
		{name: "custom paging reaches the service", query: "?limit=5&offset=10", wantLimit: 5, wantOffset: 10, resp: &dto.JournalListResponse{}, wantCode: http.StatusOK},
		{name: "internal", wantLimit: 20, wantOffset: 0, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list user journals"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := journalCtlHarness(t)
			target := uuid.New()
			js.EXPECT().ListJournalsByUser(mock.Anything, target, uuid.Nil, tc.wantLimit, tc.wantOffset).Return(tc.resp, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/users/"+target.String()+"/journals"+tc.query).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListUserFollowedJournals(t *testing.T) {
	cases := []struct {
		name     string
		resp     *dto.JournalListResponse
		svcErr   error
		wantCode int
		wantBody string
	}{
		{name: "paging defaults to 20 from 0", resp: &dto.JournalListResponse{}, wantCode: http.StatusOK},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list followed journals"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := journalCtlHarness(t)
			target := uuid.New()
			js.EXPECT().ListFollowedByUser(mock.Anything, target, uuid.Nil, bounds.NewPage(20, 0)).Return(tc.resp, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/users/"+target.String()+"/journal-follows").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateJournal(t *testing.T) {
	newID := uuid.New()
	cases := []journalCtlOutcome{
		{name: "created", newID: newID, wantCode: http.StatusCreated, wantBody: `{"id":"` + newID.String() + `"}`},
		{name: "rate limited", svcErr: journalsvc.ErrRateLimited, wantCode: http.StatusTooManyRequests, wantBody: "daily journal limit reached"},
		{name: "empty title", svcErr: journalsvc.ErrEmptyTitle, wantCode: http.StatusBadRequest, wantBody: "title is required"},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to create journal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js, userID := journalCtlSignedIn(t)
			req := dto.CreateJournalRequest{Title: "t", Work: "umineko"}
			js.EXPECT().CreateJournal(mock.Anything, userID, req).Return(tc.newID, tc.svcErr)

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

func TestGetJournal(t *testing.T) {
	cases := []struct {
		name     string
		signedIn bool
		detail   *dto.JournalDetailResponse
		svcErr   error
		wantCode int
		wantBody string
	}{
		{name: "anonymous viewers get the journal", detail: &dto.JournalDetailResponse{}, wantCode: http.StatusOK},
		{name: "signed-in viewers are passed through as the viewer", signedIn: true, detail: &dto.JournalDetailResponse{}, wantCode: http.StatusOK},
		{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "journal not found"},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to get journal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js := journalCtlHarness(t)
			id := uuid.New()

			viewerID := uuid.Nil
			req := h.NewRequest("GET", "/journals/"+id.String())
			if tc.signedIn {
				viewerID = uuid.New()
				h.ExpectValidSession("valid-cookie", viewerID)
				req = req.WithCookie("valid-cookie")
			}

			js.EXPECT().GetJournalDetail(mock.Anything, id, viewerID).Return(tc.detail, tc.svcErr)

			// when
			status, body := req.Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateJournal(t *testing.T) {
	cases := []journalCtlOutcome{
		{name: "updated", wantCode: http.StatusNoContent},
		{name: "empty title", svcErr: journalsvc.ErrEmptyTitle, wantCode: http.StatusBadRequest, wantBody: "title is required"},
		{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "journal not found"},
		{name: "not owned", svcErr: journalCtlNotOwned, wantCode: http.StatusForbidden, wantBody: "cannot update this journal"},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to update journal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js, userID := journalCtlSignedIn(t)
			id := uuid.New()
			req := dto.CreateJournalRequest{Title: "updated"}
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

func TestCreateJournalComment(t *testing.T) {
	newID := uuid.New()
	cases := []journalCtlOutcome{
		{name: "created", newID: newID, wantCode: http.StatusCreated, wantBody: `{"id":"` + newID.String() + `"}`},
		{name: "empty body", svcErr: journalsvc.ErrEmptyBody, wantCode: http.StatusBadRequest, wantBody: "body is required"},
		{name: "archived", svcErr: journalsvc.ErrArchived, wantCode: http.StatusForbidden, wantBody: "journal is archived"},
		{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "journal not found"},
		{name: "blocked", svcErr: block.ErrUserBlocked, wantCode: http.StatusForbidden, wantBody: "user is blocked"},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to create comment"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js, userID := journalCtlSignedIn(t)
			journalID := uuid.New()
			js.EXPECT().CreateComment(mock.Anything, journalID, userID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), "hello").
				Return(tc.newID, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/journals/"+journalID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(dto.CreateCommentRequest{Body: "hello"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateJournalComment(t *testing.T) {
	cases := []journalCtlOutcome{
		{name: "updated", wantCode: http.StatusNoContent},
		{name: "empty body", svcErr: journalsvc.ErrEmptyBody, wantCode: http.StatusBadRequest, wantBody: "body is required"},
		{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "comment not found"},
		{name: "not owned", svcErr: journalCtlNotOwned, wantCode: http.StatusForbidden, wantBody: "cannot update this comment"},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to update comment"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js, userID := journalCtlSignedIn(t)
			id := uuid.New()
			js.EXPECT().UpdateComment(mock.Anything, id, userID, "updated").Return(tc.svcErr)

			// when
			status, body := h.NewRequest("PUT", "/journal-comments/"+id.String()).
				WithCookie("valid-cookie").
				WithJSONBody(dto.UpdateCommentRequest{Body: "updated"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUploadJournalCommentMedia(t *testing.T) {
	cases := []struct {
		name     string
		media    *dto.PostMediaResponse
		svcErr   error
		wantCode int
		wantBody string
	}{
		{name: "uploaded", media: &dto.PostMediaResponse{MediaType: "image"}, wantCode: http.StatusCreated, wantBody: `"media_type":"image"`},
		{name: "not author", svcErr: journalsvc.ErrNotAuthor, wantCode: http.StatusForbidden, wantBody: "not the comment author"},
		{name: "not found", svcErr: journalsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "comment not found"},
		{name: "too large", svcErr: fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), wantCode: http.StatusBadRequest, wantBody: "exceeds maximum 5MB"},
		{name: "a server error mentioning a size is still a server error", svcErr: errors.New("pq: value too large for column"), wantCode: http.StatusInternalServerError, wantBody: "failed to upload media"},
		{name: "internal", svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to upload media"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, js, userID := journalCtlSignedIn(t)
			commentID := uuid.New()
			form, contentType := testutil.MediaForm(t, "media", nil)
			js.EXPECT().UploadCommentMedia(mock.Anything, commentID, userID, "image/png", mock.Anything, mock.AnythingOfType("int64"), mock.Anything, mock.Anything).
				Return(tc.media, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/journal-comments/"+commentID.String()+"/media").
				WithCookie("valid-cookie").
				WithRawBody(form, contentType).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
