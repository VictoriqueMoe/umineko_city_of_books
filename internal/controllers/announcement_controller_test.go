package controllers

import (
	"errors"
	"net/http"
	"testing"

	announcementsvc "umineko_city_of_books/internal/announcement"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type announcementDeps struct {
	svc *announcementsvc.MockService
}

func newAnnouncementHarness(t *testing.T) (*testutil.Harness, announcementDeps) {
	h := testutil.NewHarness(t)
	deps := announcementDeps{
		svc: announcementsvc.NewMockService(t),
	}
	s := &Service{
		AnnouncementService: deps.svc,
		SettingsService:     h.SettingsService,
		AuthSession:         h.SessionManager,
		AuthzService:        h.AuthzService,
	}
	for _, setup := range s.getAllAnnouncementRoutes() {
		setup(h.App)
	}
	return h, deps
}

func announcementFactory(t *testing.T) (*testutil.Harness, announcementDeps) {
	return newAnnouncementHarness(t)
}

func TestListAnnouncements_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.svc.EXPECT().List(mock.Anything, 20, 0).Return(&dto.AnnouncementListResponse{
		Announcements: []dto.AnnouncementResponse{{ID: uuid.New(), Title: "Welcome"}},
		Total:         1,
		Limit:         20,
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/announcements").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.EqualValues(t, 1, got["total"])
	assert.EqualValues(t, 20, got["limit"])
}

func TestListAnnouncements_CustomPaging(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.svc.EXPECT().List(mock.Anything, 5, 10).Return(&dto.AnnouncementListResponse{Limit: 5, Offset: 10}, nil)

	// when
	status, body := h.NewRequest("GET", "/announcements?limit=5&offset=10").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]any](t, body)
	assert.EqualValues(t, 5, got["limit"])
	assert.EqualValues(t, 10, got["offset"])
}

func TestListAnnouncements_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.svc.EXPECT().List(mock.Anything, 20, 0).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/announcements").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list announcements")
}

func TestGetAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	annID := uuid.New()
	deps.svc.EXPECT().GetDetail(mock.Anything, annID, uuid.Nil).
		Return(&dto.AnnouncementDetailResponse{
			AnnouncementResponse: dto.AnnouncementResponse{ID: annID, Title: "t"},
		}, nil)

	// when
	status, _ := h.NewRequest("GET", "/announcements/"+annID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetAnnouncement_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)

	// when
	status, body := h.NewRequest("GET", "/announcements/not-a-uuid").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestGetAnnouncement_NotFound(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	annID := uuid.New()
	deps.svc.EXPECT().GetDetail(mock.Anything, annID, uuid.Nil).Return(nil, announcementsvc.ErrNotFound)

	// when
	status, body := h.NewRequest("GET", "/announcements/"+annID.String()).Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "announcement not found")
}

func TestGetAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	annID := uuid.New()
	deps.svc.EXPECT().GetDetail(mock.Anything, annID, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/announcements/"+annID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get announcement")
}

func TestGetLatestAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	resp := &dto.AnnouncementResponse{ID: uuid.New(), Title: "Welcome"}
	deps.svc.EXPECT().GetLatest(mock.Anything).Return(resp, nil)

	// when
	status, _ := h.NewRequest("GET", "/announcements-latest").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetLatestAnnouncement_None(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.svc.EXPECT().GetLatest(mock.Anything).Return(nil, nil)

	// when
	status, body := h.NewRequest("GET", "/announcements-latest").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "\"announcement\":null")
}

func TestGetLatestAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.svc.EXPECT().GetLatest(mock.Anything).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/announcements-latest").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get latest announcement")
}

func TestCreateAnnouncement_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, announcementFactory, "POST", "/admin/announcements",
		map[string]string{"title": "t", "body": "b"}, authz.PermManageSettings)
}

func TestCreateAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Create(mock.Anything, userID, "t", "b").Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/admin/announcements").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), got["id"])
}

func TestCreateAnnouncement_BadJSON(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, body := h.NewRequest("POST", "/admin/announcements").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestCreateAnnouncement_MissingFields(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Create(mock.Anything, userID, "", "b").Return(uuid.Nil, announcementsvc.ErrEmptyTitleOrBody)

	// when
	status, body := h.NewRequest("POST", "/admin/announcements").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "title and body are required")
}

func TestCreateAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Create(mock.Anything, userID, "t", "b").Return(uuid.Nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/admin/announcements").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to create announcement")
}

func TestUpdateAnnouncement_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, announcementFactory, "PUT", "/admin/announcements/"+uuid.NewString(),
		map[string]string{"title": "t", "body": "b"}, authz.PermManageSettings)
}

func TestUpdateAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Update(mock.Anything, annID, "t", "b").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUpdateAnnouncement_MissingFields(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Update(mock.Anything, annID, "", "").Return(announcementsvc.ErrEmptyTitleOrBody)

	// when
	status, body := h.NewRequest("PUT", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": ""}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "title and body are required")
}

func TestUpdateAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Update(mock.Anything, annID, "t", "b").Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("PUT", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to update announcement")
}

func TestDeleteAnnouncement_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, announcementFactory, "DELETE", "/admin/announcements/"+uuid.NewString(),
		nil, authz.PermManageSettings)
}

func TestDeleteAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Delete(mock.Anything, annID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Delete(mock.Anything, annID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete announcement")
}

func TestPinAnnouncement_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, announcementFactory, "POST", "/admin/announcements/"+uuid.NewString()+"/pin",
		map[string]bool{"pinned": true}, authz.PermManageSettings)
}

func TestPinAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().SetPinned(mock.Anything, annID, true).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/announcements/"+annID.String()+"/pin").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"pinned": true}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestPinAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().SetPinned(mock.Anything, annID, false).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/admin/announcements/"+annID.String()+"/pin").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"pinned": false}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to pin announcement")
}

func TestCreateAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "POST", "/announcements/"+uuid.NewString()+"/comments",
		dto.CreateCommentRequest{Body: "hi"})
}

func TestCreateAnnouncementComment_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	newID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().CreateComment(mock.Anything, annID, userID, (*uuid.UUID)(nil), "hello").Return(newID, nil)

	// when
	status, body := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hello"}).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.Equal(t, newID.String(), got["id"])
}

func TestCreateAnnouncementComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty body", announcementsvc.ErrEmptyBody, http.StatusBadRequest, "body is required"},
		{"not found", announcementsvc.ErrNotFound, http.StatusNotFound, "announcement not found"},
		{"blocked", announcementsvc.ErrBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to create comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAnnouncementHarness(t)
			userID := uuid.New()
			annID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.svc.EXPECT().CreateComment(mock.Anything, annID, userID, (*uuid.UUID)(nil), "hi").Return(uuid.Nil, tc.err)

			// when
			status, body := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUpdateAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "PUT", "/announcement-comments/"+uuid.NewString(),
		dto.UpdateCommentRequest{Body: "x"})
}

func TestUpdateAnnouncementComment_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().UpdateComment(mock.Anything, commentID, userID, "new body").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "new body"}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateAnnouncementComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"empty", announcementsvc.ErrEmptyBody, http.StatusBadRequest, "body is required"},
		{"forbidden", announcementsvc.ErrForbidden, http.StatusForbidden, "cannot update this comment"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to update comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAnnouncementHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.svc.EXPECT().UpdateComment(mock.Anything, commentID, userID, "x").Return(tc.err)

			// when
			status, body := h.NewRequest("PUT", "/announcement-comments/"+commentID.String()).
				WithCookie("valid-cookie").
				WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
				Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestDeleteAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "DELETE", "/announcement-comments/"+uuid.NewString(), nil)
}

func TestDeleteAnnouncementComment_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteAnnouncementComment_Forbidden(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(announcementsvc.ErrForbidden)

	// when
	status, body := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot delete this comment")
}

func TestLikeAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "POST", "/announcement-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeAnnouncementComment_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeAnnouncementComment_ServiceErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not found", announcementsvc.ErrCommentNotFound, http.StatusNotFound, "comment not found"},
		{"blocked", announcementsvc.ErrBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to like comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAnnouncementHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.svc.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.err)

			// when
			status, body := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/like").
				WithCookie("valid-cookie").Do()

			// then
			require.Equal(t, tc.wantCode, status)
			assert.Contains(t, string(body), tc.wantBody)
		})
	}
}

func TestUnlikeAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "DELETE", "/announcement-comments/"+uuid.NewString()+"/like", nil)
}

func TestUnlikeAnnouncementComment_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeAnnouncementComment_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.svc.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to unlike comment")
}

func TestUploadAnnouncementCommentMedia_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "POST", "/announcement-comments/"+uuid.NewString()+"/media", nil)
}

func TestUploadAnnouncementCommentMedia_MissingFile(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/"+uuid.NewString()+"/media").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=----xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}
