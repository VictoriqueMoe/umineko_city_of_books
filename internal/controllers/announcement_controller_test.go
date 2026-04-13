package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type announcementDeps struct {
	repo      *repository.MockAnnouncementRepository
	blockSvc  *block.MockService
	notifSvc  *notification.MockService
	userRepo  *repository.MockUserRepository
	uploadSvc *upload.MockService
}

func newAnnouncementHarness(t *testing.T) (*testutil.Harness, announcementDeps) {
	h := testutil.NewHarness(t)
	deps := announcementDeps{
		repo:      repository.NewMockAnnouncementRepository(t),
		blockSvc:  block.NewMockService(t),
		notifSvc:  notification.NewMockService(t),
		userRepo:  repository.NewMockUserRepository(t),
		uploadSvc: upload.NewMockService(t),
	}

	s := &Service{
		AnnouncementRepo:    deps.repo,
		BlockService:        deps.blockSvc,
		NotificationService: deps.notifSvc,
		UserRepo:            deps.userRepo,
		UploadService:       deps.uploadSvc,
		SettingsService:     h.SettingsService,
		AuthSession:         h.SessionManager,
		AuthzService:        h.AuthzService,
		Hub:                 ws.NewHub(),
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
	authorID := uuid.New()
	rows := []repository.AnnouncementRow{
		{
			ID:                uuid.New(),
			Title:             "Welcome",
			Body:              "hi",
			AuthorID:          authorID,
			AuthorUsername:    "beato",
			AuthorDisplayName: "Beatrice",
			AuthorRole:        "admin",
		},
	}
	deps.repo.EXPECT().List(mock.Anything, 20, 0).Return(rows, 1, nil)

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
	deps.repo.EXPECT().List(mock.Anything, 5, 10).Return([]repository.AnnouncementRow{}, 0, nil)

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
	deps.repo.EXPECT().List(mock.Anything, 20, 0).Return(nil, 0, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/announcements").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to list announcements")
}

func TestGetAnnouncement_Anonymous_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	annID := uuid.New()
	row := &repository.AnnouncementRow{ID: annID, Title: "t", Body: "b"}
	deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(row, nil)
	deps.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	deps.repo.EXPECT().GetComments(mock.Anything, annID, uuid.Nil, 500, 0, []uuid.UUID(nil)).
		Return([]repository.AnnouncementCommentRow{}, 0, nil)
	deps.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{}).
		Return(map[uuid.UUID][]repository.AnnouncementCommentMediaRow{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/announcements/"+annID.String()).Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetAnnouncement_Authenticated_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	row := &repository.AnnouncementRow{ID: annID}
	deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(row, nil)
	deps.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, userID).Return([]uuid.UUID{}, nil)
	deps.repo.EXPECT().GetComments(mock.Anything, annID, userID, 500, 0, []uuid.UUID{}).
		Return([]repository.AnnouncementCommentRow{}, 0, nil)
	deps.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{}).
		Return(map[uuid.UUID][]repository.AnnouncementCommentMediaRow{}, nil)

	// when
	status, _ := h.NewRequest("GET", "/announcements/"+annID.String()).
		WithCookie("valid-cookie").Do()

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
	deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(nil, nil)

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
	deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(nil, errors.New("boom"))

	// when
	status, body := h.NewRequest("GET", "/announcements/"+annID.String()).Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to get announcement")
}

func TestGetLatestAnnouncement_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	row := &repository.AnnouncementRow{ID: uuid.New(), Title: "Welcome"}
	deps.repo.EXPECT().GetLatest(mock.Anything).Return(row, nil)

	// when
	status, _ := h.NewRequest("GET", "/announcements-latest").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestGetLatestAnnouncement_None(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.repo.EXPECT().GetLatest(mock.Anything).Return(nil, nil)

	// when
	status, body := h.NewRequest("GET", "/announcements-latest").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "\"announcement\":null")
}

func TestGetLatestAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	deps.repo.EXPECT().GetLatest(mock.Anything).Return(nil, errors.New("boom"))

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
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.repo.EXPECT().Create(mock.Anything, mock.AnythingOfType("uuid.UUID"), userID, "t", "b").Return(nil)

	// when
	status, body := h.NewRequest("POST", "/admin/announcements").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.NotEmpty(t, got["id"])
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
	cases := []struct {
		name string
		body map[string]string
	}{
		{"no title", map[string]string{"body": "b"}},
		{"no body", map[string]string{"title": "t"}},
		{"both empty", map[string]string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, _ := newAnnouncementHarness(t)
			userID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			h.ExpectHasPermission(userID, authz.PermManageSettings, true)

			// when
			status, body := h.NewRequest("POST", "/admin/announcements").
				WithCookie("valid-cookie").
				WithJSONBody(tc.body).
				Do()

			// then
			require.Equal(t, http.StatusBadRequest, status)
			assert.Contains(t, string(body), "title and body are required")
		})
	}
}

func TestCreateAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.repo.EXPECT().Create(mock.Anything, mock.AnythingOfType("uuid.UUID"), userID, "t", "b").
		Return(errors.New("boom"))

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
	deps.repo.EXPECT().Update(mock.Anything, annID, "t", "b").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestUpdateAnnouncement_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, body := h.NewRequest("PUT", "/admin/announcements/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]string{"title": "t", "body": "b"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateAnnouncement_BadJSON(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, _ := h.NewRequest("PUT", "/admin/announcements/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateAnnouncement_MissingFields(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, body := h.NewRequest("PUT", "/admin/announcements/"+uuid.NewString()).
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
	deps.repo.EXPECT().Update(mock.Anything, annID, "t", "b").Return(errors.New("boom"))

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
	deps.repo.EXPECT().Delete(mock.Anything, annID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/admin/announcements/"+annID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestDeleteAnnouncement_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, body := h.NewRequest("DELETE", "/admin/announcements/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.repo.EXPECT().Delete(mock.Anything, annID).Return(errors.New("boom"))

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
	deps.repo.EXPECT().SetPinned(mock.Anything, annID, true).Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/admin/announcements/"+annID.String()+"/pin").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"pinned": true}).
		Do()

	// then
	require.Equal(t, http.StatusOK, status)
}

func TestPinAnnouncement_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, body := h.NewRequest("POST", "/admin/announcements/not-a-uuid/pin").
		WithCookie("valid-cookie").
		WithJSONBody(map[string]bool{"pinned": true}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestPinAnnouncement_BadJSON(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, body := h.NewRequest("POST", "/admin/announcements/"+uuid.NewString()+"/pin").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestPinAnnouncement_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	deps.repo.EXPECT().SetPinned(mock.Anything, annID, false).Return(errors.New("boom"))

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
	authorID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ann := &repository.AnnouncementRow{ID: annID, AuthorID: authorID, Title: "Welcome"}
	deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(ann, nil)
	deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	deps.repo.EXPECT().CreateComment(mock.Anything, mock.AnythingOfType("uuid.UUID"), annID,
		(*uuid.UUID)(nil), userID, "hello").Return(nil)
	// goroutine fan-out - use Maybe since timing is non-deterministic
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).
		Return(&model.User{ID: userID, DisplayName: "Beato"}, nil).Maybe()
	h.SettingsService.EXPECT().Get(mock.Anything, mock.Anything).Return("http://test").Maybe()
	deps.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	status, body := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hello"}).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
	got := testutil.UnmarshalJSON[map[string]string](t, body)
	assert.NotEmpty(t, got["id"])
}

func TestCreateAnnouncementComment_OK_WithParent(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	authorID := uuid.New()
	parentAuthorID := uuid.New()
	annID := uuid.New()
	parentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	ann := &repository.AnnouncementRow{ID: annID, AuthorID: authorID, Title: "Welcome"}
	deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(ann, nil)
	deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	deps.repo.EXPECT().CreateComment(mock.Anything, mock.AnythingOfType("uuid.UUID"), annID,
		&parentID, userID, "reply").Return(nil)
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).
		Return(&model.User{ID: userID, DisplayName: "Beato"}, nil).Maybe()
	h.SettingsService.EXPECT().Get(mock.Anything, mock.Anything).Return("http://test").Maybe()
	deps.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, parentID).Return(parentAuthorID, nil).Maybe()

	// when
	status, _ := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "reply", ParentID: &parentID}).
		Do()

	// then
	require.Equal(t, http.StatusCreated, status)
}

func TestCreateAnnouncementComment_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/announcements/not-a-uuid/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestCreateAnnouncementComment_BadJSON(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("POST", "/announcements/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestCreateAnnouncementComment_EmptyBody(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/announcements/"+uuid.NewString()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "   "}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "body is required")
}

func TestCreateAnnouncementComment_AnnouncementNotFound(t *testing.T) {
	cases := []struct {
		name    string
		ann     *repository.AnnouncementRow
		repoErr error
	}{
		{"nil row", nil, nil},
		{"db error", nil, errors.New("boom")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAnnouncementHarness(t)
			userID := uuid.New()
			annID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.repo.EXPECT().GetByID(mock.Anything, annID).Return(tc.ann, tc.repoErr)

			// when
			status, body := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
				WithCookie("valid-cookie").
				WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
				Do()

			// then
			require.Equal(t, http.StatusNotFound, status)
			assert.Contains(t, string(body), "announcement not found")
		})
	}
}

func TestCreateAnnouncementComment_Blocked(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	authorID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetByID(mock.Anything, annID).
		Return(&repository.AnnouncementRow{ID: annID, AuthorID: authorID}, nil)
	deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	status, body := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "user is blocked")
}

func TestCreateAnnouncementComment_CreateError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	authorID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetByID(mock.Anything, annID).
		Return(&repository.AnnouncementRow{ID: annID, AuthorID: authorID}, nil)
	deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	deps.repo.EXPECT().CreateComment(mock.Anything, mock.AnythingOfType("uuid.UUID"), annID,
		(*uuid.UUID)(nil), userID, "hi").Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("POST", "/announcements/"+annID.String()+"/comments").
		WithCookie("valid-cookie").
		WithJSONBody(dto.CreateCommentRequest{Body: "hi"}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to create comment")
}

func TestUpdateAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "PUT", "/announcement-comments/"+uuid.NewString(),
		dto.UpdateCommentRequest{Body: "x"})
}

func TestUpdateAnnouncementComment_OK_AsAuthor(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditAnyComment, false)
	deps.repo.EXPECT().UpdateComment(mock.Anything, commentID, userID, "new body").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "new body"}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateAnnouncementComment_OK_AsAdmin(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditAnyComment, true)
	deps.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, commentID, "new body").Return(nil)

	// when
	status, _ := h.NewRequest("PUT", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "new body"}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUpdateAnnouncementComment_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/announcement-comments/not-a-uuid").
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUpdateAnnouncementComment_BadJSON(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, _ := h.NewRequest("PUT", "/announcement-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithRawBody("not json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestUpdateAnnouncementComment_EmptyBody(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("PUT", "/announcement-comments/"+uuid.NewString()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "   "}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "body is required")
}

func TestUpdateAnnouncementComment_AuthorForbidden(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditAnyComment, false)
	deps.repo.EXPECT().UpdateComment(mock.Anything, commentID, userID, "x").Return(errors.New("not yours"))

	// when
	status, body := h.NewRequest("PUT", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot update this comment")
}

func TestUpdateAnnouncementComment_AdminInternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermEditAnyComment, true)
	deps.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, commentID, "x").Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("PUT", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").
		WithJSONBody(dto.UpdateCommentRequest{Body: "x"}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to update comment")
}

func TestDeleteAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "DELETE", "/announcement-comments/"+uuid.NewString(), nil)
}

func TestDeleteAnnouncementComment_OK_AsAuthor(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermDeleteAnyComment, false)
	deps.repo.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteAnnouncementComment_OK_AsAdmin(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermDeleteAnyComment, true)
	deps.repo.EXPECT().DeleteCommentAsAdmin(mock.Anything, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestDeleteAnnouncementComment_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/announcement-comments/not-a-uuid").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestDeleteAnnouncementComment_AuthorForbidden(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermDeleteAnyComment, false)
	deps.repo.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("not yours"))

	// when
	status, body := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "cannot delete this comment")
}

func TestDeleteAnnouncementComment_AdminInternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermDeleteAnyComment, true)
	deps.repo.EXPECT().DeleteCommentAsAdmin(mock.Anything, commentID).Return(errors.New("boom"))

	// when
	status, body := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()).
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to delete comment")
}

func TestLikeAnnouncementComment_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, announcementFactory, "POST", "/announcement-comments/"+uuid.NewString()+"/like", nil)
}

func TestLikeAnnouncementComment_OK(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	authorID := uuid.New()
	annID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	deps.repo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)
	// goroutine fan-out
	deps.repo.EXPECT().GetCommentAnnouncementID(mock.Anything, commentID).Return(annID, nil).Maybe()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).
		Return(&model.User{ID: userID, DisplayName: "Beato"}, nil).Maybe()
	h.SettingsService.EXPECT().Get(mock.Anything, mock.Anything).Return("http://test").Maybe()
	deps.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	status, _ := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestLikeAnnouncementComment_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/not-a-uuid/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestLikeAnnouncementComment_CommentNotFound(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("not found"))

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "comment not found")
}

func TestLikeAnnouncementComment_Blocked(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	authorID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "user is blocked")
}

func TestLikeAnnouncementComment_LikeErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"blocked sentinel", block.ErrUserBlocked, http.StatusForbidden, "user is blocked"},
		{"internal", errors.New("boom"), http.StatusInternalServerError, "failed to like comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, deps := newAnnouncementHarness(t)
			userID := uuid.New()
			commentID := uuid.New()
			authorID := uuid.New()
			h.ExpectValidSession("valid-cookie", userID)
			deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
			deps.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
			deps.repo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(tc.err)

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
	deps.repo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	status, _ := h.NewRequest("DELETE", "/announcement-comments/"+commentID.String()+"/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestUnlikeAnnouncementComment_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("DELETE", "/announcement-comments/not-a-uuid/like").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUnlikeAnnouncementComment_InternalError(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

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

func TestUploadAnnouncementCommentMedia_InvalidID(t *testing.T) {
	// given
	h, _ := newAnnouncementHarness(t)
	h.ExpectValidSession("valid-cookie", uuid.New())

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/not-a-uuid/media").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid id")
}

func TestUploadAnnouncementCommentMedia_CommentNotFound(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("not found"))

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/media").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusNotFound, status)
	assert.Contains(t, string(body), "comment not found")
}

func TestUploadAnnouncementCommentMedia_NotAuthor(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	otherID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(otherID, nil)

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/media").
		WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "not the comment author")
}

func TestUploadAnnouncementCommentMedia_MissingFile(t *testing.T) {
	// given
	h, deps := newAnnouncementHarness(t)
	userID := uuid.New()
	commentID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	deps.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)

	// when
	status, body := h.NewRequest("POST", "/announcement-comments/"+commentID.String()+"/media").
		WithCookie("valid-cookie").
		WithRawBody("", "multipart/form-data; boundary=----xxx").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "no media file provided")
}
