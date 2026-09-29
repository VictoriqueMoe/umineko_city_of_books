package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/follow"
	postsvc "umineko_city_of_books/internal/post"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	postDeps struct {
		post   *postsvc.MockService
		follow *follow.MockService
	}

	postCtlOutcome struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}

	postCtlViewerOutcome struct {
		name     string
		authed   bool
		err      error
		wantCode int
		wantBody string
	}

	postCtlPagedOutcome struct {
		name     string
		query    string
		page     bounds.Page
		err      error
		wantCode int
		wantBody string
	}

	postCtlFeedQuery struct {
		tab      string
		corner   string
		search   string
		sort     string
		seed     int
		page     bounds.Page
		resolved string
	}
)

const (
	postCtlCookie    = "valid-cookie"
	postCtlJSON      = "application/json"
	postCtlMultipart = "multipart/form-data; boundary=xxx"
)

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

func postCtlViewer(h *testutil.Harness, authed bool) (uuid.UUID, string) {
	if !authed {
		return uuid.Nil, ""
	}

	viewerID := uuid.New()
	h.ExpectValidSession(postCtlCookie, viewerID)

	return viewerID, postCtlCookie
}

func postCtlSignedIn(t *testing.T) (*testutil.Harness, postDeps, uuid.UUID) {
	h, deps := newPostHarness(t)
	userID, _ := postCtlViewer(h, true)

	return h, deps, userID
}

func TestPostController_AuthFailures(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"create post", "POST", "/posts", dto.CreatePostRequest{Body: "hi"}},
		{"update post", "PUT", "/posts/" + uuid.NewString(), dto.UpdatePostRequest{Body: "x"}},
		{"delete post", "DELETE", "/posts/" + uuid.NewString(), nil},
		{"upload post media", "POST", "/posts/" + uuid.NewString() + "/media", nil},
		{"delete post media", "DELETE", "/posts/" + uuid.NewString() + "/media/42", nil},
		{"like post", "POST", "/posts/" + uuid.NewString() + "/like", nil},
		{"unlike post", "DELETE", "/posts/" + uuid.NewString() + "/like", nil},
		{"create comment", "POST", "/posts/" + uuid.NewString() + "/comments", dto.CreateCommentRequest{Body: "hi"}},
		{"update comment", "PUT", "/comments/" + uuid.NewString(), dto.UpdateCommentRequest{Body: "x"}},
		{"delete comment", "DELETE", "/comments/" + uuid.NewString(), nil},
		{"upload comment media", "POST", "/comments/" + uuid.NewString() + "/media", nil},
		{"like comment", "POST", "/comments/" + uuid.NewString() + "/like", nil},
		{"unlike comment", "DELETE", "/comments/" + uuid.NewString() + "/like", nil},
		{"follow user", "POST", "/users/" + uuid.NewString() + "/follow", nil},
		{"unfollow user", "DELETE", "/users/" + uuid.NewString() + "/follow", nil},
		{"vote poll", "POST", "/posts/" + uuid.NewString() + "/poll/vote", dto.VotePollRequest{OptionID: 1}},
		{"resolve suggestion", "POST", "/posts/" + uuid.NewString() + "/resolve", nil},
		{"unresolve suggestion", "DELETE", "/posts/" + uuid.NewString() + "/resolve", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newPostHarness, tc.method, tc.path, tc.body)
		})
	}
}

func TestPostController_RejectsMalformedRequests(t *testing.T) {
	id := uuid.NewString()
	cases := []struct {
		name        string
		method      string
		path        string
		anonymous   bool
		jsonBody    any
		rawBody     string
		contentType string
		wantBody    string
	}{
		{name: "create post bad json", method: "POST", path: "/posts", rawBody: "not json", contentType: postCtlJSON, wantBody: "invalid request body"},
		{name: "get post invalid id", method: "GET", path: "/posts/not-a-uuid", anonymous: true, wantBody: "invalid id"},
		{name: "update post invalid id", method: "PUT", path: "/posts/not-a-uuid", jsonBody: dto.UpdatePostRequest{Body: "x"}, wantBody: "invalid id"},
		{name: "update post bad json", method: "PUT", path: "/posts/" + id, rawBody: "not json", contentType: postCtlJSON, wantBody: "invalid request body"},
		{name: "delete post invalid id", method: "DELETE", path: "/posts/not-a-uuid", wantBody: "invalid id"},
		{name: "upload post media invalid id", method: "POST", path: "/posts/not-a-uuid/media", wantBody: "invalid id"},
		{name: "upload post media no file", method: "POST", path: "/posts/" + id + "/media", contentType: postCtlMultipart, wantBody: "no media file provided"},
		{name: "delete post media invalid post id", method: "DELETE", path: "/posts/not-a-uuid/media/42", wantBody: "invalid id"},
		{name: "delete post media invalid media id", method: "DELETE", path: "/posts/" + id + "/media/0", wantBody: "invalid media id"},
		{name: "like post invalid id", method: "POST", path: "/posts/not-a-uuid/like", wantBody: "invalid id"},
		{name: "unlike post invalid id", method: "DELETE", path: "/posts/not-a-uuid/like", wantBody: "invalid id"},
		{name: "create comment invalid id", method: "POST", path: "/posts/not-a-uuid/comments", jsonBody: dto.CreateCommentRequest{Body: "hi"}, wantBody: "invalid id"},
		{name: "create comment bad json", method: "POST", path: "/posts/" + id + "/comments", rawBody: "not json", contentType: postCtlJSON, wantBody: "invalid request body"},
		{name: "update comment invalid id", method: "PUT", path: "/comments/not-a-uuid", jsonBody: dto.UpdateCommentRequest{Body: "x"}, wantBody: "invalid id"},
		{name: "update comment bad json", method: "PUT", path: "/comments/" + id, rawBody: "not json", contentType: postCtlJSON, wantBody: "invalid request body"},
		{name: "delete comment invalid id", method: "DELETE", path: "/comments/not-a-uuid", wantBody: "invalid id"},
		{name: "upload comment media invalid id", method: "POST", path: "/comments/not-a-uuid/media", wantBody: "invalid id"},
		{name: "upload comment media no file", method: "POST", path: "/comments/" + id + "/media", contentType: postCtlMultipart, wantBody: "no media file provided"},
		{name: "like comment invalid id", method: "POST", path: "/comments/not-a-uuid/like", wantBody: "invalid id"},
		{name: "unlike comment invalid id", method: "DELETE", path: "/comments/not-a-uuid/like", wantBody: "invalid id"},
		{name: "list user posts invalid id", method: "GET", path: "/users/not-a-uuid/posts", anonymous: true, wantBody: "invalid id"},
		{name: "follow user invalid id", method: "POST", path: "/users/not-a-uuid/follow", wantBody: "invalid id"},
		{name: "unfollow user invalid id", method: "DELETE", path: "/users/not-a-uuid/follow", wantBody: "invalid id"},
		{name: "get follow stats invalid id", method: "GET", path: "/users/not-a-uuid/follow-stats", anonymous: true, wantBody: "invalid id"},
		{name: "get followers invalid id", method: "GET", path: "/users/not-a-uuid/followers", anonymous: true, wantBody: "invalid id"},
		{name: "get following invalid id", method: "GET", path: "/users/not-a-uuid/following", anonymous: true, wantBody: "invalid id"},
		{name: "vote poll invalid id", method: "POST", path: "/posts/not-a-uuid/poll/vote", jsonBody: dto.VotePollRequest{OptionID: 1}, wantBody: "invalid id"},
		{name: "vote poll bad json", method: "POST", path: "/posts/" + id + "/poll/vote", rawBody: "not json", contentType: postCtlJSON, wantBody: "invalid request body"},
		{name: "resolve suggestion invalid id", method: "POST", path: "/posts/not-a-uuid/resolve", wantBody: "invalid id"},
		{name: "unresolve suggestion invalid id", method: "DELETE", path: "/posts/not-a-uuid/resolve", wantBody: "invalid id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newPostHarness(t)
			_, cookie := postCtlViewer(h, !tc.anonymous)

			req := h.NewRequest(tc.method, tc.path).WithCookie(cookie)
			if tc.jsonBody != nil {
				req = req.WithJSONBody(tc.jsonBody)
			}
			if tc.contentType != "" {
				req = req.WithRawBody(tc.rawBody, tc.contentType)
			}

			// when
			status, body := req.Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListPostFeed(t *testing.T) {
	defaults := postCtlFeedQuery{tab: "everyone", corner: "general", page: bounds.NewPage(20, 0)}
	custom := postCtlFeedQuery{tab: "following", corner: "suggestions", search: "search term", sort: "top", seed: 42, page: bounds.NewPage(10, 5), resolved: "open"}
	cases := []struct {
		name     string
		url      string
		authed   bool
		want     postCtlFeedQuery
		err      error
		wantCode int
		wantBody string
	}{
		{name: "anonymous defaults", url: "/posts", want: defaults, wantCode: http.StatusOK, wantBody: `"total":3,`},
		{name: "custom query", url: "/posts?tab=following&corner=suggestions&search=search+term&sort=top&seed=42&limit=10&offset=5&resolved=open", want: custom, wantCode: http.StatusOK, wantBody: `"total":3,`},
		{name: "authenticated passes viewer id", url: "/posts", authed: true, want: defaults, wantCode: http.StatusOK, wantBody: `"total":3,`},
		{name: "internal error", url: "/posts", want: defaults, err: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list posts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			viewerID, cookie := postCtlViewer(h, tc.authed)
			deps.post.EXPECT().
				ListFeed(mock.Anything, tc.want.tab, viewerID, tc.want.corner, tc.want.search, tc.want.sort, tc.want.seed, tc.want.page, tc.want.resolved).
				Return(&dto.PostListResponse{Total: 3, Limit: 20}, tc.err)

			// when
			status, body := h.NewRequest("GET", tc.url).WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetCornerCounts(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusOK, `{"general":3}`},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to get counts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			deps.post.EXPECT().GetCornerCounts(mock.Anything).Return(map[string]int{"general": 3}, tc.err)

			// when
			status, body := h.NewRequest("GET", "/posts/corner-counts").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreatePost(t *testing.T) {
	postID := uuid.New()
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusCreated, `{"id":"` + postID.String() + `"}`},
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"invalid share type", postsvc.ErrInvalidShareType, http.StatusBadRequest, "invalid shared content type"},
		{"rate limited", postsvc.ErrRateLimited, http.StatusTooManyRequests, "daily post limit"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to create post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			req := dto.CreatePostRequest{Body: "hello"}
			deps.post.EXPECT().CreatePost(mock.Anything, userID, req).Return(postID, tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts").WithCookie(postCtlCookie).WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetPost(t *testing.T) {
	cases := []postCtlViewerOutcome{
		{"anonymous ok", false, nil, http.StatusOK, ""},
		{"authenticated ok", true, nil, http.StatusOK, ""},
		{"not found", false, postsvc.ErrNotFound, http.StatusNotFound, "post not found"},
		{"internal error", false, errors.New("boom"), http.StatusInternalServerError, "failed to get post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			viewerID, cookie := postCtlViewer(h, tc.authed)
			postID := uuid.New()
			deps.post.EXPECT().
				GetPost(mock.Anything, postID, viewerID, mock.AnythingOfType("string")).
				Return(&dto.PostDetailResponse{}, tc.err)

			// when
			status, body := h.NewRequest("GET", "/posts/"+postID.String()).WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdatePost(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"not owned", errors.Join(errors.New("post not found or not owned"), dao.ErrNotFound), http.StatusForbidden, "cannot update this post"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to update post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			req := dto.UpdatePostRequest{Body: "new"}
			deps.post.EXPECT().UpdatePost(mock.Anything, postID, userID, req).Return(tc.err)

			// when
			status, body := h.NewRequest("PUT", "/posts/"+postID.String()).WithCookie(postCtlCookie).WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeletePost(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to delete post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().DeletePost(mock.Anything, postID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/posts/"+postID.String()).WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUploadPostAndCommentMedia(t *testing.T) {
	shared := []postCtlOutcome{
		{"not found", postsvc.ErrNotFound, http.StatusNotFound, "not found"},
		{"too large", fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), http.StatusBadRequest, "exceeds maximum 5MB"},
		{"a server failure is not echoed back as a bad request", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to upload media"},
	}
	routes := []struct {
		name     string
		path     string
		expect   func(deps postDeps, id, userID uuid.UUID, err error)
		outcomes []postCtlOutcome
	}{
		{
			name: "post media",
			path: "/posts/:id/media",
			expect: func(deps postDeps, id, userID uuid.UUID, err error) {
				deps.post.EXPECT().UploadPostMedia(mock.Anything, id, userID, "image/png", "pic.png", mock.AnythingOfType("int64"), mock.Anything, false).Return(nil, err)
			},
			outcomes: append([]postCtlOutcome{{"not the post author", postsvc.ErrNotAuthor, http.StatusForbidden, "not the post author"}}, shared...),
		},
		{
			name: "comment media",
			path: "/comments/:id/media",
			expect: func(deps postDeps, id, userID uuid.UUID, err error) {
				deps.post.EXPECT().UploadCommentMedia(mock.Anything, id, userID, "image/png", "pic.png", mock.AnythingOfType("int64"), mock.Anything, false).Return(nil, err)
			},
			outcomes: append([]postCtlOutcome{{"not the comment author", authz.ErrNotCommentAuthor, http.StatusForbidden, "not the comment author"}}, shared...),
		},
	}

	for _, route := range routes {
		for _, tc := range route.outcomes {
			t.Run(route.name+": "+tc.name, func(t *testing.T) {
				// given
				h, deps, userID := postCtlSignedIn(t)
				id := uuid.New()
				form, contentType := testutil.MediaForm(t, "media", nil)
				route.expect(deps, id, userID, tc.err)

				// when
				status, body := h.NewRequest("POST", strings.ReplaceAll(route.path, ":id", id.String())).WithCookie(postCtlCookie).WithRawBody(form, contentType).Do()

				// then
				require.Equal(t, tc.wantCode, status)
				assert.Contains(t, string(body), tc.wantBody)
				assert.NotContains(t, string(body), "pq:")
			})
		}
	}
}

func TestDeletePostMedia(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"not found", postsvc.ErrNotFound, http.StatusNotFound, "post not found"},
		{"not the post author", postsvc.ErrNotAuthor, http.StatusForbidden, "not the post author"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to delete media"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().DeletePostMedia(mock.Anything, postID, int64(42), userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/posts/"+postID.String()+"/media/42").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestLikePost(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to like post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().LikePost(mock.Anything, userID, postID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/like").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlikePost(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to unlike post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().UnlikePost(mock.Anything, userID, postID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/posts/"+postID.String()+"/like").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestCreateComment(t *testing.T) {
	commentID := uuid.New()
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusCreated, `{"id":"` + commentID.String() + `"}`},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to create comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			req := dto.CreateCommentRequest{Body: "hello"}
			deps.post.EXPECT().CreateComment(mock.Anything, postID, userID, req).Return(commentID, tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/comments").WithCookie(postCtlCookie).WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateComment(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"empty body", postsvc.ErrEmptyBody, http.StatusBadRequest, "empty"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to update comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			commentID := uuid.New()
			req := dto.UpdateCommentRequest{Body: "new"}
			deps.post.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(tc.err)

			// when
			status, body := h.NewRequest("PUT", "/comments/"+commentID.String()).WithCookie(postCtlCookie).WithJSONBody(req).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteComment(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to delete comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			commentID := uuid.New()
			deps.post.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/comments/"+commentID.String()).WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestLikeComment(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"not found", postsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to like comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			commentID := uuid.New()
			deps.post.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/comments/"+commentID.String()+"/like").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlikeComment(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to unlike comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			commentID := uuid.New()
			deps.post.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/comments/"+commentID.String()+"/like").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListUserPosts(t *testing.T) {
	cases := []postCtlPagedOutcome{
		{"anonymous default paging", "", bounds.NewPage(20, 0), nil, http.StatusOK, `"total":3,`},
		{"custom paging", "?limit=5&offset=10", bounds.NewPage(5, 10), nil, http.StatusOK, `"total":3,`},
		{"internal error", "", bounds.NewPage(20, 0), errors.New("boom"), http.StatusInternalServerError, "failed to list user posts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			deps.post.EXPECT().ListUserPosts(mock.Anything, userID, uuid.Nil, tc.page).Return(&dto.PostListResponse{Total: 3}, tc.err)

			// when
			status, body := h.NewRequest("GET", "/users/"+userID.String()+"/posts"+tc.query).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestFollowUser(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"cannot follow self", follow.ErrCannotFollowSelf, http.StatusBadRequest, "cannot follow yourself"},
		{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to follow user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			targetID := uuid.New()
			deps.follow.EXPECT().Follow(mock.Anything, userID, targetID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/users/"+targetID.String()+"/follow").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnfollowUser(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusNoContent, ""},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to unfollow user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			targetID := uuid.New()
			deps.follow.EXPECT().Unfollow(mock.Anything, userID, targetID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/users/"+targetID.String()+"/follow").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetFollowStats(t *testing.T) {
	cases := []postCtlViewerOutcome{
		{"anonymous ok", false, nil, http.StatusOK, `"follower_count":2,`},
		{"authenticated ok", true, nil, http.StatusOK, `"follower_count":2,`},
		{"internal error", false, errors.New("boom"), http.StatusInternalServerError, "failed to get follow stats"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			viewerID, cookie := postCtlViewer(h, tc.authed)
			userID := uuid.New()
			deps.follow.EXPECT().GetFollowStats(mock.Anything, userID, viewerID).Return(&dto.FollowStatsResponse{FollowerCount: 2}, tc.err)

			// when
			status, body := h.NewRequest("GET", "/users/"+userID.String()+"/follow-stats").WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetFollowers(t *testing.T) {
	cases := []postCtlPagedOutcome{
		{"default paging", "", bounds.NewPage(50, 0), nil, http.StatusOK, `"total":1,`},
		{"custom paging", "?limit=5&offset=10", bounds.NewPage(5, 10), nil, http.StatusOK, `"total":1,`},
		{"internal error", "", bounds.NewPage(50, 0), errors.New("boom"), http.StatusInternalServerError, "failed to get followers"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			users := []dto.UserResponse{{ID: uuid.New(), Username: "beato"}}
			deps.follow.EXPECT().GetFollowers(mock.Anything, userID, tc.page).Return(users, 1, tc.err)

			// when
			status, body := h.NewRequest("GET", "/users/"+userID.String()+"/followers"+tc.query).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetFollowing(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusOK, `"total":1,`},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to get following"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			userID := uuid.New()
			users := []dto.UserResponse{{ID: uuid.New(), Username: "beato"}}
			deps.follow.EXPECT().GetFollowing(mock.Anything, userID, bounds.NewPage(50, 0)).Return(users, 1, tc.err)

			// when
			status, body := h.NewRequest("GET", "/users/"+userID.String()+"/following").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestVotePoll(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusOK, `"id":"p1"`},
		{"not found", postsvc.ErrNotFound, http.StatusNotFound, "poll not found"},
		{"expired", postsvc.ErrPollExpired, http.StatusGone, "expired"},
		{"already voted", postsvc.ErrAlreadyVoted, http.StatusConflict, "already voted"},
		{"invalid option", postsvc.ErrInvalidOption, http.StatusBadRequest, "invalid poll option"},
		{"internal error", errors.New("boom"), http.StatusInternalServerError, "failed to vote"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().VotePoll(mock.Anything, postID, userID, 3).Return(&dto.PollResponse{ID: "p1"}, tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/poll/vote").
				WithCookie(postCtlCookie).
				WithJSONBody(dto.VotePollRequest{OptionID: 3}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestResolveSuggestion(t *testing.T) {
	cases := []struct {
		name       string
		body       map[string]string
		wantStatus string
		err        error
		wantCode   int
		wantBody   string
	}{
		{"defaults to done", map[string]string{}, "done", nil, http.StatusOK, `"status":"ok"`},
		{"custom status", map[string]string{"status": "wont_fix"}, "wont_fix", nil, http.StatusOK, `"status":"ok"`},
		{"a user without the permission is forbidden", map[string]string{}, "done", postsvc.ErrNotAuthorised, http.StatusForbidden, "not authorised"},
		{"a server failure is a 500, not a 403 carrying the database error", map[string]string{}, "done", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to update the suggestion"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, tc.wantStatus).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/posts/"+postID.String()+"/resolve").WithCookie(postCtlCookie).WithJSONBody(tc.body).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnresolveSuggestion(t *testing.T) {
	cases := []postCtlOutcome{
		{"ok", nil, http.StatusOK, `"status":"ok"`},
		{"a user without the permission is forbidden", postsvc.ErrNotAuthorised, http.StatusForbidden, "not authorised"},
		{"a server failure is a 500, not a 403 carrying the database error", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to update the suggestion"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps, userID := postCtlSignedIn(t)
			postID := uuid.New()
			deps.post.EXPECT().UnresolveSuggestion(mock.Anything, postID, userID).Return(tc.err)

			// when
			status, body := h.NewRequest("DELETE", "/posts/"+postID.String()+"/resolve").WithCookie(postCtlCookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestGetShareCount(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		count       int
		err         error
		wantCode    int
		wantBody    string
	}{
		{"ok", "art", 7, nil, http.StatusOK, `"share_count":7`},
		{"a failed count is a server error, not a zero", "post", 0, errors.New("boom"), http.StatusInternalServerError, "failed to get share count"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newPostHarness(t)
			deps.post.EXPECT().GetShareCount(mock.Anything, "abc", tc.contentType).Return(tc.count, tc.err)

			// when
			status, body := h.NewRequest("GET", "/share-count/"+tc.contentType+"/abc").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}
