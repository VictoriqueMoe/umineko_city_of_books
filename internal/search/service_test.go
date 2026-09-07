package search_test

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/search"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	viewer = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	room   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

func newSvc(t *testing.T) (search.Service, *repository.MockSearchRepository, *repository.MockChatRepository) {
	repo := repository.NewMockSearchRepository(t)
	chat := repository.NewMockChatRepository(t)
	return search.NewService(repo, chat), repo, chat
}

func TestService_Search_DelegatesAndDecoratesURLs(t *testing.T) {
	// given
	svc, repo, _ := newSvc(t)
	repo.EXPECT().
		Search(mock.Anything, spec.SearchQuery{Query: "battler", Types: []model.SearchEntityType{model.SearchEntityTheory}, Limit: 20, Offset: 0}).
		Return([]model.SearchResult{
			{EntityType: model.SearchEntityTheory, ID: "t1"},
			{EntityType: model.SearchEntityPostComment, ID: "c1", ParentID: new("parent-id")},
		}, 2, nil)

	// when
	results, total, err := svc.Search(context.Background(), "battler",
		[]model.SearchEntityType{model.SearchEntityTheory}, bounds.NewPage(20, 0), uuid.Nil, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, results, 2)
	assert.Equal(t, "/theory/t1", results[0].URL)
	assert.Equal(t, "/game-board/parent-id#comment-c1", results[1].URL)
}

func TestService_Search_EmptyQuery_NoRepoCall(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	results, total, err := svc.Search(context.Background(), "  ", nil, bounds.NewPage(20, 0), uuid.Nil, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, 0, total)
}

func TestService_Search_ClampsLimit(t *testing.T) {
	// given
	svc, repo, _ := newSvc(t)
	repo.EXPECT().Search(mock.Anything, spec.SearchQuery{Query: "x", Types: nil, Limit: 100, Offset: 0}).Return(nil, 0, nil)

	// when
	_, _, err := svc.Search(context.Background(), "x", nil, bounds.NewPage(9999, 0), uuid.Nil, uuid.Nil)

	// then
	require.NoError(t, err)
}

func TestService_Search_AppliesDefaults(t *testing.T) {
	// given
	svc, repo, _ := newSvc(t)
	repo.EXPECT().Search(mock.Anything, spec.SearchQuery{Query: "x", Types: nil, Limit: 20, Offset: 0}).Return(nil, 0, nil)

	// when
	_, _, err := svc.Search(context.Background(), "x", nil, bounds.NewPage(0, -5), uuid.Nil, uuid.Nil)

	// then
	require.NoError(t, err)
}

func TestService_Search_PropagatesError(t *testing.T) {
	// given
	svc, repo, _ := newSvc(t)
	repo.EXPECT().Search(mock.Anything, mock.Anything).
		Return(nil, 0, errors.New("boom"))

	// when
	_, _, err := svc.Search(context.Background(), "x", nil, bounds.NewPage(20, 0), uuid.Nil, uuid.Nil)

	// then
	assert.Error(t, err)
}

func TestService_Search_MergesChatForViewerSortedByRank(t *testing.T) {
	// given
	svc, repo, chat := newSvc(t)
	repo.EXPECT().Search(mock.Anything, spec.SearchQuery{Query: "knox", Types: nil, Limit: 20, Offset: 0}).
		Return([]model.SearchResult{
			{EntityType: model.SearchEntityTheory, ID: "t1", Rank: 0.4},
		}, 1, nil)
	chat.EXPECT().SearchMessagesForViewer(mock.Anything, spec.ChatMessageSearch{ViewerID: viewer, RoomID: uuid.Nil, Query: "knox", Limit: 20, Offset: 0}).
		Return([]model.SearchResult{
			{EntityType: model.SearchEntityChatMessage, ID: "m1", ParentID: new("room1"), Rank: 0.9},
		}, 1, nil)

	// when
	results, total, err := svc.Search(context.Background(), "knox", nil, bounds.NewPage(20, 0), viewer, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, results, 2)
	assert.Equal(t, "/rooms/room1#msg-m1", results[0].URL)
	assert.Equal(t, "/theory/t1", results[1].URL)
}

func TestService_Search_OnlyChatType_SkipsRepo(t *testing.T) {
	// given
	svc, _, chat := newSvc(t)
	chat.EXPECT().SearchMessagesForViewer(mock.Anything, spec.ChatMessageSearch{ViewerID: viewer, RoomID: uuid.Nil, Query: "knox", Limit: 20, Offset: 0}).
		Return([]model.SearchResult{
			{EntityType: model.SearchEntityChatMessage, ID: "m1", ParentID: new("room1")},
		}, 1, nil)

	// when
	results, total, err := svc.Search(context.Background(), "knox",
		[]model.SearchEntityType{model.SearchEntityChatMessage}, bounds.NewPage(20, 0), viewer, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "/rooms/room1#msg-m1", results[0].URL)
}

func TestService_Search_ScopesChatToRoom(t *testing.T) {
	// given
	svc, _, chat := newSvc(t)
	chat.EXPECT().SearchMessagesForViewer(mock.Anything, spec.ChatMessageSearch{ViewerID: viewer, RoomID: room, Query: "knox", Limit: 20, Offset: 0}).
		Return([]model.SearchResult{
			{EntityType: model.SearchEntityChatMessage, ID: "m1", ParentID: new("room1")},
		}, 1, nil)

	// when
	results, total, err := svc.Search(context.Background(), "knox",
		[]model.SearchEntityType{model.SearchEntityChatMessage}, bounds.NewPage(20, 0), viewer, room)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "/rooms/room1#msg-m1", results[0].URL)
}

func TestService_Search_ChatTypeAnonymous_ReturnsNothing(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	results, total, err := svc.Search(context.Background(), "knox",
		[]model.SearchEntityType{model.SearchEntityChatMessage}, bounds.NewPage(20, 0), uuid.Nil, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, 0, total)
}

func TestService_QuickSearch_DelegatesAndDecoratesURL(t *testing.T) {
	// given
	svc, repo, _ := newSvc(t)
	repo.EXPECT().QuickSearch(mock.Anything, spec.QuickSearchQuery{Query: "x", PerTypeLimit: 3}).Return([]model.SearchResult{
		{EntityType: model.SearchEntityMystery, ID: "m1"},
	}, nil)

	// when
	results, err := svc.QuickSearch(context.Background(), "x", 3, uuid.Nil)

	// then
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/mystery/m1", results[0].URL)
}

func TestService_QuickSearch_MergesChatForViewer(t *testing.T) {
	// given
	svc, repo, chat := newSvc(t)
	repo.EXPECT().QuickSearch(mock.Anything, spec.QuickSearchQuery{Query: "x", PerTypeLimit: 3}).Return([]model.SearchResult{
		{EntityType: model.SearchEntityMystery, ID: "m1", Rank: 0.2},
	}, nil)
	chat.EXPECT().SearchMessagesForViewer(mock.Anything, spec.ChatMessageSearch{ViewerID: viewer, RoomID: uuid.Nil, Query: "x", Limit: 3, Offset: 0}).Return([]model.SearchResult{
		{EntityType: model.SearchEntityChatMessage, ID: "c1", ParentID: new("room1"), Rank: 0.8},
	}, 1, nil)

	// when
	results, err := svc.QuickSearch(context.Background(), "x", 3, viewer)

	// then
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "/rooms/room1#msg-c1", results[0].URL)
}

func TestService_QuickSearch_EmptyQuery_NoRepoCall(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	results, err := svc.QuickSearch(context.Background(), " ", 3, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestService_QuickSearch_ClampsPerTypeLimit(t *testing.T) {
	// given
	svc, repo, _ := newSvc(t)
	repo.EXPECT().QuickSearch(mock.Anything, spec.QuickSearchQuery{Query: "x", PerTypeLimit: 10}).Return(nil, nil)

	// when
	_, err := svc.QuickSearch(context.Background(), "x", 999, uuid.Nil)

	// then
	require.NoError(t, err)
}

func TestService_ParseTypes_AllReturnsNil(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when / then
	assert.Nil(t, svc.ParseTypes(""))
	assert.Nil(t, svc.ParseTypes("all"))
}

func TestService_ParseTypes_CommaList(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	got := svc.ParseTypes("theory, post,art")

	// then
	assert.Equal(t, []model.SearchEntityType{
		model.SearchEntityTheory,
		model.SearchEntityPost,
		model.SearchEntityArt,
	}, got)
}

func TestService_ParseTypes_CommentsAlias_ExpandsToAllChildren(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	got := svc.ParseTypes("comments")

	// then
	assert.NotEmpty(t, got)
	for _, typ := range got {
		src, ok := model.SearchSourceFor(typ)
		require.True(t, ok)
		assert.NotEmptyf(t, src.ParentIDExpr, "%s should be a child entity", typ)
	}
}

func TestService_ParseTypes_MixedSingleAndAlias(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	got := svc.ParseTypes("theory,comments,user")

	// then
	assert.Contains(t, got, model.SearchEntityTheory)
	assert.Contains(t, got, model.SearchEntityUser)
	assert.Contains(t, got, model.SearchEntityPostComment)
}

func TestService_ChildEntityTypes(t *testing.T) {
	// given
	svc, _, _ := newSvc(t)

	// when
	children := svc.ChildEntityTypes()

	// then
	assert.NotContains(t, children, model.SearchEntityTheory)
	assert.NotContains(t, children, model.SearchEntityUser)
	assert.Contains(t, children, model.SearchEntityPostComment)
}

func TestBuildURL(t *testing.T) {
	// given
	cases := []struct {
		name string
		r    model.SearchResult
		want string
	}{
		{"theory", model.SearchResult{EntityType: model.SearchEntityTheory, ID: "t1"}, "/theory/t1"},
		{"post", model.SearchResult{EntityType: model.SearchEntityPost, ID: "p1"}, "/game-board/p1"},
		{"post comment", model.SearchResult{EntityType: model.SearchEntityPostComment, ID: "c1", ParentID: new("p1")}, "/game-board/p1#comment-c1"},
		{"user", model.SearchResult{EntityType: model.SearchEntityUser, AuthorUsername: "beato"}, "/user/beato"},
		{"chat message", model.SearchResult{EntityType: model.SearchEntityChatMessage, ID: "m1", ParentID: new("room1")}, "/rooms/room1#msg-m1"},
		{"chat message with timestamp", model.SearchResult{EntityType: model.SearchEntityChatMessage, ID: "m1", ParentID: new("room1"), CreatedAt: "2026-05-29T19:00:00.5Z"}, "/rooms/room1?at=2026-05-29T19%3A00%3A00.5Z#msg-m1"},
		{"chat message without room", model.SearchResult{EntityType: model.SearchEntityChatMessage, ID: "m1", ParentID: nil}, ""},
		{"unknown", model.SearchResult{EntityType: "nonsense"}, ""},
		{"comment without parent", model.SearchResult{EntityType: model.SearchEntityPostComment, ID: "c1", ParentID: nil}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when / then
			assert.Equal(t, tc.want, search.BuildURL(tc.r))
		})
	}
}
