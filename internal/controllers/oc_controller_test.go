package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	ocsvc "umineko_city_of_books/internal/oc"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	ocCtlFilterRejection = &contentfilter.RejectedError{Rejection: contentfilter.Rejection{Rule: "slurs", Reason: "nope", Detail: "slur"}}
)

func newOCHarness(t *testing.T) (*testutil.Harness, *ocsvc.MockService) {
	h := testutil.NewHarness(t)
	os := ocsvc.NewMockService(t)

	s := &Service{
		OCService:    os,
		AuthSession:  h.SessionManager,
		AuthzService: h.AuthzService,
	}
	for _, setup := range s.getAllOCRoutes() {
		setup(h.App)
	}
	return h, os
}

func TestListOCs_Anonymous_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	expected := &dto.OCListResponse{Total: 0, Limit: 20, Offset: 0}
	os.EXPECT().ListOCs(mock.Anything, uuid.Nil, "new", false, "", "", uuid.Nil, bounds.NewPage(20, 0)).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/ocs").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.OCListResponse](t, body)
	assert.Equal(t, expected.Total, got.Total)
}

func TestListOCs_CustomQuery_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	ownerID := uuid.New()
	os.EXPECT().ListOCs(mock.Anything, uuid.Nil, "top", true, "umineko", "Higanbana", ownerID, bounds.NewPage(10, 5)).
		Return(&dto.OCListResponse{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/ocs?sort=top&crack=true&series=umineko&custom=Higanbana&user_id="+ownerID.String()+"&limit=10&offset=5").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestListOCs_InvalidUserID(t *testing.T) {
	// given
	h, _ := newOCHarness(t)

	// when
	status, body := h.NewRequest("GET", "/ocs?user_id=not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid user_id")
}

func TestListOCs_InternalError(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	os.EXPECT().ListOCs(mock.Anything, uuid.Nil, "new", false, "", "", uuid.Nil, bounds.NewPage(20, 0)).
		Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/ocs").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list ocs")
}

func TestGetOC_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	id := uuid.New()
	expected := &dto.OCDetailResponse{OCResponse: dto.OCResponse{ID: id, Name: "Linda"}}
	os.EXPECT().GetOC(mock.Anything, id, uuid.Nil).Return(expected, nil)

	// when
	status, body := h.NewRequest("GET", "/ocs/"+id.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.OCDetailResponse](t, body)
	assert.Equal(t, id, got.ID)
}

func TestGetOC_NotFound(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	id := uuid.New()
	os.EXPECT().GetOC(mock.Anything, id, uuid.Nil).Return(nil, ocsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/ocs/"+id.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "oc not found")
}

func TestGetOC_InvalidID(t *testing.T) {
	// given
	h, _ := newOCHarness(t)

	// when
	status, _ := h.NewRequest("GET", "/ocs/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateOC_RequiresAuth(t *testing.T) {
	// given
	h, _ := newOCHarness(t)

	// when
	status, _ := h.NewRequest("POST", "/ocs").WithJSONBody(dto.CreateOCRequest{Name: "Linda", Series: "umineko"}).Do()

	// then
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestCreateOC_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateOCRequest{Name: "Linda", Description: "bio", Series: "umineko"}
	os.EXPECT().CreateOC(mock.Anything, userID, req).Return(id, nil)

	// when
	status, body := h.NewRequest("POST", "/ocs").WithCookie("valid-cookie").WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	assert.Contains(t, string(body), id.String())
}

func TestCreateOC_BadRequestErrors(t *testing.T) {
	// given
	cases := []struct {
		name string
		err  error
	}{
		{"empty name", ocsvc.ErrEmptyName},
		{"invalid series", ocsvc.ErrInvalidSeries},
		{"empty custom", ocsvc.ErrEmptyCustomSeries},
		{"duplicate", ocsvc.ErrDuplicateName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, os := newOCHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			req := dto.CreateOCRequest{Name: "x", Series: "umineko"}
			os.EXPECT().CreateOC(mock.Anything, userID, req).Return(uuid.Nil, c.err)

			// when
			status, _ := h.NewRequest("POST", "/ocs").WithCookie("valid-cookie").WithJSONBody(req).Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
		})
	}
}

func TestCreateOC_InternalError(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateOCRequest{Name: "Linda", Series: "umineko"}
	os.EXPECT().CreateOC(mock.Anything, userID, req).Return(uuid.Nil, errors.New("boom"))

	// when
	status, _ := h.NewRequest("POST", "/ocs").WithCookie("valid-cookie").WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
}

func TestUpdateOC_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.UpdateOCRequest{Name: "Linda", Series: "umineko"}
	os.EXPECT().UpdateOC(mock.Anything, id, userID, req).Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/ocs/"+id.String()).WithCookie("valid-cookie").WithJSONBody(req).Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteOC_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	os.EXPECT().DeleteOC(mock.Anything, id, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/ocs/"+id.String()).WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestVoteOC_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	os.EXPECT().Vote(mock.Anything, userID, id, 1).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/ocs/"+id.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestVoteOC_InvalidValue(t *testing.T) {
	// given
	h, _ := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, _ := h.NewRequest("POST", "/ocs/"+id.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 7}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestVoteOC_BlockedReturnsForbidden(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	os.EXPECT().Vote(mock.Anything, userID, id, 1).Return(block.ErrUserBlocked)

	// when
	status, _ := h.NewRequest("POST", "/ocs/"+id.String()+"/vote").
		WithCookie("valid-cookie").
		WithJSONBody(dto.VoteRequest{Value: 1}).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
}

func TestFavouriteOC_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	os.EXPECT().ToggleFavourite(mock.Anything, userID, id).Return(true, nil)

	// when
	status, body := h.NewRequest("POST", "/ocs/"+id.String()+"/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), `"favourited":true`)
}

func TestFavouriteOC_NotFound(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	id := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	os.EXPECT().ToggleFavourite(mock.Anything, userID, id).Return(false, ocsvc.ErrNotFound)

	// when
	status, _ := h.NewRequest("POST", "/ocs/"+id.String()+"/favourite").
		WithCookie("valid-cookie").
		Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
}

func TestOCWriteRoutes_ServiceOutcomes(t *testing.T) {
	type outcome struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}

	notOwned := fmt.Errorf("oc not found or not owned: %w", dao.ErrNotFound)
	imageMissing := fmt.Errorf("gallery image not found or not in oc: %w", dao.ErrNotFound)
	commentNotOwned := fmt.Errorf("comment not found or not owned: %w", dao.ErrNotFound)
	tooLarge := fmt.Errorf("%w: file size 9MB exceeds maximum 5MB", upload.ErrFileTooLarge)
	dbDown := errors.New("pq: connection refused")
	ocMissing := outcome{"missing oc", ocsvc.ErrNotFound, http.StatusNotFound, "oc not found"}
	notOwner := outcome{"not the owner", ocsvc.ErrNotOwner, http.StatusForbidden, "cannot edit this oc"}

	routes := []struct {
		name     string
		method   string
		path     string
		json     any
		form     bool
		expect   func(os *ocsvc.MockService, id, userID uuid.UUID, err error)
		outcomes []outcome
	}{
		{
			name:   "update oc",
			method: "PUT",
			path:   "/ocs/:id",
			json:   dto.UpdateOCRequest{Name: "Linda", Series: "umineko"},
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().UpdateOC(mock.Anything, id, userID, dto.UpdateOCRequest{Name: "Linda", Series: "umineko"}).Return(err)
			},
			outcomes: []outcome{
				{"not owned", notOwned, http.StatusForbidden, "cannot update this oc"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to update oc"}`},
			},
		},
		{
			name:   "delete oc",
			method: "DELETE",
			path:   "/ocs/:id",
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().DeleteOC(mock.Anything, id, userID).Return(err)
			},
			outcomes: []outcome{
				ocMissing,
				{"not owned", notOwned, http.StatusForbidden, "cannot delete this oc"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to delete oc"}`},
			},
		},
		{
			name:   "upload portrait",
			method: "POST",
			path:   "/ocs/:id/image",
			form:   true,
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().UploadOCImage(mock.Anything, id, userID, "image/png", mock.AnythingOfType("int64"), mock.Anything).Return("", err)
			},
			outcomes: []outcome{
				ocMissing,
				notOwner,
				{"too large", tooLarge, http.StatusBadRequest, "exceeds maximum 5MB"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to upload image"}`},
			},
		},
		{
			name:   "add gallery image",
			method: "POST",
			path:   "/ocs/:id/gallery",
			form:   true,
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().AddGalleryImage(mock.Anything, id, userID, "", "image/png", mock.AnythingOfType("int64"), mock.Anything).Return(nil, err)
			},
			outcomes: []outcome{
				ocMissing,
				notOwner,
				{"caption rejected by the filter", ocCtlFilterRejection, http.StatusBadRequest, "content_rejected"},
				{"too large", tooLarge, http.StatusBadRequest, "exceeds maximum 5MB"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to add gallery image"}`},
			},
		},
		{
			name:   "update gallery image",
			method: "PATCH",
			path:   "/ocs/:id/gallery/7",
			json:   dto.UpdateOCImageRequest{},
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().UpdateGalleryImage(mock.Anything, id, int64(7), userID, dto.UpdateOCImageRequest{}).Return(err)
			},
			outcomes: []outcome{
				ocMissing,
				notOwner,
				{"missing image", imageMissing, http.StatusNotFound, "gallery image not found"},
				{"caption rejected by the filter", ocCtlFilterRejection, http.StatusBadRequest, "content_rejected"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to update gallery image"}`},
			},
		},
		{
			name:   "delete gallery image",
			method: "DELETE",
			path:   "/ocs/:id/gallery/7",
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().DeleteGalleryImage(mock.Anything, id, int64(7), userID).Return(err)
			},
			outcomes: []outcome{
				ocMissing,
				notOwner,
				{"missing image", imageMissing, http.StatusNotFound, "gallery image not found"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to delete gallery image"}`},
			},
		},
		{
			name:   "vote",
			method: "POST",
			path:   "/ocs/:id/vote",
			json:   dto.VoteRequest{Value: 1},
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().Vote(mock.Anything, userID, id, 1).Return(err)
			},
			outcomes: []outcome{
				ocMissing,
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to vote"}`},
			},
		},
		{
			name:   "create comment",
			method: "POST",
			path:   "/ocs/:id/comments",
			json:   dto.CreateCommentRequest{Body: "hi"},
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().CreateComment(mock.Anything, id, userID, dto.CreateCommentRequest{Body: "hi"}).Return(uuid.Nil, err)
			},
			outcomes: []outcome{
				ocMissing,
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to create comment"}`},
			},
		},
		{
			name:   "update comment",
			method: "PUT",
			path:   "/oc-comments/:id",
			json:   dto.UpdateCommentRequest{Body: "hi"},
			expect: func(os *ocsvc.MockService, id, userID uuid.UUID, err error) {
				os.EXPECT().UpdateComment(mock.Anything, id, userID, dto.UpdateCommentRequest{Body: "hi"}).Return(err)
			},
			outcomes: []outcome{
				{"missing comment", ocsvc.ErrNotFound, http.StatusNotFound, "comment not found"},
				{"not owned", commentNotOwned, http.StatusForbidden, "cannot update this comment"},
				{"server failure", dbDown, http.StatusInternalServerError, `{"error":"failed to update comment"}`},
			},
		},
	}

	for _, route := range routes {
		for _, tc := range route.outcomes {
			t.Run(route.name+": "+tc.name, func(t *testing.T) {
				// given
				h, os := newOCHarness(t)
				userID := uuid.New()
				id := uuid.New()
				h.ExpectValidSession("valid-cookie", userID)
				route.expect(os, id, userID, tc.err)

				req := h.NewRequest(route.method, strings.ReplaceAll(route.path, ":id", id.String())).WithCookie("valid-cookie")
				if route.json != nil {
					req = req.WithJSONBody(route.json)
				}
				if route.form {
					form, contentType := testutil.MediaForm(t, "image", nil)
					req = req.WithRawBody(form, contentType)
				}

				// when
				status, body := req.Do()

				// then
				require.Equal(t, tc.wantCode, status)
				assert.Contains(t, string(body), tc.wantBody)
				assert.NotContains(t, string(body), "pq:")
			})
		}
	}
}

func TestCreateOCComment_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	ocID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	req := dto.CreateCommentRequest{Body: "nice OC"}
	os.EXPECT().CreateComment(mock.Anything, ocID, userID, req).Return(commentID, nil)

	// when
	status, body := h.NewRequest("POST", "/ocs/"+ocID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(req).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	assert.Contains(t, string(body), commentID.String())
}

func TestListUserOCs_OK(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	os.EXPECT().ListOCsByUser(mock.Anything, userID, uuid.Nil, bounds.NewPage(20, 0)).
		Return(&dto.OCListResponse{Total: 3}, nil)

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/ocs").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.OCListResponse](t, body)
	assert.Equal(t, 3, got.Total)
}

func TestListUserOCSummaries_EmptyArray(t *testing.T) {
	// given
	h, os := newOCHarness(t)
	userID := uuid.New()
	os.EXPECT().ListOCSummariesByUser(mock.Anything, userID).Return(nil, nil)

	// when
	status, body := h.NewRequest("GET", "/users/"+userID.String()+"/oc-summaries").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "[]", string(body))
}
