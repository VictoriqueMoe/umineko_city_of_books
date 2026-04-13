package controllers

import (
	"errors"
	"net/http"
	"testing"

	artsvc "umineko_city_of_books/internal/art"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestListArt_Anonymous_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	expected := &dto.ArtListResponse{Total: 0, Limit: 24, Offset: 0}
	as.EXPECT().ListArt(mock.Anything, uuid.Nil, "general", "", "", "", "new", 24, 0).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/art").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ArtListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListArt_CustomQuery_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().ListArt(mock.Anything, uuid.Nil, "nsfw", "drawing", "beato", "cute", "top", 10, 5).
		Return(&dto.ArtListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/art?corner=nsfw&type=drawing&search=beato&tag=cute&sort=top&limit=10&offset=5").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListArt_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().ListArt(mock.Anything, userID, "general", "", "", "", "new", 24, 0).
		Return(&dto.ArtListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/art").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListArt_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().ListArt(mock.Anything, uuid.Nil, "general", "", "", "", "new", 24, 0).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/art").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list art")
}

func TestGetArtCornerCounts_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().GetCornerCounts(mock.Anything).Return(map[string]int{"general": 5}, nil)

	// when
	status, body := h.NewRequest("GET", "/art/corner-counts").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]int](t, body)
	assert.Equal(t, 5, got["general"])
}

func TestGetArtCornerCounts_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().GetCornerCounts(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/art/corner-counts").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get art counts")
}

func TestGetPopularTags_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	expected := []dto.TagCountResponse{{Tag: "cute", Count: 3}}
	as.EXPECT().GetPopularTags(mock.Anything, "").Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/art/tags").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[[]dto.TagCountResponse](t, body)
	require.Len(t, got, 1)
	assert.Equal(t, "cute", got[0].Tag)
}

func TestGetPopularTags_WithCorner_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().GetPopularTags(mock.Anything, "nsfw").Return([]dto.TagCountResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/art/tags?corner=nsfw").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetPopularTags_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().GetPopularTags(mock.Anything, "").Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/art/tags").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get tags")
}

func TestGetArt_Anonymous_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	artID := uuid.New()
	expected := &dto.ArtDetailResponse{ArtResponse: dto.ArtResponse{ID: artID, Title: "A piece"}}
	as.EXPECT().GetArt(mock.Anything, artID, uuid.Nil, mock.AnythingOfType("string")).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/art/"+artID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.ArtDetailResponse](t, body)
	assert.Equal(t, artID, got.ID)
}

func TestGetArt_Authenticated_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().GetArt(mock.Anything, artID, userID, mock.AnythingOfType("string")).
		Return(&dto.ArtDetailResponse{ArtResponse: dto.ArtResponse{ID: artID}}, nil)

	// when
	status, _ := h.NewRequest("GET", "/art/"+artID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetArt_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)

	// when
	status, body := h.NewRequest("GET", "/art/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetArt_NotFound(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	artID := uuid.New()
	as.EXPECT().GetArt(mock.Anything, artID, uuid.Nil, mock.AnythingOfType("string")).
		Return(nil, artsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/art/"+artID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "art not found")
}

func TestGetArt_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	artID := uuid.New()
	as.EXPECT().GetArt(mock.Anything, artID, uuid.Nil, mock.AnythingOfType("string")).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/art/"+artID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get art")
}

func TestCreateArt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "POST", "/art", nil)
}

func TestCreateArt_MissingMetadata_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/art").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "metadata is required")
}

func TestUpdateArt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "PUT", "/art/"+uuid.NewString(), dto.UpdateArtRequest{Title: "x"})
}

func TestUpdateArt_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/art/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateArtRequest{Title: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateArt_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/art/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateArt_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateArtRequest{Title: "Updated", Description: "new desc"}
	as.EXPECT().UpdateArt(mock.Anything, artID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/art/"+artID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateArt_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"empty title", artsvc.ErrEmptyTitle, http.StatusBadRequest},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			userID := uuid.New()
			artID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.UpdateArtRequest{Title: "x"}
			as.EXPECT().UpdateArt(mock.Anything, artID, userID, req).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("PUT", "/art/"+artID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestDeleteArt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "DELETE", "/art/"+uuid.NewString(), nil)
}

func TestDeleteArt_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/art/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteArt_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().DeleteArt(mock.Anything, artID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/art/"+artID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteArt_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().DeleteArt(mock.Anything, artID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/art/"+artID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestLikeArt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "POST", "/art/"+uuid.NewString()+"/like", nil)
}

func TestLikeArt_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/art/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeArt_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().LikeArt(mock.Anything, userID, artID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/art/"+artID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeArt_ServiceErrors(t *testing.T) {
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
			h, as := newArtHarness(t)
			userID := uuid.New()
			artID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			as.EXPECT().LikeArt(mock.Anything, userID, artID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/art/"+artID.String()+"/like").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUnlikeArt_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "DELETE", "/art/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeArt_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/art/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeArt_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().UnlikeArt(mock.Anything, userID, artID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/art/"+artID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeArt_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().UnlikeArt(mock.Anything, userID, artID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/art/"+artID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestCreateArtComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "POST", "/art/"+uuid.NewString()+"/comments", dto.CreateCommentRequest{Body: "hi"})
}

func TestCreateArtComment_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/art/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateArtComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/art/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateArtComment_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "hi"}
	as.EXPECT().CreateComment(mock.Anything, artID, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/art/"+artID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateArtComment_ServiceErrors(t *testing.T) {
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
			h, as := newArtHarness(t)
			userID := uuid.New()
			artID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateCommentRequest{Body: "hi"}
			as.EXPECT().CreateComment(mock.Anything, artID, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/art/"+artID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUpdateArtComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "PUT", "/art-comments/"+uuid.NewString(), dto.UpdateCommentRequest{Body: "x"})
}

func TestUpdateArtComment_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/art-comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateArtComment_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/art-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateArtComment_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateCommentRequest{Body: "updated"}
	as.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/art-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateArtComment_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateCommentRequest{Body: "updated"}
	as.EXPECT().UpdateComment(mock.Anything, commentID, userID, req).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("PUT", "/art-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestDeleteArtComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "DELETE", "/art-comments/"+uuid.NewString(), nil)
}

func TestDeleteArtComment_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/art-comments/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteArtComment_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/art-comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteArtComment_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/art-comments/"+commentID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestLikeArtComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "POST", "/art-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeArtComment_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/art-comments/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeArtComment_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/art-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeArtComment_ServiceErrors(t *testing.T) {
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
			h, as := newArtHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			as.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.svcErr)

			// when
			status, _ := h.NewRequest("POST", "/art-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
		})
	}
}

func TestUnlikeArtComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "DELETE", "/art-comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeArtComment_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/art-comments/not-a-uuid/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeArtComment_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/art-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeArtComment_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/art-comments/"+commentID.String()+"/like").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestUploadArtCommentMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "POST", "/art-comments/"+uuid.NewString()+"/media", nil)
}

func TestUploadArtCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/art-comments/not-a-uuid/media").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadArtCommentMedia_NoFile_BadRequest(t *testing.T) {
	// given — skip happy-path multipart test; cover auth/UUID/no-file branches only.
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/art-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}

func TestListUserArt_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	targetUserID := uuid.New()
	as.EXPECT().ListByUser(mock.Anything, targetUserID, uuid.Nil, 24, 0).Return(&dto.ArtListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/art").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserArt_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	viewerID := uuid.New()
	targetUserID := uuid.New()
	h.ExpectValidSession("valid-cookie", viewerID)
	as.EXPECT().ListByUser(mock.Anything, targetUserID, viewerID, 24, 0).Return(&dto.ArtListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/art").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserArt_CustomPaging(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	targetUserID := uuid.New()
	as.EXPECT().ListByUser(mock.Anything, targetUserID, uuid.Nil, 5, 10).Return(&dto.ArtListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/users/"+targetUserID.String()+"/art?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListUserArt_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/art").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserArt_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	targetUserID := uuid.New()
	as.EXPECT().ListByUser(mock.Anything, targetUserID, uuid.Nil, 24, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/art").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list user art")
}

func TestCreateGallery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "POST", "/galleries", dto.CreateGalleryRequest{Name: "g"})
}

func TestCreateGallery_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/galleries").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateGallery_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateGalleryRequest{Name: "My Gallery", Description: "desc"}
	as.EXPECT().CreateGallery(mock.Anything, userID, req).Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/galleries").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	resp := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), resp["id"])
}

func TestCreateGallery_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
		wantBody string
	}{
		{"empty title", artsvc.ErrEmptyTitle, http.StatusBadRequest, artsvc.ErrEmptyTitle.Error()},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create gallery"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, as := newArtHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateGalleryRequest{Name: "x"}
			as.EXPECT().CreateGallery(mock.Anything, userID, req).Return(uuid.Nil, tc.svcErr)

			// when
			status, body := h.NewRequest("POST", "/galleries").
				WithCookie("valid-cookie").
				WithJSONBody(req).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestListAllGalleries_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	expected := []dto.GalleryResponse{{ID: uuid.New(), Name: "g"}}
	as.EXPECT().ListAllGalleries(mock.Anything, "").Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/galleries").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[[]dto.GalleryResponse](t, body)
	require.Len(t, got, 1)
	assert.Equal(t, "g", got[0].Name)
}

func TestListAllGalleries_WithCorner_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().ListAllGalleries(mock.Anything, "nsfw").Return([]dto.GalleryResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/galleries?corner=nsfw").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListAllGalleries_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	as.EXPECT().ListAllGalleries(mock.Anything, "").Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/galleries").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list galleries")
}

func TestUpdateGallery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "PUT", "/galleries/"+uuid.NewString(), dto.UpdateGalleryRequest{Name: "g"})
}

func TestUpdateGallery_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/galleries/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateGalleryRequest{Name: "g"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateGallery_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/galleries/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateGallery_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateGalleryRequest{Name: "Updated"}
	as.EXPECT().UpdateGallery(mock.Anything, galleryID, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/galleries/"+galleryID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateGallery_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateGalleryRequest{Name: "Updated"}
	as.EXPECT().UpdateGallery(mock.Anything, galleryID, userID, req).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("PUT", "/galleries/"+galleryID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestSetGalleryCover_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "PUT", "/galleries/"+uuid.NewString()+"/cover", map[string]any{})
}

func TestSetGalleryCover_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/galleries/not-a-uuid/cover").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestSetGalleryCover_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/galleries/"+uuid.NewString()+"/cover").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestSetGalleryCover_OK_WithCover(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	coverID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().SetGalleryCover(mock.Anything, galleryID, userID, mock.MatchedBy(func(p *uuid.UUID) bool {
		return p != nil && *p == coverID
	})).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/galleries/"+galleryID.String()+"/cover").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"cover_art_id": coverID.String()}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestSetGalleryCover_OK_ClearCover(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().SetGalleryCover(mock.Anything, galleryID, userID, (*uuid.UUID)(nil)).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/galleries/"+galleryID.String()+"/cover").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"cover_art_id": nil}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestSetGalleryCover_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().SetGalleryCover(mock.Anything, galleryID, userID, (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("PUT", "/galleries/"+galleryID.String()+"/cover").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"cover_art_id": nil}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to set cover")
}

func TestDeleteGallery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "DELETE", "/galleries/"+uuid.NewString(), nil)
}

func TestDeleteGallery_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/galleries/not-a-uuid").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteGallery_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().DeleteGallery(mock.Anything, galleryID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/galleries/"+galleryID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteGallery_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().DeleteGallery(mock.Anything, galleryID, userID).Return(errors.New("boom"))

	// when
	status, _ := h.NewRequest("DELETE", "/galleries/"+galleryID.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestGetGallery_Anonymous_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	galleryID := uuid.New()
	gallery := &dto.GalleryResponse{ID: galleryID, Name: "g"}
	arts := []dto.ArtResponse{{ID: uuid.New(), Title: "piece"}}
	as.EXPECT().GetGallery(mock.Anything, galleryID, uuid.Nil, 24, 0).Return(gallery, arts, 1, nil)

	// when
	status, body := h.NewRequest("GET", "/galleries/"+galleryID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	resp := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.Equal(t, float64(1), resp["total"])
	assert.Equal(t, float64(24), resp["limit"])
	assert.Equal(t, float64(0), resp["offset"])
}

func TestGetGallery_Authenticated_PassesViewerID(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().GetGallery(mock.Anything, galleryID, userID, 5, 10).
		Return(&dto.GalleryResponse{ID: galleryID}, []dto.ArtResponse{}, 0, nil)

	// when
	status, _ := h.NewRequest("GET", "/galleries/"+galleryID.String()+"?limit=5&offset=10").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetGallery_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)

	// when
	status, body := h.NewRequest("GET", "/galleries/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetGallery_NotFound(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	galleryID := uuid.New()
	as.EXPECT().GetGallery(mock.Anything, galleryID, uuid.Nil, 24, 0).
		Return(nil, nil, 0, artsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/galleries/"+galleryID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "gallery not found")
}

func TestGetGallery_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	galleryID := uuid.New()
	as.EXPECT().GetGallery(mock.Anything, galleryID, uuid.Nil, 24, 0).
		Return(nil, nil, 0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/galleries/"+galleryID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get gallery")
}

func TestListUserGalleries_OK(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	targetUserID := uuid.New()
	expected := []dto.GalleryResponse{{ID: uuid.New(), Name: "g"}}
	as.EXPECT().ListUserGalleries(mock.Anything, targetUserID).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/galleries").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[[]dto.GalleryResponse](t, body)
	require.Len(t, got, 1)
	assert.Equal(t, "g", got[0].Name)
}

func TestListUserGalleries_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)

	// when
	status, body := h.NewRequest("GET", "/users/not-a-uuid/galleries").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestListUserGalleries_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	targetUserID := uuid.New()
	as.EXPECT().ListUserGalleries(mock.Anything, targetUserID).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/users/"+targetUserID.String()+"/galleries").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list galleries")
}

func TestSetArtGallery_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, newArtHarness, "PUT", "/art/"+uuid.NewString()+"/gallery", map[string]any{})
}

func TestSetArtGallery_InvalidID(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/art/not-a-uuid/gallery").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestSetArtGallery_BadJSON_BadRequest(t *testing.T) {
	// given
	h, _ := newArtHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/art/"+uuid.NewString()+"/gallery").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestSetArtGallery_OK_WithGallery(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	galleryID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().SetArtGallery(mock.Anything, artID, userID, mock.MatchedBy(func(p *uuid.UUID) bool {
		return p != nil && *p == galleryID
	})).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/art/"+artID.String()+"/gallery").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"gallery_id": galleryID.String()}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestSetArtGallery_OK_ClearGallery(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().SetArtGallery(mock.Anything, artID, userID, (*uuid.UUID)(nil)).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/art/"+artID.String()+"/gallery").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"gallery_id": nil}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestSetArtGallery_InternalError(t *testing.T) {
	// given
	h, as := newArtHarness(t)
	userID := uuid.New()
	artID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	as.EXPECT().SetArtGallery(mock.Anything, artID, userID, (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("PUT", "/art/"+artID.String()+"/gallery").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]any{"gallery_id": nil}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to set gallery")
}
