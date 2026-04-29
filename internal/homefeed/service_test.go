package homefeed_test

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_HomeActivity_ComposesAllSections(t *testing.T) {
	// given
	repo := repository.NewMockHomeFeedRepository(t)
	hub := ws.NewHub()
	svc := homefeed.NewService(repo, hub)

	theoryID := uuid.New()
	postID := uuid.New()
	journalID := uuid.New()
	artID := uuid.New()
	authorID := uuid.New()
	memberID := uuid.New()
	roomID := uuid.New()
	lastMsg := "2026-04-24T12:00:00Z"
	lastPost := "2026-04-24T11:00:00Z"

	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{
		{Kind: "theory", ID: theoryID, Title: "T", Body: "x", Corner: "umineko", AuthorID: authorID, Username: "u", DisplayName: "U"},
		{Kind: "post", ID: postID, Title: "", Body: "p", Corner: "general", AuthorID: authorID, Username: "u", DisplayName: "U"},
		{Kind: "journal", ID: journalID, Title: "J", Body: "j", AuthorID: authorID},
		{Kind: "art", ID: artID, Title: "A", AuthorID: authorID},
	}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{
		{ID: memberID, Username: "newbie", DisplayName: "Newbie"},
	}, nil)
	repo.EXPECT().ListPublicRooms(mock.Anything, 5).Return([]repository.HomePublicRoomRow{
		{ID: roomID, Name: "Hangout", Description: "d", MemberCount: 3, LastMessageAt: &lastMsg},
	}, nil)
	repo.EXPECT().ListCornerActivity24h(mock.Anything).Return([]repository.HomeCornerActivityRow{
		{Corner: "umineko", PostCount: 7, UniquePosters: 3, LastPostAt: &lastPost},
	}, nil)

	// when
	resp, err := svc.HomeActivity(context.Background())

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.RecentActivity, 4)
	assert.Equal(t, "/theory/"+theoryID.String(), resp.RecentActivity[0].URL)
	assert.Equal(t, "/game-board/"+postID.String(), resp.RecentActivity[1].URL)
	assert.Equal(t, "/journals/"+journalID.String(), resp.RecentActivity[2].URL)
	assert.Equal(t, "/gallery/art/"+artID.String(), resp.RecentActivity[3].URL)
	assert.Equal(t, authorID, resp.RecentActivity[0].Author.ID)

	require.Len(t, resp.RecentMembers, 1)
	assert.Equal(t, memberID, resp.RecentMembers[0].ID)

	require.Len(t, resp.PublicRooms, 1)
	assert.Equal(t, roomID, resp.PublicRooms[0].ID)
	assert.Equal(t, 3, resp.PublicRooms[0].MemberCount)

	require.Len(t, resp.CornerActivity, 1)
	assert.Equal(t, "umineko", resp.CornerActivity[0].Corner)
}

func TestService_HomeActivity_UnknownKindFallsBackToRoot(t *testing.T) {
	// given
	repo := repository.NewMockHomeFeedRepository(t)
	svc := homefeed.NewService(repo, ws.NewHub())
	repo.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{
		{Kind: "mystery", ID: uuid.New()},
	}, nil)
	repo.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{}, nil)
	repo.EXPECT().ListPublicRooms(mock.Anything, 5).Return([]repository.HomePublicRoomRow{}, nil)
	repo.EXPECT().ListCornerActivity24h(mock.Anything).Return([]repository.HomeCornerActivityRow{}, nil)

	// when
	resp, err := svc.HomeActivity(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, resp.RecentActivity, 1)
	assert.Equal(t, "/", resp.RecentActivity[0].URL)
}

func TestService_HomeActivity_PropagatesErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*repository.MockHomeFeedRepository)
	}{
		{
			"activity error",
			func(r *repository.MockHomeFeedRepository) {
				r.EXPECT().ListRecentActivity(mock.Anything, 10).Return(nil, errors.New("boom"))
			},
		},
		{
			"members error",
			func(r *repository.MockHomeFeedRepository) {
				r.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
				r.EXPECT().ListRecentMembers(mock.Anything, 5).Return(nil, errors.New("boom"))
			},
		},
		{
			"rooms error",
			func(r *repository.MockHomeFeedRepository) {
				r.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
				r.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{}, nil)
				r.EXPECT().ListPublicRooms(mock.Anything, 5).Return(nil, errors.New("boom"))
			},
		},
		{
			"corners error",
			func(r *repository.MockHomeFeedRepository) {
				r.EXPECT().ListRecentActivity(mock.Anything, 10).Return([]repository.HomeActivityRow{}, nil)
				r.EXPECT().ListRecentMembers(mock.Anything, 5).Return([]repository.HomeMemberRow{}, nil)
				r.EXPECT().ListPublicRooms(mock.Anything, 5).Return([]repository.HomePublicRoomRow{}, nil)
				r.EXPECT().ListCornerActivity24h(mock.Anything).Return(nil, errors.New("boom"))
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo := repository.NewMockHomeFeedRepository(t)
			svc := homefeed.NewService(repo, ws.NewHub())
			tc.setup(repo)

			// when
			_, err := svc.HomeActivity(context.Background())

			// then
			assert.Error(t, err)
		})
	}
}

func TestService_SidebarActivity_BuildsMap(t *testing.T) {
	// given
	repo := repository.NewMockHomeFeedRepository(t)
	svc := homefeed.NewService(repo, ws.NewHub())
	repo.EXPECT().ListSidebarActivity(mock.Anything).Return([]repository.SidebarActivityEntry{
		{Key: "rooms", LatestAt: "2026-04-24T09:00:00Z"},
		{Key: "mysteries", LatestAt: "2026-04-24T08:00:00Z"},
	}, nil)

	// when
	resp, err := svc.SidebarActivity(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, "2026-04-24T09:00:00Z", resp.Activity["rooms"])
	assert.Equal(t, "2026-04-24T08:00:00Z", resp.Activity["mysteries"])
}

func TestService_SidebarActivity_PropagatesError(t *testing.T) {
	// given
	repo := repository.NewMockHomeFeedRepository(t)
	svc := homefeed.NewService(repo, ws.NewHub())
	repo.EXPECT().ListSidebarActivity(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.SidebarActivity(context.Background())

	// then
	assert.Error(t, err)
}
