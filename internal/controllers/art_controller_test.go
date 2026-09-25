package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	artCtlNotOwned = errors.Join(errors.New("not found or not owned"), dao.ErrNotFound)
)

type (
	artCtlOutcome struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}

	artCtlLinkOutcome struct {
		artCtlOutcome
		linked bool
	}
)

func newArtHarness(t *testing.T) (*testutil.Harness, *artsvc.MockService) {
	h := testutil.NewHarness(t)
	as := artsvc.NewMockService(t)

	s := &Service{
		ArtService:   as,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllArtRoutes() {
		setup(h.App)
	}

	return h, as
}

func artCtlSession(h *testutil.Harness, authenticated bool) (uuid.UUID, string) {
	if !authenticated {
		return uuid.Nil, ""
	}

	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	return userID, "valid-cookie"
}

func artCtlDo(h *testutil.Harness, method, path, cookie string, body any) (int, []byte) {
	req := h.NewRequest(method, path).WithCookie(cookie)
	if body != nil {
		req = req.WithJSONBody(body)
	}

	return req.Do()
}

func TestArtRoutes_AuthFailures(t *testing.T) {
	id := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"create art", "POST", "/art", nil},
		{"update art", "PUT", "/art/" + id, dto.UpdateArtRequest{Title: "x"}},
		{"delete art", "DELETE", "/art/" + id, nil},
		{"like art", "POST", "/art/" + id + "/like", nil},
		{"unlike art", "DELETE", "/art/" + id + "/like", nil},
		{"create art comment", "POST", "/art/" + id + "/comments", dto.CreateCommentRequest{Body: "hi"}},
		{"update art comment", "PUT", "/art-comments/" + id, dto.UpdateCommentRequest{Body: "x"}},
		{"delete art comment", "DELETE", "/art-comments/" + id, nil},
		{"like art comment", "POST", "/art-comments/" + id + "/like", nil},
		{"unlike art comment", "DELETE", "/art-comments/" + id + "/like", nil},
		{"upload art comment media", "POST", "/art-comments/" + id + "/media", nil},
		{"create gallery", "POST", "/galleries", dto.CreateGalleryRequest{Name: "g"}},
		{"update gallery", "PUT", "/galleries/" + id, dto.UpdateGalleryRequest{Name: "g"}},
		{"set gallery cover", "PUT", "/galleries/" + id + "/cover", map[string]any{}},
		{"delete gallery", "DELETE", "/galleries/" + id, nil},
		{"set art gallery", "PUT", "/art/" + id + "/gallery", map[string]any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.RunAuthFailureSuite(t, newArtHarness, tc.method, tc.path, tc.body)
		})
	}
}

func TestArtRoutes_InvalidID(t *testing.T) {
	cases := []struct {
		name          string
		method        string
		path          string
		body          any
		authenticated bool
	}{
		{"get art", "GET", "/art/not-a-uuid", nil, false},
		{"update art", "PUT", "/art/not-a-uuid", dto.UpdateArtRequest{Title: "x"}, true},
		{"delete art", "DELETE", "/art/not-a-uuid", nil, true},
		{"like art", "POST", "/art/not-a-uuid/like", nil, true},
		{"unlike art", "DELETE", "/art/not-a-uuid/like", nil, true},
		{"create art comment", "POST", "/art/not-a-uuid/comments", dto.CreateCommentRequest{Body: "hi"}, true},
		{"update art comment", "PUT", "/art-comments/not-a-uuid", dto.UpdateCommentRequest{Body: "x"}, true},
		{"delete art comment", "DELETE", "/art-comments/not-a-uuid", nil, true},
		{"like art comment", "POST", "/art-comments/not-a-uuid/like", nil, true},
		{"unlike art comment", "DELETE", "/art-comments/not-a-uuid/like", nil, true},
		{"upload art comment media", "POST", "/art-comments/not-a-uuid/media", nil, true},
		{"list user art", "GET", "/users/not-a-uuid/art", nil, false},
		{"update gallery", "PUT", "/galleries/not-a-uuid", dto.UpdateGalleryRequest{Name: "g"}, true},
		{"set gallery cover", "PUT", "/galleries/not-a-uuid/cover", map[string]any{}, true},
		{"delete gallery", "DELETE", "/galleries/not-a-uuid", nil, true},
		{"get gallery", "GET", "/galleries/not-a-uuid", nil, false},
		{"list user galleries", "GET", "/users/not-a-uuid/galleries", nil, false},
		{"set art gallery", "PUT", "/art/not-a-uuid/gallery", map[string]any{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newArtHarness(t)
			_, cookie := artCtlSession(h, tc.authenticated)

			// when
			status, body := artCtlDo(h, tc.method, tc.path, cookie, tc.body)

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "invalid id")
		})
	}
}

func TestArtRoutes_BadJSON(t *testing.T) {
	id := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"update art", "PUT", "/art/" + id},
		{"create art comment", "POST", "/art/" + id + "/comments"},
		{"update art comment", "PUT", "/art-comments/" + id},
		{"create gallery", "POST", "/galleries"},
		{"update gallery", "PUT", "/galleries/" + id},
		{"set gallery cover", "PUT", "/galleries/" + id + "/cover"},
		{"set art gallery", "PUT", "/art/" + id + "/gallery"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newArtHarness(t)
			_, cookie := artCtlSession(h, true)

			// when
			status, body := h.NewRequest(tc.method, tc.path).
				WithCookie(cookie).
				WithRawBody("not json", "application/json").
				Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "invalid request body")
		})
	}
}

func TestArtRoutes_Writes(t *testing.T) {
	newID := uuid.New()
	created := `{"id":"` + newID.String() + `"}`
	updateArt := dto.UpdateArtRequest{Title: "Updated", Description: "new desc"}
	createComment := dto.CreateCommentRequest{Body: "hi"}
	updateComment := dto.UpdateCommentRequest{Body: "updated"}
	createGallery := dto.CreateGalleryRequest{Name: "My Gallery", Description: "desc"}
	updateGallery := dto.UpdateGalleryRequest{Name: "Updated"}
	cases := []struct {
		name     string
		method   string
		path     string
		body     any
		expect   func(as *artsvc.MockService, id, userID uuid.UUID, err error)
		outcomes []artCtlOutcome
	}{
		{
			name:   "update art",
			method: "PUT",
			path:   "/art/:id",
			body:   updateArt,
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().UpdateArt(mock.Anything, id, userID, updateArt).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"updated", nil, http.StatusNoContent, ""},
				{"empty title", artsvc.ErrEmptyTitle, http.StatusBadRequest, artsvc.ErrEmptyTitle.Error()},
				{"not owned", artCtlNotOwned, http.StatusForbidden, "cannot update this art"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to update art"},
			},
		},
		{
			name:   "delete art",
			method: "DELETE",
			path:   "/art/:id",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().DeleteArt(mock.Anything, id, userID).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"deleted", nil, http.StatusNoContent, ""},
				{"not found", artsvc.ErrNotFound, http.StatusNotFound, "art not found"},
				{"not owned", artCtlNotOwned, http.StatusForbidden, "cannot delete this art"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete art"},
			},
		},
		{
			name:   "like art",
			method: "POST",
			path:   "/art/:id/like",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().LikeArt(mock.Anything, userID, id).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"liked", nil, http.StatusNoContent, ""},
				{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
				{"not found", artsvc.ErrNotFound, http.StatusNotFound, "art not found"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to like art"},
			},
		},
		{
			name:   "unlike art",
			method: "DELETE",
			path:   "/art/:id/like",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().UnlikeArt(mock.Anything, userID, id).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"unliked", nil, http.StatusNoContent, ""},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to unlike art"},
			},
		},
		{
			name:   "create art comment",
			method: "POST",
			path:   "/art/:id/comments",
			body:   createComment,
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().CreateComment(mock.Anything, id, userID, createComment).Return(newID, err)
			},
			outcomes: []artCtlOutcome{
				{"created", nil, http.StatusCreated, created},
				{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
				{"empty body", artsvc.ErrEmptyBody, http.StatusBadRequest, artsvc.ErrEmptyBody.Error()},
				{"not found", artsvc.ErrNotFound, http.StatusNotFound, "art not found"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create comment"},
			},
		},
		{
			name:   "update art comment",
			method: "PUT",
			path:   "/art-comments/:id",
			body:   updateComment,
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().UpdateComment(mock.Anything, id, userID, updateComment).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"updated", nil, http.StatusNoContent, ""},
				{"empty body", artsvc.ErrEmptyBody, http.StatusBadRequest, artsvc.ErrEmptyBody.Error()},
				{"not found", artsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
				{"not owned", artCtlNotOwned, http.StatusForbidden, "cannot update this comment"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to update comment"},
			},
		},
		{
			name:   "delete art comment",
			method: "DELETE",
			path:   "/art-comments/:id",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().DeleteComment(mock.Anything, id, userID).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"deleted", nil, http.StatusNoContent, ""},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete comment"},
			},
		},
		{
			name:   "like art comment",
			method: "POST",
			path:   "/art-comments/:id/like",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().LikeComment(mock.Anything, userID, id).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"liked", nil, http.StatusNoContent, ""},
				{"blocked", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to like comment"},
			},
		},
		{
			name:   "unlike art comment",
			method: "DELETE",
			path:   "/art-comments/:id/like",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().UnlikeComment(mock.Anything, userID, id).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"unliked", nil, http.StatusNoContent, ""},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to unlike comment"},
			},
		},
		{
			name:   "create gallery",
			method: "POST",
			path:   "/galleries",
			body:   createGallery,
			expect: func(as *artsvc.MockService, _, userID uuid.UUID, err error) {
				as.EXPECT().CreateGallery(mock.Anything, userID, createGallery).Return(newID, err)
			},
			outcomes: []artCtlOutcome{
				{"created", nil, http.StatusCreated, created},
				{"empty title", artsvc.ErrEmptyTitle, http.StatusBadRequest, artsvc.ErrEmptyTitle.Error()},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create gallery"},
			},
		},
		{
			name:   "update gallery",
			method: "PUT",
			path:   "/galleries/:id",
			body:   updateGallery,
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().UpdateGallery(mock.Anything, id, userID, updateGallery).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"updated", nil, http.StatusNoContent, ""},
				{"empty name", artsvc.ErrEmptyTitle, http.StatusBadRequest, artsvc.ErrEmptyTitle.Error()},
				{"not owned", artCtlNotOwned, http.StatusForbidden, "cannot update this gallery"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to update gallery"},
			},
		},
		{
			name:   "delete gallery",
			method: "DELETE",
			path:   "/galleries/:id",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, err error) {
				as.EXPECT().DeleteGallery(mock.Anything, id, userID).Return(err)
			},
			outcomes: []artCtlOutcome{
				{"deleted", nil, http.StatusNoContent, ""},
				{"not found", artsvc.ErrNotFound, http.StatusNotFound, "gallery not found"},
				{"not owned", artCtlNotOwned, http.StatusForbidden, "cannot delete this gallery"},
				{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to delete gallery"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, outcome := range tc.outcomes {
				t.Run(outcome.name, func(t *testing.T) {
					// given
					h, as := newArtHarness(t)
					userID, cookie := artCtlSession(h, true)
					id := uuid.New()
					tc.expect(as, id, userID, outcome.svcErr)

					// when
					status, body := artCtlDo(h, tc.method, strings.ReplaceAll(tc.path, ":id", id.String()), cookie, tc.body)

					// then
					require.Equal(t, outcome.wantCode, status)
					assert.Contains(t, string(body), outcome.wantBody)
				})
			}
		})
	}
}

func TestArtRoutes_OptionalLinks(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		field    string
		expect   func(as *artsvc.MockService, id, userID uuid.UUID, target *uuid.UUID, err error)
		outcomes []artCtlLinkOutcome
	}{
		{
			name:  "set gallery cover",
			path:  "/galleries/:id/cover",
			field: "cover_art_id",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, target *uuid.UUID, err error) {
				as.EXPECT().SetGalleryCover(mock.Anything, id, userID, target).Return(err)
			},
			outcomes: []artCtlLinkOutcome{
				{artCtlOutcome{"sets the cover", nil, http.StatusNoContent, ""}, true},
				{artCtlOutcome{"clears the cover", nil, http.StatusNoContent, ""}, false},
				{artCtlOutcome{"foreign art", dao.ErrArtNotOwned, http.StatusNotFound, "gallery or art not found"}, false},
				{artCtlOutcome{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to set cover"}, false},
			},
		},
		{
			name:  "set art gallery",
			path:  "/art/:id/gallery",
			field: "gallery_id",
			expect: func(as *artsvc.MockService, id, userID uuid.UUID, target *uuid.UUID, err error) {
				as.EXPECT().SetArtGallery(mock.Anything, id, userID, target).Return(err)
			},
			outcomes: []artCtlLinkOutcome{
				{artCtlOutcome{"sets the gallery", nil, http.StatusNoContent, ""}, true},
				{artCtlOutcome{"clears the gallery", nil, http.StatusNoContent, ""}, false},
				{artCtlOutcome{"foreign gallery", dao.ErrArtNotOwned, http.StatusNotFound, "art or gallery not found"}, true},
				{artCtlOutcome{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to set gallery"}, false},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, outcome := range tc.outcomes {
				t.Run(outcome.name, func(t *testing.T) {
					// given
					h, as := newArtHarness(t)
					userID, cookie := artCtlSession(h, true)
					id := uuid.New()

					var target *uuid.UUID
					if outcome.linked {
						target = new(uuid.New())
					}

					tc.expect(as, id, userID, target, outcome.svcErr)

					// when
					status, body := artCtlDo(h, "PUT", strings.ReplaceAll(tc.path, ":id", id.String()), cookie, map[string]any{tc.field: target})

					// then
					require.Equal(t, outcome.wantCode, status)
					assert.Contains(t, string(body), outcome.wantBody)
				})
			}
		})
	}
}

func TestListArt(t *testing.T) {
	cases := []struct {
		name          string
		authenticated bool
		query         string
		corner        string
		artType       string
		search        string
		tag           string
		sort          string
		limit         int
		offset        int
		svcErr        error
		wantCode      int
		wantBody      string
	}{
		{name: "anonymous gets the defaults", corner: "general", sort: "new", limit: 24, wantCode: http.StatusOK},
		{name: "authenticated passes the viewer", authenticated: true, corner: "general", sort: "new", limit: 24, wantCode: http.StatusOK},
		{name: "custom query", query: "?corner=nsfw&type=drawing&search=beato&tag=cute&sort=top&limit=10&offset=5", corner: "nsfw", artType: "drawing", search: "beato", tag: "cute", sort: "top", limit: 10, offset: 5, wantCode: http.StatusOK},
		{name: "internal", corner: "general", sort: "new", limit: 24, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list art"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			viewerID, cookie := artCtlSession(h, tc.authenticated)
			as.EXPECT().ListArt(mock.Anything, viewerID, tc.corner, tc.artType, tc.search, tc.tag, tc.sort, bounds.NewPage(tc.limit, tc.offset)).
				Return(&dto.ArtListResponse{Total: 7}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/art"+tc.query).WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[dto.ArtListResponse](t, body)
				assert.Equal(t, 7, got.Total)
			}
		})
	}
}

func TestGetArtCornerCounts(t *testing.T) {
	cases := []artCtlOutcome{
		{"ok", nil, http.StatusOK, ""},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to get art counts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			as.EXPECT().GetCornerCounts(mock.Anything).Return(map[string]int{"general": 5}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/art/corner-counts").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[map[string]int](t, body)
				assert.Equal(t, 5, got["general"])
			}
		})
	}
}

func TestGetPopularTags(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		corner   string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"every corner", "", "", nil, http.StatusOK, ""},
		{"one corner", "?corner=nsfw", "nsfw", nil, http.StatusOK, ""},
		{"internal", "", "", errors.New("boom"), http.StatusInternalServerError, "failed to get tags"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			as.EXPECT().GetPopularTags(mock.Anything, tc.corner).Return([]dto.TagCountResponse{{Tag: "cute", Count: 3}}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/art/tags"+tc.query).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[[]dto.TagCountResponse](t, body)
				require.Len(t, got, 1)
				assert.Equal(t, "cute", got[0].Tag)
			}
		})
	}
}

func TestGetArt(t *testing.T) {
	cases := []struct {
		name          string
		authenticated bool
		svcErr        error
		wantCode      int
		wantBody      string
	}{
		{"anonymous", false, nil, http.StatusOK, ""},
		{"authenticated passes the viewer", true, nil, http.StatusOK, ""},
		{"not found", false, artsvc.ErrNotFound, http.StatusNotFound, "art not found"},
		{"internal", false, errors.New("boom"), http.StatusInternalServerError, "failed to get art"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			viewerID, cookie := artCtlSession(h, tc.authenticated)
			artID := uuid.New()
			as.EXPECT().GetArt(mock.Anything, artID, viewerID, mock.AnythingOfType("string")).
				Return(&dto.ArtDetailResponse{ArtResponse: dto.ArtResponse{ID: artID, Title: "A piece"}}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/art/"+artID.String()).WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[dto.ArtDetailResponse](t, body)
				assert.Equal(t, artID, got.ID)
			}
		})
	}
}

func TestCreateArt_MissingMetadata_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	_, cookie := artCtlSession(h, true)

	// when
	status, body := h.NewRequest("POST", "/art").
		WithCookie(cookie).
		WithRawBody("", "multipart/form-data; boundary=xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "metadata is required")
}

func TestCreateArt_ServiceOutcomes(t *testing.T) {
	cases := []artCtlOutcome{
		{"empty title", artsvc.ErrEmptyTitle, http.StatusBadRequest, artsvc.ErrEmptyTitle.Error()},
		{"too large", fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge), http.StatusBadRequest, "exceeds maximum 5MB"},
		{"a server failure is a 500, not a 400 carrying the database error", errors.New("pq: connection refused"), http.StatusInternalServerError, "failed to create art"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			userID, cookie := artCtlSession(h, true)
			form, contentType := testutil.MediaForm(t, "image", map[string]string{"metadata": `{"title":"Rokkenjima"}`})
			as.EXPECT().CreateArt(mock.Anything, userID, dto.CreateArtRequest{Title: "Rokkenjima"}, "image/png", mock.AnythingOfType("int64"), mock.Anything).Return(uuid.Nil, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/art").WithCookie(cookie).WithRawBody(form, contentType).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
			assert.NotContains(t, string(body), "pq:")
		})
	}
}

func TestUploadArtCommentMedia_NoFile_BadRequest(t *testing.T) {
	// given: skip happy-path multipart test; cover auth/UUID/no-file branches only.
	h, _ := newArtHarness(t)
	_, cookie := artCtlSession(h, true)

	// when
	status, body := h.NewRequest("POST", "/art-comments/"+uuid.NewString()+"/media").
		WithCookie(cookie).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestListUserArt(t *testing.T) {
	cases := []struct {
		name          string
		authenticated bool
		query         string
		limit         int
		offset        int
		svcErr        error
		wantCode      int
		wantBody      string
	}{
		{name: "anonymous gets the default page", limit: 24, wantCode: http.StatusOK},
		{name: "authenticated passes the viewer", authenticated: true, limit: 24, wantCode: http.StatusOK},
		{name: "custom paging", query: "?limit=5&offset=10", limit: 5, offset: 10, wantCode: http.StatusOK},
		{name: "internal", limit: 24, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to list user art"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			viewerID, cookie := artCtlSession(h, tc.authenticated)
			targetUserID := uuid.New()
			as.EXPECT().ListByUser(mock.Anything, targetUserID, viewerID, bounds.NewPage(tc.limit, tc.offset)).
				Return(&dto.ArtListResponse{Total: 7}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/art"+tc.query).WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[dto.ArtListResponse](t, body)
				assert.Equal(t, 7, got.Total)
			}
		})
	}
}

func TestListAllGalleries(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		corner   string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"every corner", "", "", nil, http.StatusOK, ""},
		{"one corner", "?corner=nsfw", "nsfw", nil, http.StatusOK, ""},
		{"internal", "", "", errors.New("boom"), http.StatusInternalServerError, "failed to list galleries"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			as.EXPECT().ListAllGalleries(mock.Anything, tc.corner).Return([]dto.GalleryResponse{{ID: uuid.New(), Name: "g"}}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/galleries"+tc.query).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[[]dto.GalleryResponse](t, body)
				require.Len(t, got, 1)
				assert.Equal(t, "g", got[0].Name)
			}
		})
	}
}

func TestGetGallery(t *testing.T) {
	cases := []struct {
		name          string
		authenticated bool
		query         string
		limit         int
		offset        int
		svcErr        error
		wantCode      int
		wantBody      string
	}{
		{name: "anonymous gets the default page", limit: 24, wantCode: http.StatusOK},
		{name: "authenticated passes the viewer and custom paging", authenticated: true, query: "?limit=5&offset=10", limit: 5, offset: 10, wantCode: http.StatusOK},
		{name: "not found", limit: 24, svcErr: artsvc.ErrNotFound, wantCode: http.StatusNotFound, wantBody: "gallery not found"},
		{name: "internal", limit: 24, svcErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantBody: "failed to get gallery"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			viewerID, cookie := artCtlSession(h, tc.authenticated)
			galleryID := uuid.New()
			gallery := &dto.GalleryResponse{ID: galleryID, Name: "g"}
			arts := []dto.ArtResponse{{ID: uuid.New(), Title: "piece"}}
			as.EXPECT().GetGallery(mock.Anything, galleryID, viewerID, bounds.NewPage(tc.limit, tc.offset)).Return(gallery, arts, 1, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/galleries/"+galleryID.String()+tc.query).WithCookie(cookie).Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				resp := testutil.UnmarshalJSON[map[string]any](t, body)
				assert.Equal(t, float64(1), resp["total"])
				assert.Equal(t, float64(tc.limit), resp["limit"])
				assert.Equal(t, float64(tc.offset), resp["offset"])
			}
		})
	}
}

func TestListUserGalleries(t *testing.T) {
	cases := []artCtlOutcome{
		{"ok", nil, http.StatusOK, ""},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to list galleries"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			targetUserID := uuid.New()
			as.EXPECT().ListUserGalleries(mock.Anything, targetUserID).Return([]dto.GalleryResponse{{ID: uuid.New(), Name: "g"}}, tc.svcErr)

			// when
			status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/galleries").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)

			if tc.svcErr == nil {
				got := testutil.UnmarshalJSON[[]dto.GalleryResponse](t, body)
				require.Len(t, got, 1)
				assert.Equal(t, "g", got[0].Name)
			}
		})
	}
}
