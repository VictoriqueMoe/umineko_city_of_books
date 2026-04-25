package controllers

import (
	"errors"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newHomeHarness(t *testing.T) (*testutil.Harness, *repository.MockHomeFeedRepository) {
	h, homeRepo, _ := newHomeHarnessFull(t)
	return h, homeRepo
}

func newHomeHarnessFull(t *testing.T) (*testutil.Harness, *repository.MockHomeFeedRepository, *repository.MockSidebarLastVisitedRepository) {
	h := testutil.NewHarness(t)
	homeRepo := repository.NewMockHomeFeedRepository(t)
	sidebarRepo := repository.NewMockSidebarLastVisitedRepository(t)

	s := &Service{
		HomeFeedRepo:       homeRepo,
		SidebarVisitedRepo: sidebarRepo,
		AuthSession:        h.SessionManager,
		AuthzService:       h.AuthzService,
		Hub:                ws.NewHub(),
	}
	for _, setup := range s.getAllHomeRoutes() {
		setup(h.App)
	}
	return h, homeRepo, sidebarRepo
}

func TestGetSidebarActivity_OK(t *testing.T) {
	// given
	h, repo := newHomeHarness(t)
	repo.EXPECT().ListSidebarActivity(mock.Anything).Return([]repository.SidebarActivityEntry{
		{Key: "game_board_umineko", LatestAt: "2026-04-23T10:00:00Z"},
		{Key: "rooms", LatestAt: "2026-04-24T09:30:00Z"},
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/sidebar/activity").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SidebarActivityResponse](t, body)
	assert.Equal(t, "2026-04-23T10:00:00Z", got.Activity["game_board_umineko"])
	assert.Equal(t, "2026-04-24T09:30:00Z", got.Activity["rooms"])
}

func TestGetSidebarActivity_RepoError(t *testing.T) {
	// given
	h, repo := newHomeHarness(t)
	repo.EXPECT().ListSidebarActivity(mock.Anything).Return(nil, errors.New("db down"))

	// when
	status, body := h.NewRequest("GET", "/sidebar/activity").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load sidebar activity")
}

func expectHomeActivitySuccess(repo *repository.MockHomeFeedRepository) {
	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{}, nil)
	repo.EXPECT().ListPublicRooms(mock.Anything, 5).Return([]repository.HomePublicRoomRow{}, nil)
	repo.EXPECT().ListCornerActivity24h(mock.Anything).Return([]repository.HomeCornerActivityRow{}, nil)
}

func TestGetHomeActivity_OK_EmptyEverything(t *testing.T) {
	// given
	h, repo := newHomeHarness(t)
	expectHomeActivitySuccess(repo)

	// when
	status, body := h.NewRequest("GET", "/home/activity").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.HomeActivityResponse](t, body)
	assert.Equal(t, 0, got.OnlineCount)
	assert.Empty(t, got.RecentActivity)
	assert.Empty(t, got.RecentMembers)
	assert.Empty(t, got.PublicRooms)
	assert.Empty(t, got.CornerActivity)
}

func TestGetHomeActivity_OK_MapsAllSections(t *testing.T) {
	// given
	h, repo := newHomeHarness(t)
	theoryID := uuid.New()
	postID := uuid.New()
	journalID := uuid.New()
	artID := uuid.New()
	memberID := uuid.New()
	roomID := uuid.New()
	authorID := uuid.New()
	lastMsg := "2026-04-24T12:00:00Z"
	lastPost := "2026-04-24T11:00:00Z"

	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{
		{Kind: "theory", ID: theoryID, Title: "T", Body: "x", Corner: "umineko", CreatedAt: "2026-04-24T10:00:00Z", AuthorID: authorID, Username: "u", DisplayName: "U", AvatarURL: "a"},
		{Kind: "post", ID: postID, Title: "", Body: "p", Corner: "general", CreatedAt: "2026-04-24T09:00:00Z", AuthorID: authorID, Username: "u", DisplayName: "U"},
		{Kind: "journal", ID: journalID, Title: "J", Body: "j", Corner: "umineko", CreatedAt: "2026-04-24T08:00:00Z", AuthorID: authorID, Username: "u", DisplayName: "U"},
		{Kind: "art", ID: artID, Title: "A", Body: "", Corner: "higurashi", CreatedAt: "2026-04-24T07:00:00Z", AuthorID: authorID, Username: "u", DisplayName: "U"},
	}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{
		{ID: memberID, Username: "newbie", DisplayName: "Newbie", CreatedAt: "2026-04-24T00:00:00Z"},
	}, nil)
	repo.EXPECT().ListPublicRooms(mock.Anything, 5).Return([]repository.HomePublicRoomRow{
		{ID: roomID, Name: "Hangout", Description: "d", MemberCount: 3, LastMessageAt: &lastMsg},
	}, nil)
	repo.EXPECT().ListCornerActivity24h(mock.Anything).Return([]repository.HomeCornerActivityRow{
		{Corner: "umineko", PostCount: 7, UniquePosters: 3, LastPostAt: &lastPost},
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/home/activity").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.HomeActivityResponse](t, body)

	require.Len(t, got.RecentActivity, 4)
	assert.Equal(t, "theory", got.RecentActivity[0].Kind)
	assert.Equal(t, "/theory/"+theoryID.String(), got.RecentActivity[0].URL)
	assert.Equal(t, "/game-board/"+postID.String(), got.RecentActivity[1].URL)
	assert.Equal(t, "/journals/"+journalID.String(), got.RecentActivity[2].URL)
	assert.Equal(t, "/gallery/art/"+artID.String(), got.RecentActivity[3].URL)
	assert.Equal(t, authorID, got.RecentActivity[0].Author.ID)

	require.Len(t, got.RecentMembers, 1)
	assert.Equal(t, memberID, got.RecentMembers[0].ID)
	assert.Equal(t, "Newbie", got.RecentMembers[0].DisplayName)

	require.Len(t, got.PublicRooms, 1)
	assert.Equal(t, roomID, got.PublicRooms[0].ID)
	assert.Equal(t, 3, got.PublicRooms[0].MemberCount)
	require.NotNil(t, got.PublicRooms[0].LastMessageAt)
	assert.Equal(t, lastMsg, *got.PublicRooms[0].LastMessageAt)

	require.Len(t, got.CornerActivity, 1)
	assert.Equal(t, "umineko", got.CornerActivity[0].Corner)
	assert.Equal(t, 7, got.CornerActivity[0].PostCount)
	assert.Equal(t, 3, got.CornerActivity[0].UniquePosters)
}

func TestGetHomeActivity_ActivityError(t *testing.T) {
	h, repo := newHomeHarness(t)
	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return(nil, errors.New("boom"))

	status, body := h.NewRequest("GET", "/home/activity").Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load activity")
}

func TestGetHomeActivity_MembersError(t *testing.T) {
	h, repo := newHomeHarness(t)
	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return(nil, errors.New("boom"))

	status, body := h.NewRequest("GET", "/home/activity").Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load members")
}

func TestGetHomeActivity_RoomsError(t *testing.T) {
	h, repo := newHomeHarness(t)
	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{}, nil)
	repo.EXPECT().ListPublicRooms(mock.Anything, 5).Return(nil, errors.New("boom"))

	status, body := h.NewRequest("GET", "/home/activity").Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load public rooms")
}

func TestGetHomeActivity_CornersError(t *testing.T) {
	h, repo := newHomeHarness(t)
	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{}, nil)
	repo.EXPECT().ListPublicRooms(mock.Anything, 5).Return([]repository.HomePublicRoomRow{}, nil)
	repo.EXPECT().ListCornerActivity24h(mock.Anything).Return(nil, errors.New("boom"))

	status, body := h.NewRequest("GET", "/home/activity").Do()

	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load corner activity")
}

func TestActivityURL_UnknownKindFallsBackToRoot(t *testing.T) {
	assert.Equal(t, "/", activityURL("mystery", uuid.New()))
}

func TestGetSidebarLastVisited_OK(t *testing.T) {
	// given
	h, _, sidebarRepo := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	sidebarRepo.EXPECT().ListForUser(mock.Anything, userID).Return(map[string]string{
		"mysteries": "2026-04-24T10:00:00Z",
		"rooms":     "2026-04-24T11:00:00Z",
	}, nil)

	// when
	status, body := h.NewRequest("GET", "/sidebar/last-visited").WithCookie("valid").Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[dto.SidebarLastVisitedResponse](t, body)
	assert.Equal(t, "2026-04-24T10:00:00Z", got.Visited["mysteries"])
	assert.Equal(t, "2026-04-24T11:00:00Z", got.Visited["rooms"])
}

func TestGetSidebarLastVisited_RepoError(t *testing.T) {
	// given
	h, _, sidebarRepo := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	sidebarRepo.EXPECT().ListForUser(mock.Anything, userID).Return(nil, errors.New("db down"))

	// when
	status, body := h.NewRequest("GET", "/sidebar/last-visited").WithCookie("valid").Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to load sidebar last visited")
}

func TestGetSidebarLastVisited_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, func(t *testing.T) (*testutil.Harness, *repository.MockSidebarLastVisitedRepository) {
		h, _, sidebarRepo := newHomeHarnessFull(t)
		return h, sidebarRepo
	}, "GET", "/sidebar/last-visited", nil)
}

func TestMarkSidebarVisited_OK(t *testing.T) {
	// given
	h, _, sidebarRepo := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	sidebarRepo.EXPECT().Upsert(mock.Anything, userID, "mysteries").Return(nil)

	// when
	status, _ := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithJSONBody(dto.MarkSidebarVisitedRequest{Key: "mysteries"}).
		Do()

	// then
	require.Equal(t, http.StatusNoContent, status)
}

func TestMarkSidebarVisited_EmptyKey(t *testing.T) {
	// given
	h, _, _ := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)

	// when
	status, body := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithJSONBody(dto.MarkSidebarVisitedRequest{Key: ""}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "key is required")
}

func TestMarkSidebarVisited_KeyTooLong(t *testing.T) {
	// given
	h, _, _ := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	longKey := make([]byte, 101)
	for i := 0; i < len(longKey); i++ {
		longKey[i] = 'a'
	}

	// when
	status, body := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithJSONBody(dto.MarkSidebarVisitedRequest{Key: string(longKey)}).
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "key too long")
}

func TestMarkSidebarVisited_InvalidBody(t *testing.T) {
	// given
	h, _, _ := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)

	// when
	status, body := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithRawBody("not-json", "application/json").
		Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "invalid request body")
}

func TestMarkSidebarVisited_RepoError(t *testing.T) {
	// given
	h, _, sidebarRepo := newHomeHarnessFull(t)
	userID := uuid.New()
	h.ExpectValidSession("valid", userID)
	sidebarRepo.EXPECT().Upsert(mock.Anything, userID, "mysteries").Return(errors.New("db down"))

	// when
	status, body := h.NewRequest("POST", "/sidebar/last-visited").
		WithCookie("valid").
		WithJSONBody(dto.MarkSidebarVisitedRequest{Key: "mysteries"}).
		Do()

	// then
	require.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, string(body), "failed to mark sidebar visited")
}

func TestMarkSidebarVisited_AuthFailures(t *testing.T) {
	testutil.RunAuthFailureSuite(t, func(t *testing.T) (*testutil.Harness, *repository.MockSidebarLastVisitedRepository) {
		h, _, sidebarRepo := newHomeHarnessFull(t)
		return h, sidebarRepo
	}, "POST", "/sidebar/last-visited", dto.MarkSidebarVisitedRequest{Key: "mysteries"})
}
