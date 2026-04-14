package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createArt(t *testing.T, repos *repository.Repositories, userID uuid.UUID, corner, artType, title string, tags []string, spoiler bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := repos.Art.CreateWithTags(context.Background(), id, userID, corner, artType, title, "desc", "https://example.com/img.png", "https://example.com/thumb.png", tags, spoiler)
	require.NoError(t, err)
	return id
}

func createGallery(t *testing.T, repos *repository.Repositories, userID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Art.CreateGallery(context.Background(), id, userID, name, "desc"))
	return id
}

func createArtComment(t *testing.T, repos *repository.Repositories, artID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Art.CreateComment(context.Background(), id, artID, parentID, userID, body))
	return id
}

func TestArtRepository_CreateWithTags_GetByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Artist"))

	// when
	id := createArt(t, repos, user.ID, "general", "drawing", "My Art", []string{"tagA", "TagB", "  "}, false)

	// then
	row, err := repos.Art.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, id, row.ID)
	assert.Equal(t, user.ID, row.UserID)
	assert.Equal(t, "general", row.Corner)
	assert.Equal(t, "drawing", row.ArtType)
	assert.Equal(t, "My Art", row.Title)
	assert.Equal(t, "Artist", row.AuthorDisplayName)
	assert.Equal(t, 0, row.LikeCount)
	assert.Equal(t, 0, row.CommentCount)
	assert.False(t, row.IsSpoiler)
}

func TestArtRepository_CreateWithTags_SpoilerTrue(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	id := createArt(t, repos, user.ID, "general", "drawing", "S", nil, true)

	// then
	row, err := repos.Art.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsSpoiler)
}

func TestArtRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	row, err := repos.Art.GetByID(context.Background(), uuid.New(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtRepository_GetTags_TrimsAndLowercases(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", []string{"FooBar", " baz ", ""}, false)

	// when
	tags, err := repos.Art.GetTags(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"foobar", "baz"}, tags)
}

func TestArtRepository_UpdateWithTags_Owner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "Old", []string{"a"}, false)

	// when
	err := repos.Art.UpdateWithTags(context.Background(), id, user.ID, "New Title", "New Desc", []string{"b", "c"}, true, false)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Title", row.Title)
	assert.Equal(t, "New Desc", row.Description)
	assert.True(t, row.IsSpoiler)
	tags, err := repos.Art.GetTags(context.Background(), id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"b", "c"}, tags)
}

func TestArtRepository_UpdateWithTags_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.UpdateWithTags(context.Background(), id, other.ID, "Hack", "Hack", nil, false, false)

	// then
	require.Error(t, err)
}

func TestArtRepository_UpdateWithTags_AsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	admin := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.UpdateWithTags(context.Background(), id, admin.ID, "Admin Title", "d", []string{"x"}, false, true)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "Admin Title", row.Title)
}

func TestArtRepository_Delete_Owner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtRepository_Delete_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.Delete(context.Background(), id, other.ID)

	// then
	require.Error(t, err)
}

func TestArtRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtRepository_GetArtAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	authorID, err := repos.Art.GetArtAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, authorID)
}

func TestArtRepository_GetArtAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Art.GetArtAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtRepository_GetImageURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	url, err := repos.Art.GetImageURL(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/img.png", url)
}

func TestArtRepository_GetImageURL_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Art.GetImageURL(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtRepository_LikeAndUnlike(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	require.NoError(t, repos.Art.Like(context.Background(), liker.ID, id))
	require.NoError(t, repos.Art.Like(context.Background(), liker.ID, id))

	// then
	row, err := repos.Art.GetByID(context.Background(), id, liker.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.LikeCount)
	assert.True(t, row.UserLiked)

	require.NoError(t, repos.Art.Unlike(context.Background(), liker.ID, id))
	row, err = repos.Art.GetByID(context.Background(), id, liker.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.LikeCount)
	assert.False(t, row.UserLiked)
}

func TestArtRepository_GetLikedBy(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)
	require.NoError(t, repos.Art.Like(context.Background(), a.ID, id))
	require.NoError(t, repos.Art.Like(context.Background(), b.ID, id))

	// when
	users, err := repos.Art.GetLikedBy(context.Background(), id, nil)

	// then
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestArtRepository_GetLikedBy_ExcludesUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)
	require.NoError(t, repos.Art.Like(context.Background(), a.ID, id))
	require.NoError(t, repos.Art.Like(context.Background(), b.ID, id))

	// when
	users, err := repos.Art.GetLikedBy(context.Background(), id, []uuid.UUID{a.ID})

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, b.ID, users[0].ID)
}

func TestArtRepository_RecordView(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	firstNew, err := repos.Art.RecordView(context.Background(), id, "hash1")
	require.NoError(t, err)
	dupNew, err := repos.Art.RecordView(context.Background(), id, "hash1")
	require.NoError(t, err)
	secondNew, err := repos.Art.RecordView(context.Background(), id, "hash2")
	require.NoError(t, err)

	// then
	assert.True(t, firstNew)
	assert.False(t, dupNew)
	assert.True(t, secondNew)
	row, err := repos.Art.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, row.ViewCount)
}

func TestArtRepository_GetTagsBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id1 := createArt(t, repos, user.ID, "general", "drawing", "A", []string{"x", "y"}, false)
	id2 := createArt(t, repos, user.ID, "general", "drawing", "B", []string{"z"}, false)
	id3 := createArt(t, repos, user.ID, "general", "drawing", "C", nil, false)

	// when
	result, err := repos.Art.GetTagsBatch(context.Background(), []uuid.UUID{id1, id2, id3})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"x", "y"}, result[id1])
	assert.ElementsMatch(t, []string{"z"}, result[id2])
	assert.Empty(t, result[id3])
}

func TestArtRepository_GetTagsBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Art.GetTagsBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArtRepository_GetPopularTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"common", "rare"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", []string{"common"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "C", []string{"common"}, false)

	// when
	tags, err := repos.Art.GetPopularTags(context.Background(), "general", 10)

	// then
	require.NoError(t, err)
	require.NotEmpty(t, tags)
	assert.Equal(t, "common", tags[0].Tag)
	assert.Equal(t, 3, tags[0].Count)
}

func TestArtRepository_GetPopularTags_NoCornerFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"foo"}, false)
	createArt(t, repos, user.ID, "other", "drawing", "B", []string{"foo"}, false)

	// when
	tags, err := repos.Art.GetPopularTags(context.Background(), "", 10)

	// then
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, 2, tags[0].Count)
}

func TestArtRepository_GetCornerCounts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, user.ID, "alt", "drawing", "C", nil, false)

	// when
	counts, err := repos.Art.GetCornerCounts(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, counts["general"])
	assert.Equal(t, 1, counts["alt"])
}

func TestArtRepository_CountUserArtToday(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, other.ID, "general", "drawing", "C", nil, false)

	// when
	count, err := repos.Art.CountUserArtToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestArtRepository_ListAll_Basic(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, user.ID, "alt", "drawing", "C", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), user.ID, "general", "", "", "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, arts, 2)
}

func TestArtRepository_ListAll_FilterByArtType(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "photo", "B", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), user.ID, "general", "photo", "", "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, "B", arts[0].Title)
}

func TestArtRepository_ListAll_Search(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "UniqueApple", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "Banana", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), user.ID, "general", "", "Apple", "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, "UniqueApple", arts[0].Title)
}

func TestArtRepository_ListAll_FilterByTag(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"red"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", []string{"blue"}, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), user.ID, "general", "", "", "red", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, "A", arts[0].Title)
}

func TestArtRepository_ListAll_SortPopular(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	idA := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	idB := createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	require.NoError(t, repos.Art.Like(context.Background(), liker.ID, idB))

	// when
	arts, _, err := repos.Art.ListAll(context.Background(), user.ID, "general", "", "", "", "popular", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, arts, 2)
	assert.Equal(t, idB, arts[0].ID)
	assert.Equal(t, idA, arts[1].ID)
}

func TestArtRepository_ListAll_SortViews(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	idA := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	idB := createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	_, err := repos.Art.RecordView(context.Background(), idB, "h1")
	require.NoError(t, err)
	_, err = repos.Art.RecordView(context.Background(), idB, "h2")
	require.NoError(t, err)
	_, err = repos.Art.RecordView(context.Background(), idA, "h3")
	require.NoError(t, err)

	// when
	arts, _, err := repos.Art.ListAll(context.Background(), user.ID, "general", "", "", "", "views", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, arts, 2)
	assert.Equal(t, idB, arts[0].ID)
	assert.Equal(t, idA, arts[1].ID)
}

func TestArtRepository_ListAll_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 5; i++ {
		createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	}

	// when
	page1, total, err := repos.Art.ListAll(context.Background(), user.ID, "general", "", "", "", "", 2, 0, nil)
	page2, _, err2 := repos.Art.ListAll(context.Background(), user.ID, "general", "", "", "", "", 2, 2, nil)

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
}

func TestArtRepository_ListAll_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	userA := repotest.CreateUser(t, repos)
	userB := repotest.CreateUser(t, repos)
	createArt(t, repos, userA.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, userB.ID, "general", "drawing", "B", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), userA.ID, "general", "", "", "", "", 10, 0, []uuid.UUID{userB.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, userA.ID, arts[0].UserID)
}

func TestArtRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	userA := repotest.CreateUser(t, repos)
	userB := repotest.CreateUser(t, repos)
	createArt(t, repos, userA.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, userA.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, userB.ID, "general", "drawing", "C", nil, false)

	// when
	arts, total, err := repos.Art.ListByUser(context.Background(), userA.ID, userA.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, arts, 2)
}

func TestArtRepository_ListByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)
	}

	// when
	arts, total, err := repos.Art.ListByUser(context.Background(), user.ID, user.ID, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, arts, 2)
}

func TestArtRepository_CreateComment_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)

	// when
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hello")

	// then
	comments, total, err := repos.Art.GetComments(context.Background(), artID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Equal(t, "hello", comments[0].Body)
	assert.Nil(t, comments[0].ParentID)
}

func TestArtRepository_CreateComment_Threaded(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	parent := createArtComment(t, repos, artID, user.ID, nil, "parent")

	// when
	child := createArtComment(t, repos, artID, user.ID, &parent, "child")

	// then
	comments, total, err := repos.Art.GetComments(context.Background(), artID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	var childRow *struct{ parentID *uuid.UUID }
	for _, c := range comments {
		if c.ID == child {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parent, *c.ParentID)
			childRow = &struct{ parentID *uuid.UUID }{parentID: c.ParentID}
		}
	}
	assert.NotNil(t, childRow)
}

func TestArtRepository_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	createArtComment(t, repos, artID, owner.ID, nil, "ok")
	createArtComment(t, repos, artID, other.ID, nil, "hide")

	// when
	comments, total, err := repos.Art.GetComments(context.Background(), artID, owner.ID, 10, 0, []uuid.UUID{other.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, comments, 1)
	assert.Equal(t, "ok", comments[0].Body)
}

func TestArtRepository_UpdateComment_Owner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "old")

	// when
	err := repos.Art.UpdateComment(context.Background(), commentID, user.ID, "new body")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Art.GetComments(context.Background(), artID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new body", comments[0].Body)
}

func TestArtRepository_UpdateComment_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "old")

	// when
	err := repos.Art.UpdateComment(context.Background(), commentID, other.ID, "hack")

	// then
	require.Error(t, err)
}

func TestArtRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "old")

	// when
	err := repos.Art.UpdateCommentAsAdmin(context.Background(), commentID, "admin body")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Art.GetComments(context.Background(), artID, owner.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin body", comments[0].Body)
}

func TestArtRepository_UpdateCommentAsAdmin_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Art.UpdateCommentAsAdmin(context.Background(), uuid.New(), "body")

	// then
	require.Error(t, err)
}

func TestArtRepository_DeleteComment_Owner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	err := repos.Art.DeleteComment(context.Background(), commentID, user.ID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), artID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestArtRepository_DeleteComment_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "hi")

	// when
	err := repos.Art.DeleteComment(context.Background(), commentID, other.ID)

	// then
	require.Error(t, err)
}

func TestArtRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	err := repos.Art.DeleteCommentAsAdmin(context.Background(), commentID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), artID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestArtRepository_GetCommentArtID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	got, err := repos.Art.GetCommentArtID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, artID, got)
}

func TestArtRepository_GetCommentArtID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Art.GetCommentArtID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtRepository_GetCommentAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	got, err := repos.Art.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestArtRepository_GetCommentAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Art.GetCommentAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtRepository_LikeAndUnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	require.NoError(t, repos.Art.LikeComment(context.Background(), liker.ID, commentID))
	require.NoError(t, repos.Art.LikeComment(context.Background(), liker.ID, commentID))

	// then
	comments, _, err := repos.Art.GetComments(context.Background(), artID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)

	require.NoError(t, repos.Art.UnlikeComment(context.Background(), liker.ID, commentID))
	comments, _, err = repos.Art.GetComments(context.Background(), artID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].LikeCount)
	assert.False(t, comments[0].UserLiked)
}

func TestArtRepository_AddCommentMedia_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	mediaID, err := repos.Art.AddCommentMedia(context.Background(), commentID, "https://example.com/m.png", "image", "https://example.com/m-thumb.png", 0)

	// then
	require.NoError(t, err)
	require.Greater(t, mediaID, int64(0))
	media, err := repos.Art.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "https://example.com/m.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
}

func TestArtRepository_GetCommentMedia_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	media, err := repos.Art.GetCommentMedia(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Empty(t, media)
}

func TestArtRepository_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	c1 := createArtComment(t, repos, artID, user.ID, nil, "one")
	c2 := createArtComment(t, repos, artID, user.ID, nil, "two")
	_, err := repos.Art.AddCommentMedia(context.Background(), c1, "u1", "image", "t1", 0)
	require.NoError(t, err)
	_, err = repos.Art.AddCommentMedia(context.Background(), c1, "u2", "image", "t2", 1)
	require.NoError(t, err)
	_, err = repos.Art.AddCommentMedia(context.Background(), c2, "u3", "video", "t3", 0)
	require.NoError(t, err)

	// when
	result, err := repos.Art.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, result[c1], 2)
	assert.Len(t, result[c2], 1)
}

func TestArtRepository_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Art.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArtRepository_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")
	mediaID, err := repos.Art.AddCommentMedia(context.Background(), commentID, "old", "image", "t", 0)
	require.NoError(t, err)

	// when
	err = repos.Art.UpdateCommentMediaURL(context.Background(), mediaID, "new")

	// then
	require.NoError(t, err)
	media, err := repos.Art.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "new", media[0].MediaURL)
}

func TestArtRepository_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")
	mediaID, err := repos.Art.AddCommentMedia(context.Background(), commentID, "u", "image", "old", 0)
	require.NoError(t, err)

	// when
	err = repos.Art.UpdateCommentMediaThumbnail(context.Background(), mediaID, "new")

	// then
	require.NoError(t, err)
	media, err := repos.Art.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "new", media[0].ThumbnailURL)
}

func TestArtRepository_CreateGallery_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("GalleryOwner"))

	// when
	id := createGallery(t, repos, user.ID, "My Gallery")

	// then
	row, err := repos.Art.GetGalleryByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "My Gallery", row.Name)
	assert.Equal(t, "desc", row.Description)
	assert.Equal(t, "GalleryOwner", row.AuthorDisplayName)
	assert.Equal(t, 0, row.ArtCount)
}

func TestArtRepository_GetGalleryByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	row, err := repos.Art.GetGalleryByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtRepository_UpdateGallery_Owner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createGallery(t, repos, user.ID, "Old")

	// when
	err := repos.Art.UpdateGallery(context.Background(), id, user.ID, "New Name", "New Desc")

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "New Name", row.Name)
	assert.Equal(t, "New Desc", row.Description)
}

func TestArtRepository_UpdateGallery_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createGallery(t, repos, owner.ID, "Old")

	// when
	err := repos.Art.UpdateGallery(context.Background(), id, other.ID, "Hack", "")

	// then
	require.Error(t, err)
}

func TestArtRepository_SetGalleryCover(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))

	// when
	err := repos.Art.SetGalleryCover(context.Background(), galleryID, user.ID, &artID)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.NotNil(t, row.CoverArtID)
	assert.Equal(t, artID, *row.CoverArtID)
	assert.Equal(t, "https://example.com/img.png", row.CoverImageURL)
}

func TestArtRepository_SetGalleryCover_Clear(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))
	require.NoError(t, repos.Art.SetGalleryCover(context.Background(), galleryID, user.ID, &artID))

	// when
	err := repos.Art.SetGalleryCover(context.Background(), galleryID, user.ID, nil)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Nil(t, row.CoverArtID)
}

func TestArtRepository_SetGalleryCover_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, owner.ID, "G")

	// when
	err := repos.Art.SetGalleryCover(context.Background(), galleryID, other.ID, nil)

	// then
	require.Error(t, err)
}

func TestArtRepository_SetGallery_AndClear(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)

	// when
	require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))

	// then
	row, err := repos.Art.GetByID(context.Background(), artID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row.GalleryID)
	assert.Equal(t, galleryID, *row.GalleryID)

	require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, nil))
	row, err = repos.Art.GetByID(context.Background(), artID, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row.GalleryID)
}

func TestArtRepository_SetGallery_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, owner.ID, "G")
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)

	// when
	err := repos.Art.SetGallery(context.Background(), artID, other.ID, &galleryID)

	// then
	require.Error(t, err)
}

func TestArtRepository_DeleteGallery_Owner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))

	// when
	err := repos.Art.DeleteGallery(context.Background(), galleryID, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	assert.Nil(t, row)
	art, err := repos.Art.GetByID(context.Background(), artID, user.ID)
	require.NoError(t, err)
	assert.Nil(t, art)
}

func TestArtRepository_DeleteGallery_NotOwner_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, owner.ID, "G")

	// when
	err := repos.Art.DeleteGallery(context.Background(), galleryID, other.ID)

	// then
	require.Error(t, err)
}

func TestArtRepository_ListGalleriesByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	userA := repotest.CreateUser(t, repos)
	userB := repotest.CreateUser(t, repos)
	createGallery(t, repos, userA.ID, "A1")
	createGallery(t, repos, userA.ID, "A2")
	createGallery(t, repos, userB.ID, "B1")

	// when
	galleries, err := repos.Art.ListGalleriesByUser(context.Background(), userA.ID)

	// then
	require.NoError(t, err)
	assert.Len(t, galleries, 2)
}

func TestArtRepository_ListAllGalleries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createGallery(t, repos, user.ID, "G1")
	createGallery(t, repos, user.ID, "G2")

	// when
	galleries, err := repos.Art.ListAllGalleries(context.Background(), "")

	// then
	require.NoError(t, err)
	assert.Len(t, galleries, 2)
}

func TestArtRepository_ListAllGalleries_FilterByCorner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	g1 := createGallery(t, repos, user.ID, "G1")
	g2 := createGallery(t, repos, user.ID, "G2")
	a1 := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	a2 := createArt(t, repos, user.ID, "alt", "drawing", "B", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), a1, user.ID, &g1))
	require.NoError(t, repos.Art.SetGallery(context.Background(), a2, user.ID, &g2))

	// when
	galleries, err := repos.Art.ListAllGalleries(context.Background(), "general")

	// then
	require.NoError(t, err)
	require.Len(t, galleries, 1)
	assert.Equal(t, g1, galleries[0].ID)
}

func TestArtRepository_GetGalleryPreviewImages(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	for i := 0; i < 3; i++ {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))
	}

	// when
	imgs, err := repos.Art.GetGalleryPreviewImages(context.Background(), galleryID, 2)

	// then
	require.NoError(t, err)
	assert.Len(t, imgs, 2)
	assert.Equal(t, "https://example.com/img.png", imgs[0].ImageURL)
	assert.Equal(t, "https://example.com/thumb.png", imgs[0].ThumbnailURL)
}

func TestArtRepository_GetGalleryPreviewImages_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	imgs, err := repos.Art.GetGalleryPreviewImages(context.Background(), uuid.New(), 5)

	// then
	require.NoError(t, err)
	assert.Empty(t, imgs)
}

func TestArtRepository_ListArtInGallery(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	otherGallery := createGallery(t, repos, user.ID, "H")
	for i := 0; i < 3; i++ {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))
	}
	otherArt := createArt(t, repos, user.ID, "general", "drawing", "X", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), otherArt, user.ID, &otherGallery))

	// when
	arts, total, err := repos.Art.ListArtInGallery(context.Background(), galleryID, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, arts, 3)
}

func TestArtRepository_ListArtInGallery_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	for i := 0; i < 4; i++ {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(context.Background(), artID, user.ID, &galleryID))
	}

	// when
	arts, total, err := repos.Art.ListArtInGallery(context.Background(), galleryID, user.ID, 2, 1)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, arts, 2)
}

func TestArtRepository_GetComments_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	for i := 0; i < 5; i++ {
		createArtComment(t, repos, artID, user.ID, nil, "c")
	}

	// when
	comments, total, err := repos.Art.GetComments(context.Background(), artID, user.ID, 2, 2, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, comments, 2)
}

func TestArtRepository_CreateComment_UnknownArt_Fails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Art.CreateComment(context.Background(), uuid.New(), uuid.New(), nil, user.ID, "body")

	// then
	require.Error(t, err)
}

func TestArtRepository_DeleteComment_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Art.DeleteComment(context.Background(), uuid.New(), user.ID)

	// then
	require.Error(t, err)
}
