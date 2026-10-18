package controllers

import (
	"errors"
	"net/http"
	"testing"

	announcementsvc "umineko_city_of_books/internal/announcement"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/bounds"
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
		AuthService:         h.AuthService,
		AuthzService:        h.AuthzService,
	}
	for _, setup := range s.getAllAnnouncementRoutes() {
		setup(h.App)
	}
	return h, deps
}

func TestListAnnouncements_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.svc.EXPECT().List(mock.Anything, bounds.NewPage(20, 0)).Return(&dto.AnnouncementListResponse{
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
	deps.svc.EXPECT().List(mock.Anything, bounds.NewPage(5, 10)).Return(&dto.AnnouncementListResponse{Limit: 5, Offset: 10}, nil)

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
	deps.svc.EXPECT().List(mock.Anything, bounds.NewPage(20, 0)).Return(nil, errors.New("boom"))

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
			ID: annID, Title: "t",
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
	testutil.RunPermissionFailureSuite(t, newAnnouncementHarness, "POST", "/admin/announcements",
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
	testutil.RunPermissionFailureSuite(t, newAnnouncementHarness, "PUT", "/admin/announcements/"+uuid.NewString(),
		map[string]string{"title": "t", "body": "b"}, authz.PermManageSettings)
}

func TestUpdateAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Update(mock.Anything, userID, annID, "t", "b").Return(nil)

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
	deps.svc.EXPECT().Update(mock.Anything, userID, annID, "", "").Return(announcementsvc.ErrEmptyTitleOrBody)

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
	deps.svc.EXPECT().Update(mock.Anything, userID, annID, "t", "b").Return(errors.New("boom"))

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
	testutil.RunPermissionFailureSuite(t, newAnnouncementHarness, "DELETE", "/admin/announcements/"+uuid.NewString(),
		nil, authz.PermManageSettings)
}

func TestDeleteAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().Delete(mock.Anything, userID, annID).Return(nil)

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
	deps.svc.EXPECT().Delete(mock.Anything, userID, annID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete announcement")
}

func TestAnnouncementAdminWrites_AMissingAnnouncementIsNotFound(t *testing.T) {
	cases := []struct {
		name   string
		method string
		suffix string
		body   any
		expect func(deps announcementDeps, userID, annID uuid.UUID)
	}{
		{
			name:   "update",
			method: "PUT",
			body:   map[string]string{"title": "t", "body": "b"},
			expect: func(deps announcementDeps, userID, annID uuid.UUID) {
				deps.svc.EXPECT().Update(mock.Anything, userID, annID, "t", "b").Return(announcementsvc.ErrNotFound)
			},
		},
		{
			name:   "delete",
			method: "DELETE",
			expect: func(deps announcementDeps, userID, annID uuid.UUID) {
				deps.svc.EXPECT().Delete(mock.Anything, userID, annID).Return(announcementsvc.ErrNotFound)
			},
		},
		{
			name:   "pin",
			method: "POST",
			suffix: "/pin",
			body:   map[string]bool{"pinned": true},
			expect: func(deps announcementDeps, userID, annID uuid.UUID) {
				deps.svc.EXPECT().SetPinned(mock.Anything, userID, annID, true).Return(announcementsvc.ErrNotFound)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAnnouncementHarness(t)
			userID := uuid.New()
			annID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageSettings, true)
			tc.expect(deps, userID, annID)

			req := h.NewRequest(tc.method, "/admin/announcements/"+annID.String()+tc.suffix).WithCookie("valid-cookie")
			if tc.body != nil {
				req = req.WithJSONBody(tc.body)
			}

			// when
			status, body := req.Do()

			// then
			require.Equal(t, http.StatusNotFound, status)
			assert.Contains(t, string(body), "announcement not found")
		})
	}
}

func TestPinAnnouncement_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, newAnnouncementHarness, "POST", "/admin/announcements/"+uuid.NewString()+"/pin",
		map[string]bool{"pinned": true}, authz.PermManageSettings)
}

func TestPinAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.svc.EXPECT().SetPinned(mock.Anything, userID, annID, true).Return(nil)

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
	deps.svc.EXPECT().SetPinned(mock.Anything, userID, annID, false).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/admin/announcements/"+annID.String()+"/pin").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"pinned": false}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to pin announcement")
}
