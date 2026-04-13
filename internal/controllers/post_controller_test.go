package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/follow"
	postsvc "umineko_city_of_books/internal/post"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type postDeps struct {
	post   *postsvc.MockService
	follow *follow.MockService
}

func newPostHarness(t *testing.T) (*testutil.Harness, postDeps) {
	h := testutil.NewHarness(t)
	deps := postDeps{
		post:   postsvc.NewMockService(t),
		follow: follow.NewMockService(t),
	}

	s := &Service{
		PostService:   deps.post,
		FollowService: deps.follow,
		AuthSession:   h.SessionManager,
		AuthzService:  h.AuthzService,
	}
	for _, setup := range s.getAllPostRoutes() {
		setup(h.App)
	}
	return h, deps
}

func postFactory(t *testing.T) (*testutil.Harness, postDeps) {
	return newPostHarness(t)
}

func TestListPostFeed_Anonymous_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	expected := &dto.PostListResponse{Total: 0, Limit: 20}
	deps.post.EXPECT().
		ListFeed(mock.Anything, "everyone", uuid.Nil, "general", "", "", 0, 20, 0, "").
		Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/posts").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.PostListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListPostFeed_CustomQuery_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	deps.post.EXPECT().
		ListFeed(mock.Anything, "following", uuid.Nil, "suggestions", "search term", "top", 42, 10, 5, "open").
		Return(&dto.PostListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/posts?tab=following&corner=suggestions&search=search+term&sort=top&seed=42&limit=10&offset=5&resolved=open").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListPostFeed_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().
		ListFeed(mock.Anything, "everyone", userID, "general", "", "", 0, 20, 0, "").
		Return(&dto.PostListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/posts").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListPostFeed_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	deps.post.EXPECT().
		ListFeed(mock.Anything, "everyone", uuid.Nil, "general", "", "", 0, 20, 0, "").
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/posts").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list posts")
}

func TestGetCornerCounts_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	deps.post.EXPECT().GetCornerCounts(mock.Anything).Return(map[string]int{"general": 3}, nil)

	// when
	status, body := h.NewRequest("GET", "/posts/corner-counts").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]int](t, body)
	assert.Equal(t, 3, got["general"])
}

func TestGetCornerCounts_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	deps.post.EXPECT().GetCornerCounts(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/posts/corner-counts").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get counts")
}

func TestCreatePost_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/posts", dto.CreatePostRequest{Body: "hi"})
}

func TestCreatePost_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreatePostRequest{Body: "hello"}
	deps.post.EXPECT().CreatePost(mock.Anything, userID, req).Return(postID, nil)

	// when
	status, body := h.NewRequest("POST", "/posts").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, postID.String(), got["id"])
}

func TestCreatePost_BadJSON(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestCreatePost_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"invalid share type", postsvc.ErrInvalidShareType, http.StatusBadRequest, "invalid shared content type"},
		{"rate limited", postsvc.ErrRateLimited, http.StatusTooManyRequests, "daily post limit"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to create post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreatePostRequest{Body: "hi"}
			deps.post.EXPECT().CreatePost(mock.Anything, userID, req).Return(uuid.Nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetPost_Anonymous_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	postID := uuid.New()
	deps.post.EXPECT().
		GetPost(mock.Anything, postID, uuid.Nil, mock.AnythingOfType("string")).
		Return(&dto.PostDetailResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/posts/"+postID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetPost_Authenticated_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().
		GetPost(mock.Anything, postID, userID, mock.AnythingOfType("string")).
		Return(&dto.PostDetailResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/posts/"+postID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetPost_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)

	// when
	status, body := h.NewRequest("GET", "/posts/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetPost_NotFound(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	postID := uuid.New()
	deps.post.EXPECT().
		GetPost(mock.Anything, postID, uuid.Nil, mock.AnythingOfType("string")).
		Return(nil, postsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/posts/"+postID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "post not found")
}

func TestGetPost_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	postID := uuid.New()
	deps.post.EXPECT().
		GetPost(mock.Anything, postID, uuid.Nil, mock.AnythingOfType("string")).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/posts/"+postID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get post")
}

func TestUpdatePost_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "PUT", "/posts/"+uuid.NewString(), dto.UpdatePostRequest{Body: "x"})
}

func TestUpdatePost_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdatePostRequest{Body: "new"}
	deps.post.EXPECT().UpdatePost(mock.Anything, postID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/posts/"+postID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdatePost_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/posts/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdatePostRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdatePost_BadJSON(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/posts/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdatePost_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to update post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			postID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdatePostRequest{Body: "x"}
			deps.post.EXPECT().UpdatePost(mock.Anything, postID, userID, req).Return(tc.err)

			// when
			status, body := h.NewRequest("PUT", "/posts/"+postID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeletePost_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/posts/"+uuid.NewString(), nil)
}

func TestDeletePost_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().DeletePost(mock.Anything, postID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/posts/"+postID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeletePost_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/posts/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeletePost_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().DeletePost(mock.Anything, postID, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/posts/"+postID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete post")
}

func TestUploadPostMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/posts/"+uuid.NewString()+"/media", nil)
}

func TestUploadPostMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts/not-a-uuid/media").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadPostMedia_NoFile(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestDeletePostMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/posts/"+uuid.NewString()+"/media/42", nil)
}

func TestDeletePostMedia_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().DeletePostMedia(mock.Anything, postID, int64(42), userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/posts/"+postID.String()+"/media/42").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeletePostMedia_InvalidPostID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/posts/not-a-uuid/media/42").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeletePostMedia_InvalidMediaID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/posts/"+uuid.NewString()+"/media/0").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid media id")
}

func TestDeletePostMedia_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().DeletePostMedia(mock.Anything, postID, int64(7), userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/posts/"+postID.String()+"/media/7").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete media")
}

func TestLikePost_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/posts/"+uuid.NewString()+"/like", nil)
}

func TestLikePost_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().LikePost(mock.Anything, userID, postID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/posts/"+postID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikePost_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikePost_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to like post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			postID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.post.EXPECT().LikePost(mock.Anything, userID, postID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/like").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlikePost_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/posts/"+uuid.NewString()+"/like", nil)
}

func TestUnlikePost_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().UnlikePost(mock.Anything, userID, postID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/posts/"+postID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikePost_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/posts/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikePost_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().UnlikePost(mock.Anything, userID, postID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/posts/"+postID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unlike post")
}

func TestCreateComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/posts/"+uuid.NewString()+"/comments",
		dto.CreateCommentRequest{Body: "hi"})
}

func TestCreateComment_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "hello"}
	deps.post.EXPECT().CreateComment(mock.Anything, postID, userID, req).Return(commentID, nil)

	// when
	status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, commentID.String(), got["id"])
}

func TestCreateComment_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateComment_BadJSON(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/posts/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to create comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			postID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateCommentRequest{Body: "hi"}
			deps.post.EXPECT().CreateComment(mock.Anything, postID, userID, req).Return(uuid.Nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "PUT", "/comments/"+uuid.NewString(),
		dto.UpdateCommentRequest{Body: "x"})
}

func TestUpdateComment_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateCommentRequest{Body: "new"}
	deps.post.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateComment_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateComment_BadJSON(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to update comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateCommentRequest{Body: "x"}
			deps.post.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(tc.err)

			// when
			status, body := h.NewRequest("PUT", "/comments/"+commentID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/comments/"+uuid.NewString(), nil)
}

func TestDeleteComment_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteComment_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/comments/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteComment_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete comment")
}

func TestUploadCommentMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/comments/"+uuid.NewString()+"/media", nil)
}

func TestUploadCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/comments/not-a-uuid/media").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadCommentMedia_NoFile(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestLikeComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeComment_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeComment_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/comments/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to like comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.post.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/comments/"+commentID.String()+"/like").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlikeComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeComment_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeComment_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/comments/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeComment_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unlike comment")
}

func TestListUserPosts_Anonymous_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.post.EXPECT().ListUserPosts(mock.Anything, userID, uuid.Nil, 20, 0).
		Return(&dto.PostListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+userID.String()+"/posts").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserPosts_CustomPaging(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.post.EXPECT().ListUserPosts(mock.Anything, userID, uuid.Nil, 5, 10).
		Return(&dto.PostListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+userID.String()+"/posts?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserPosts_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/posts").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserPosts_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.post.EXPECT().ListUserPosts(mock.Anything, userID, uuid.Nil, 20, 0).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/posts").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list user posts")
}

func TestFollowUser_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/users/"+uuid.NewString()+"/follow", nil)
}

func TestFollowUser_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.follow.EXPECT().Follow(mock.Anything, userID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/users/"+targetID.String()+"/follow").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestFollowUser_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/users/not-a-uuid/follow").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestFollowUser_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"cannot follow self", follow.ErrCannotFollowSelf, http.StatusBadRequest, "cannot follow yourself"},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to follow user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			targetID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.follow.EXPECT().Follow(mock.Anything, userID, targetID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/users/"+targetID.String()+"/follow").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnfollowUser_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/users/"+uuid.NewString()+"/follow", nil)
}

func TestUnfollowUser_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.follow.EXPECT().Unfollow(mock.Anything, userID, targetID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/users/"+targetID.String()+"/follow").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnfollowUser_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/users/not-a-uuid/follow").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnfollowUser_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	targetID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.follow.EXPECT().Unfollow(mock.Anything, userID, targetID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/users/"+targetID.String()+"/follow").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unfollow user")
}

func TestGetFollowStats_Anonymous_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.follow.EXPECT().GetFollowStats(mock.Anything, userID, uuid.Nil).
		Return(&dto.FollowStatsResponse{FollowerCount: 2}, nil)

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/follow-stats").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.FollowStatsResponse](t, body)
	assert.Equal(t, 2, got.FollowerCount)
}

func TestGetFollowStats_Authenticated_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	viewerID := uuid.New()
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", viewerID)
	deps.follow.EXPECT().GetFollowStats(mock.Anything, userID, viewerID).
		Return(&dto.FollowStatsResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+userID.String()+"/follow-stats").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetFollowStats_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/follow-stats").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetFollowStats_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.follow.EXPECT().GetFollowStats(mock.Anything, userID, uuid.Nil).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/follow-stats").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get follow stats")
}

func TestGetFollowers_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	users := []dto.UserResponse{{ID: uuid.New(), Username: "beato"}}
	deps.follow.EXPECT().GetFollowers(mock.Anything, userID, 50, 0).Return(users, 1, nil)

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/followers").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.EqualValues(t, 1, got["total"])
}

func TestGetFollowers_CustomPaging(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.follow.EXPECT().GetFollowers(mock.Anything, userID, 5, 10).
		Return([]dto.UserResponse{}, 0, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+userID.String()+"/followers?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetFollowers_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/followers").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetFollowers_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.follow.EXPECT().GetFollowers(mock.Anything, userID, 50, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/followers").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get followers")
}

func TestGetFollowing_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.follow.EXPECT().GetFollowing(mock.Anything, userID, 50, 0).
		Return([]dto.UserResponse{}, 0, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+userID.String()+"/following").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetFollowing_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/following").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetFollowing_InternalError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	deps.follow.EXPECT().GetFollowing(mock.Anything, userID, 50, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/following").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get following")
}

func TestVotePoll_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/posts/"+uuid.NewString()+"/poll/vote",
		dto.VotePollRequest{OptionID: 1})
}

func TestVotePoll_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	poll := &dto.PollResponse{ID: "p1"}
	deps.post.EXPECT().VotePoll(mock.Anything, postID, userID, 3).Return(poll, nil)

	// when
	status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/poll/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VotePollRequest{OptionID: 3}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.PollResponse](t, body)
	assert.Equal(t, "p1", got.ID)
}

func TestVotePoll_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts/not-a-uuid/poll/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VotePollRequest{OptionID: 1}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestVotePoll_BadJSON(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/posts/"+uuid.NewString()+"/poll/vote").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestVotePoll_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not found", postsvc.ErrNotFound, http.StatusNotFound, "poll not found"},
		{"expired", postsvc.ErrPollExpired, http.StatusGone, "expired"},
		{"already voted", postsvc.ErrAlreadyVoted, http.StatusConflict, "already voted"},
		{"invalid option", postsvc.ErrInvalidOption, http.StatusBadRequest, "invalid poll option"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to vote"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			postID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.post.EXPECT().VotePoll(mock.Anything, postID, userID, 1).Return(nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/poll/vote").
				WithCookie("valid-cookie").
				WithJSONBody(dto.VotePollRequest{OptionID: 1}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestResolveSuggestion_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "POST", "/posts/"+uuid.NewString()+"/resolve", nil)
}

func TestResolveSuggestion_OK_DefaultStatus(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, "done").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/posts/"+postID.String()+"/resolve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestResolveSuggestion_OK_CustomStatus(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, "wont_fix").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/posts/"+postID.String()+"/resolve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"status": "wont_fix"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestResolveSuggestion_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/posts/not-a-uuid/resolve").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestResolveSuggestion_ServiceError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, "done").
		Return(errors.New("not allowed"))

	// when
	status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/resolve").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{}).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "not allowed")
}

func TestUnresolveSuggestion_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, postFactory, "DELETE", "/posts/"+uuid.NewString()+"/resolve", nil)
}

func TestUnresolveSuggestion_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().UnresolveSuggestion(mock.Anything, postID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/posts/"+postID.String()+"/resolve").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUnresolveSuggestion_InvalidID(t *testing.T) {
	// given
	h, _ := newPostHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/posts/not-a-uuid/resolve").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnresolveSuggestion_ServiceError(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	userID := uuid.New()
	postID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.post.EXPECT().UnresolveSuggestion(mock.Anything, postID, userID).
		Return(errors.New("not allowed"))

	// when
	status, body := h.NewRequest("DELETE", "/posts/"+postID.String()+"/resolve").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "not allowed")
}

func TestGetShareCount_OK(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	deps.post.EXPECT().GetShareCount(mock.Anything, "abc", "art").Return(7, nil)

	// when
	status, body := h.NewRequest("GET", "/share-count/art/abc").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]int](t, body)
	assert.Equal(t, 7, got["share_count"])
}

func TestGetShareCount_ServiceError_ReturnsZero(t *testing.T) {
	// given
	h, deps := newPostHarness(t)
	deps.post.EXPECT().GetShareCount(mock.Anything, "abc", "post").Return(0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/share-count/post/abc").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]int](t, body)
	assert.Equal(t, 0, got["share_count"])
}
