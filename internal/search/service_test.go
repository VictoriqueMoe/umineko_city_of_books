package search_test

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_Search_DelegatesAndDecoratesURLs(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)
	parentID := "parent-id"
	repo.EXPECT().
		Search(mock.Anything, "battler", []repository.SearchEntityType{repository.SearchEntityTheory}, 20, 0).
		Return([]repository.SearchResult{
			{EntityType: repository.SearchEntityTheory, ID: "t1"},
			{EntityType: repository.SearchEntityPostComment, ID: "c1", ParentID: &parentID},
		}, 2, nil)

	// when
	results, total, err := svc.Search(context.Background(), "battler",
		[]repository.SearchEntityType{repository.SearchEntityTheory}, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, results, 2)
	assert.Equal(t, "/theory/t1", results[0].URL)
	assert.Equal(t, "/game-board/parent-id#comment-c1", results[1].URL)
}

func TestService_Search_EmptyQuery_NoRepoCall(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)

	// when
	results, total, err := svc.Search(context.Background(), "  ", nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, 0, total)
}

func TestService_Search_ClampsLimit(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)
	repo.EXPECT().Search(mock.Anything, "x", mock.Anything, 100, 0).Return(nil, 0, nil)

	// when
	_, _, err := svc.Search(context.Background(), "x", nil, 9999, 0)

	// then
	require.NoError(t, err)
}

func TestService_Search_AppliesDefaults(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)
	repo.EXPECT().Search(mock.Anything, "x", mock.Anything, 20, 0).Return(nil, 0, nil)

	// when
	_, _, err := svc.Search(context.Background(), "x", nil, 0, -5)

	// then
	require.NoError(t, err)
}

func TestService_Search_PropagatesError(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)
	repo.EXPECT().Search(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, 0, errors.New("boom"))

	// when
	_, _, err := svc.Search(context.Background(), "x", nil, 20, 0)

	// then
	assert.Error(t, err)
}

func TestService_QuickSearch_DelegatesAndDecoratesURL(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)
	repo.EXPECT().QuickSearch(mock.Anything, "x", 3).Return([]repository.SearchResult{
		{EntityType: repository.SearchEntityMystery, ID: "m1"},
	}, nil)

	// when
	results, err := svc.QuickSearch(context.Background(), "x", 3)

	// then
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/mystery/m1", results[0].URL)
}

func TestService_QuickSearch_EmptyQuery_NoRepoCall(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)

	// when
	results, err := svc.QuickSearch(context.Background(), " ", 3)

	// then
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestService_QuickSearch_ClampsPerTypeLimit(t *testing.T) {
	// given
	repo := repository.NewMockSearchRepository(t)
	svc := search.NewService(repo)
	repo.EXPECT().QuickSearch(mock.Anything, "x", 10).Return(nil, nil)

	// when
	_, err := svc.QuickSearch(context.Background(), "x", 999)

	// then
	require.NoError(t, err)
}

func TestService_ParseTypes_AllReturnsNil(t *testing.T) {
	// given
	svc := search.NewService(repository.NewMockSearchRepository(t))

	// when / then
	assert.Nil(t, svc.ParseTypes(""))
	assert.Nil(t, svc.ParseTypes("all"))
}

func TestService_ParseTypes_CommaList(t *testing.T) {
	// given
	svc := search.NewService(repository.NewMockSearchRepository(t))

	// when
	got := svc.ParseTypes("theory, post,art")

	// then
	assert.Equal(t, []repository.SearchEntityType{
		repository.SearchEntityTheory,
		repository.SearchEntityPost,
		repository.SearchEntityArt,
	}, got)
}

func TestService_ParseTypes_CommentsAlias_ExpandsToAllChildren(t *testing.T) {
	// given
	svc := search.NewService(repository.NewMockSearchRepository(t))

	// when
	got := svc.ParseTypes("comments")

	// then
	assert.NotEmpty(t, got)
	for _, typ := range got {
		src, ok := repository.SearchSourceFor(typ)
		require.True(t, ok)
		assert.NotEmptyf(t, src.ParentIDExpr, "%s should be a child entity", typ)
	}
}

func TestService_ParseTypes_MixedSingleAndAlias(t *testing.T) {
	// given
	svc := search.NewService(repository.NewMockSearchRepository(t))

	// when
	got := svc.ParseTypes("theory,comments,user")

	// then
	assert.Contains(t, got, repository.SearchEntityTheory)
	assert.Contains(t, got, repository.SearchEntityUser)
	assert.Contains(t, got, repository.SearchEntityPostComment)
}

func TestService_ChildEntityTypes(t *testing.T) {
	// given
	svc := search.NewService(repository.NewMockSearchRepository(t))

	// when
	children := svc.ChildEntityTypes()

	// then
	assert.NotContains(t, children, repository.SearchEntityTheory)
	assert.NotContains(t, children, repository.SearchEntityUser)
	assert.Contains(t, children, repository.SearchEntityPostComment)
}

func TestBuildURL(t *testing.T) {
	// given
	parentID := "p1"
	cases := []struct {
		name string
		r    repository.SearchResult
		want string
	}{
		{"theory", repository.SearchResult{EntityType: repository.SearchEntityTheory, ID: "t1"}, "/theory/t1"},
		{"post", repository.SearchResult{EntityType: repository.SearchEntityPost, ID: "p1"}, "/game-board/p1"},
		{"post comment", repository.SearchResult{EntityType: repository.SearchEntityPostComment, ID: "c1", ParentID: &parentID}, "/game-board/p1#comment-c1"},
		{"user", repository.SearchResult{EntityType: repository.SearchEntityUser, AuthorUsername: "beato"}, "/user/beato"},
		{"unknown", repository.SearchResult{EntityType: "nonsense"}, ""},
		{"comment without parent", repository.SearchResult{EntityType: repository.SearchEntityPostComment, ID: "c1", ParentID: nil}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when / then
			assert.Equal(t, tc.want, search.BuildURL(tc.r))
		})
	}
}
