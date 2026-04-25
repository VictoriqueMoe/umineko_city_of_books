package repository_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createPost(t *testing.T, repos *repository.Repositories, userID uuid.UUID, corner, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Post.Create(context.Background(), id, userID, corner, body, nil, nil))
	return id
}

func createComment(t *testing.T, repos *repository.Repositories, postID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Post.CreateComment(context.Background(), id, postID, parentID, userID, body))
	return id
}

func strPtr(s string) *string {
	return &s
}

func TestPostRepository_CreateAndGetByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Poster"))

	// when
	id := createPost(t, repos, user.ID, "general", "hello world")

	// then
	row, err := repos.Post.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "hello world", row.Body)
	assert.Equal(t, "general", row.Corner)
	assert.Equal(t, user.ID, row.UserID)
	assert.Equal(t, "Poster", row.AuthorDisplayName)
}

func TestPostRepository_Create_WithSharedContent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.Post.Create(context.Background(), id, user.ID, "general", "shared", strPtr("abc123"), strPtr("theory"))

	// then
	require.NoError(t, err)
	cid, ctype, err := repos.Post.GetSharedContentFields(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, cid)
	require.NotNil(t, ctype)
	assert.Equal(t, "abc123", *cid)
	assert.Equal(t, "theory", *ctype)
}

func TestPostRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	row, err := repos.Post.GetByID(context.Background(), uuid.New(), user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestPostRepository_UpdatePost(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createPost(t, repos, user.ID, "general", "original")

	// when
	err := repos.Post.UpdatePost(context.Background(), id, user.ID, "updated")

	// then
	require.NoError(t, err)
	row, err := repos.Post.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated", row.Body)
}

func TestPostRepository_UpdatePost_NotOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createPost(t, repos, owner.ID, "general", "body")

	// when
	err := repos.Post.UpdatePost(context.Background(), id, other.ID, "hacked")

	// then
	require.Error(t, err)
}

func TestPostRepository_UpdatePostAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createPost(t, repos, owner.ID, "general", "body")

	// when
	err := repos.Post.UpdatePostAsAdmin(context.Background(), id, "admin edited")

	// then
	require.NoError(t, err)
	row, err := repos.Post.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin edited", row.Body)
}

func TestPostRepository_UpdatePostAsAdmin_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Post.UpdatePostAsAdmin(context.Background(), uuid.New(), "x")

	// then
	require.Error(t, err)
}

func TestPostRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createPost(t, repos, user.ID, "general", "body")

	// when
	err := repos.Post.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Post.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestPostRepository_Delete_NotOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createPost(t, repos, owner.ID, "general", "body")

	// when
	err := repos.Post.Delete(context.Background(), id, other.ID)

	// then
	require.Error(t, err)
}

func TestPostRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createPost(t, repos, owner.ID, "general", "body")

	// when
	err := repos.Post.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Post.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestPostRepository_ListAll(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "general", "post one")
	createPost(t, repos, user.ID, "general", "post two")
	createPost(t, repos, user.ID, "suggestions", "different corner")

	// when
	posts, total, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "new", 0, 10, 0, nil, "")

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, posts, 2)
}

func TestPostRepository_ListAll_Search(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "general", "apple pie")
	createPost(t, repos, user.ID, "general", "banana bread")

	// when
	posts, total, err := repos.Post.ListAll(context.Background(), user.ID, "general", "apple", "new", 0, 10, 0, nil, "")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, posts, 1)
	assert.Contains(t, posts[0].Body, "apple")
}

func TestPostRepository_ListAll_SortLikes(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postA := createPost(t, repos, user.ID, "general", "a")
	postB := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.Like(context.Background(), liker.ID, postB))

	// when
	posts, _, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "likes", 0, 10, 0, nil, "")

	// then
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, postB, posts[0].ID)
	assert.Equal(t, postA, posts[1].ID)
}

func TestPostRepository_ListAll_SortComments(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postA := createPost(t, repos, user.ID, "general", "a")
	postB := createPost(t, repos, user.ID, "general", "b")
	createComment(t, repos, postB, user.ID, nil, "c1")

	// when
	posts, _, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "comments", 0, 10, 0, nil, "")

	// then
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, postB, posts[0].ID)
	assert.Equal(t, postA, posts[1].ID)
}

func TestPostRepository_ListAll_SortViews(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postA := createPost(t, repos, user.ID, "general", "a")
	postB := createPost(t, repos, user.ID, "general", "b")
	_, _ = repos.Post.RecordView(context.Background(), postB, "hash1")

	// when
	posts, _, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "views", 0, 10, 0, nil, "")

	// then
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, postB, posts[0].ID)
	assert.Equal(t, postA, posts[1].ID)
}

func TestPostRepository_ListAll_SortRelevance(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "general", "a")
	createPost(t, repos, user.ID, "general", "b")

	// when
	posts, _, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "", 42, 10, 0, nil, "")

	// then
	require.NoError(t, err)
	assert.Len(t, posts, 2)
}

func TestPostRepository_ListAll_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 5; i++ {
		createPost(t, repos, user.ID, "general", "p")
	}

	// when
	page1, total, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "new", 0, 2, 0, nil, "")
	page2, _, err2 := repos.Post.ListAll(context.Background(), user.ID, "general", "", "new", 0, 2, 2, nil, "")

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
}

func TestPostRepository_ListAll_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "general", "mine")
	createPost(t, repos, blocked.ID, "general", "blocked")

	// when
	posts, total, err := repos.Post.ListAll(context.Background(), user.ID, "general", "", "new", 0, 10, 0, []uuid.UUID{blocked.ID}, "")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, posts, 1)
	assert.Equal(t, user.ID, posts[0].UserID)
}

func TestPostRepository_ListAll_ResolvedFilterOpen(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	open := createPost(t, repos, user.ID, "suggestions", "open suggestion")
	done := createPost(t, repos, user.ID, "suggestions", "done suggestion")
	require.NoError(t, repos.Post.ResolveSuggestion(context.Background(), done, user.ID, "done"))

	// when
	posts, total, err := repos.Post.ListAll(context.Background(), user.ID, "suggestions", "", "new", 0, 10, 0, nil, "open")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, posts, 1)
	assert.Equal(t, open, posts[0].ID)
}

func TestPostRepository_ListAll_ResolvedFilterDone(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "suggestions", "open")
	done := createPost(t, repos, user.ID, "suggestions", "done one")
	require.NoError(t, repos.Post.ResolveSuggestion(context.Background(), done, user.ID, "done"))

	// when
	posts, total, err := repos.Post.ListAll(context.Background(), user.ID, "suggestions", "", "new", 0, 10, 0, nil, "done")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, posts, 1)
	assert.Equal(t, done, posts[0].ID)
}

func TestPostRepository_ListAll_ResolvedFilterArchived(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	archived := createPost(t, repos, user.ID, "suggestions", "archived one")
	require.NoError(t, repos.Post.ResolveSuggestion(context.Background(), archived, user.ID, "archived"))

	// when
	posts, total, err := repos.Post.ListAll(context.Background(), user.ID, "suggestions", "", "new", 0, 10, 0, nil, "archived")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, posts, 1)
}

func TestPostRepository_ListByFollowing(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)
	followed := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), viewer.ID, followed.ID))
	createPost(t, repos, viewer.ID, "general", "self")
	createPost(t, repos, followed.ID, "general", "followed")
	createPost(t, repos, other.ID, "general", "other")

	// when
	posts, total, err := repos.Post.ListByFollowing(context.Background(), viewer.ID, "general", "new", 0, 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, posts, 2)
}

func TestPostRepository_ListByFollowing_RelevanceSort(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)
	followed := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), viewer.ID, followed.ID))
	createPost(t, repos, followed.ID, "general", "a")

	// when
	posts, _, err := repos.Post.ListByFollowing(context.Background(), viewer.ID, "general", "", 7, 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Len(t, posts, 1)
}

func TestPostRepository_ListByFollowing_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)
	followed := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), viewer.ID, followed.ID))
	createPost(t, repos, followed.ID, "general", "f")

	// when
	posts, total, err := repos.Post.ListByFollowing(context.Background(), viewer.ID, "general", "new", 0, 10, 0, []uuid.UUID{followed.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, posts, 0)
}

func TestPostRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	target := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createPost(t, repos, target.ID, "general", "mine1")
	createPost(t, repos, target.ID, "general", "mine2")
	createPost(t, repos, other.ID, "general", "not mine")

	// when
	posts, total, err := repos.Post.ListByUser(context.Background(), target.ID, target.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, posts, 2)
}

func TestPostRepository_ListByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	target := repotest.CreateUser(t, repos)
	for i := 0; i < 4; i++ {
		createPost(t, repos, target.ID, "general", "p")
	}

	// when
	posts, total, err := repos.Post.ListByUser(context.Background(), target.ID, target.ID, 2, 1)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, posts, 2)
}

func TestPostRepository_AddMedia_GetMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "body")

	// when
	id, err := repos.Post.AddMedia(context.Background(), postID, "/m.jpg", "image", "/t.jpg", 0)

	// then
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
	media, err := repos.Post.GetMedia(context.Background(), postID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/m.jpg", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
}

func TestPostRepository_DeleteMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id, err := repos.Post.AddMedia(context.Background(), postID, "/m.jpg", "image", "", 0)
	require.NoError(t, err)

	// when
	url, err := repos.Post.DeleteMedia(context.Background(), id, postID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "/m.jpg", url)
	media, _ := repos.Post.GetMedia(context.Background(), postID)
	assert.Len(t, media, 0)
}

func TestPostRepository_DeleteMedia_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	_, err := repos.Post.DeleteMedia(context.Background(), 99999, postID)

	// then
	require.Error(t, err)
}

func TestPostRepository_UpdateMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id, _ := repos.Post.AddMedia(context.Background(), postID, "/old.jpg", "image", "", 0)

	// when
	err := repos.Post.UpdateMediaURL(context.Background(), id, "/new.jpg")

	// then
	require.NoError(t, err)
	media, _ := repos.Post.GetMedia(context.Background(), postID)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.jpg", media[0].MediaURL)
}

func TestPostRepository_UpdateMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id, _ := repos.Post.AddMedia(context.Background(), postID, "/m.jpg", "image", "", 0)

	// when
	err := repos.Post.UpdateMediaThumbnail(context.Background(), id, "/thumb.jpg")

	// then
	require.NoError(t, err)
	media, _ := repos.Post.GetMedia(context.Background(), postID)
	require.Len(t, media, 1)
	assert.Equal(t, "/thumb.jpg", media[0].ThumbnailURL)
}

func TestPostRepository_GetMediaBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	p1 := createPost(t, repos, user.ID, "general", "a")
	p2 := createPost(t, repos, user.ID, "general", "b")
	_, _ = repos.Post.AddMedia(context.Background(), p1, "/a.jpg", "image", "", 0)
	_, _ = repos.Post.AddMedia(context.Background(), p2, "/b.jpg", "image", "", 0)

	// when
	result, err := repos.Post.GetMediaBatch(context.Background(), []uuid.UUID{p1, p2})

	// then
	require.NoError(t, err)
	assert.Len(t, result[p1], 1)
	assert.Len(t, result[p2], 1)
}

func TestPostRepository_GetMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Post.GetMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPostRepository_Like(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	err := repos.Post.Like(context.Background(), liker.ID, postID)

	// then
	require.NoError(t, err)
	row, _ := repos.Post.GetByID(context.Background(), postID, liker.ID)
	assert.Equal(t, 1, row.LikeCount)
	assert.True(t, row.UserLiked)
}

func TestPostRepository_Like_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.Like(context.Background(), liker.ID, postID))

	// when
	err := repos.Post.Like(context.Background(), liker.ID, postID)

	// then
	require.NoError(t, err)
	row, _ := repos.Post.GetByID(context.Background(), postID, liker.ID)
	assert.Equal(t, 1, row.LikeCount)
}

func TestPostRepository_Unlike(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.Like(context.Background(), liker.ID, postID))

	// when
	err := repos.Post.Unlike(context.Background(), liker.ID, postID)

	// then
	require.NoError(t, err)
	row, _ := repos.Post.GetByID(context.Background(), postID, liker.ID)
	assert.Equal(t, 0, row.LikeCount)
	assert.False(t, row.UserLiked)
}

func TestPostRepository_GetLikedBy(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos, repotest.WithDisplayName("Liker"))
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.Like(context.Background(), liker.ID, postID))

	// when
	users, err := repos.Post.GetLikedBy(context.Background(), postID, nil)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, liker.ID, users[0].ID)
	assert.Equal(t, "Liker", users[0].DisplayName)
}

func TestPostRepository_GetLikedBy_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker1 := repotest.CreateUser(t, repos)
	liker2 := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.Like(context.Background(), liker1.ID, postID))
	require.NoError(t, repos.Post.Like(context.Background(), liker2.ID, postID))

	// when
	users, err := repos.Post.GetLikedBy(context.Background(), postID, []uuid.UUID{liker2.ID})

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, liker1.ID, users[0].ID)
}

func TestPostRepository_RecordView_NewAndDuplicate(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	first, err := repos.Post.RecordView(context.Background(), postID, "hash1")
	second, err2 := repos.Post.RecordView(context.Background(), postID, "hash1")

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.True(t, first)
	assert.False(t, second)
	row, _ := repos.Post.GetByID(context.Background(), postID, user.ID)
	assert.Equal(t, 1, row.ViewCount)
}

func TestPostRepository_GetPostAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	authorID, err := repos.Post.GetPostAuthorID(context.Background(), postID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, authorID)
}

func TestPostRepository_GetPostAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Post.GetPostAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestPostRepository_ResolveSuggestion(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "suggestions", "idea")

	// when
	err := repos.Post.ResolveSuggestion(context.Background(), postID, user.ID, "done")

	// then
	require.NoError(t, err)
	row, _ := repos.Post.GetByID(context.Background(), postID, user.ID)
	assert.Equal(t, "done", row.ResolvedStatus)
}

func TestPostRepository_ResolveSuggestion_UpdateStatus(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "suggestions", "idea")
	require.NoError(t, repos.Post.ResolveSuggestion(context.Background(), postID, user.ID, "done"))

	// when
	err := repos.Post.ResolveSuggestion(context.Background(), postID, user.ID, "archived")

	// then
	require.NoError(t, err)
	row, _ := repos.Post.GetByID(context.Background(), postID, user.ID)
	assert.Equal(t, "archived", row.ResolvedStatus)
}

func TestPostRepository_UnresolveSuggestion(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "suggestions", "idea")
	require.NoError(t, repos.Post.ResolveSuggestion(context.Background(), postID, user.ID, "done"))

	// when
	err := repos.Post.UnresolveSuggestion(context.Background(), postID)

	// then
	require.NoError(t, err)
	row, _ := repos.Post.GetByID(context.Background(), postID, user.ID)
	assert.Equal(t, "", row.ResolvedStatus)
}

func TestPostRepository_CreateComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	id := createComment(t, repos, postID, user.ID, nil, "reply")

	// then
	comments, total, err := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, comments, 1)
	assert.Equal(t, id, comments[0].ID)
	assert.Equal(t, "reply", comments[0].Body)
}

func TestPostRepository_CreateComment_Threaded(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	parent := createComment(t, repos, postID, user.ID, nil, "parent")

	// when
	child := createComment(t, repos, postID, user.ID, &parent, "child")

	// then
	comments, _, err := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 2)
	var childRow *uuid.UUID
	for _, c := range comments {
		if c.ID == child {
			childRow = c.ParentID
		}
	}
	require.NotNil(t, childRow)
	assert.Equal(t, parent, *childRow)
}

func TestPostRepository_UpdateComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id := createComment(t, repos, postID, user.ID, nil, "original")

	// when
	err := repos.Post.UpdateComment(context.Background(), id, user.ID, "updated")

	// then
	require.NoError(t, err)
	comments, _, _ := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, nil)
	assert.Equal(t, "updated", comments[0].Body)
}

func TestPostRepository_UpdateComment_NotOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, owner.ID, "general", "b")
	id := createComment(t, repos, postID, owner.ID, nil, "c")

	// when
	err := repos.Post.UpdateComment(context.Background(), id, other.ID, "hack")

	// then
	require.Error(t, err)
}

func TestPostRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id := createComment(t, repos, postID, user.ID, nil, "orig")

	// when
	err := repos.Post.UpdateCommentAsAdmin(context.Background(), id, "admin edit")

	// then
	require.NoError(t, err)
	comments, _, _ := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, nil)
	assert.Equal(t, "admin edit", comments[0].Body)
}

func TestPostRepository_UpdateCommentAsAdmin_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Post.UpdateCommentAsAdmin(context.Background(), uuid.New(), "x")

	// then
	require.Error(t, err)
}

func TestPostRepository_DeleteComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id := createComment(t, repos, postID, user.ID, nil, "c")

	// when
	err := repos.Post.DeleteComment(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	_, total, _ := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, nil)
	assert.Equal(t, 0, total)
}

func TestPostRepository_DeleteComment_NotOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, owner.ID, "general", "b")
	id := createComment(t, repos, postID, owner.ID, nil, "c")

	// when
	err := repos.Post.DeleteComment(context.Background(), id, other.ID)

	// then
	require.Error(t, err)
}

func TestPostRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	id := createComment(t, repos, postID, user.ID, nil, "c")

	// when
	err := repos.Post.DeleteCommentAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	_, total, _ := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, nil)
	assert.Equal(t, 0, total)
}

func TestPostRepository_GetComments_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	for i := 0; i < 3; i++ {
		createComment(t, repos, postID, user.ID, nil, "c")
	}

	// when
	comments, total, err := repos.Post.GetComments(context.Background(), postID, user.ID, 2, 1, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, comments, 2)
}

func TestPostRepository_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	createComment(t, repos, postID, user.ID, nil, "ok")
	createComment(t, repos, postID, blocked.ID, nil, "blocked")

	// when
	comments, total, err := repos.Post.GetComments(context.Background(), postID, user.ID, 10, 0, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, comments, 1)
}

func TestPostRepository_GetCommentPostID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")

	// when
	got, err := repos.Post.GetCommentPostID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, postID, got)
}

func TestPostRepository_GetCommentPostID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Post.GetCommentPostID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestPostRepository_GetCommentAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")

	// when
	got, err := repos.Post.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestPostRepository_LikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")

	// when
	err := repos.Post.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	comments, _, _ := repos.Post.GetComments(context.Background(), postID, liker.ID, 10, 0, nil)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)
}

func TestPostRepository_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")
	require.NoError(t, repos.Post.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err := repos.Post.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	comments, _, _ := repos.Post.GetComments(context.Background(), postID, liker.ID, 10, 0, nil)
	assert.Equal(t, 1, comments[0].LikeCount)
}

func TestPostRepository_UnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")
	require.NoError(t, repos.Post.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err := repos.Post.UnlikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	comments, _, _ := repos.Post.GetComments(context.Background(), postID, liker.ID, 10, 0, nil)
	assert.Equal(t, 0, comments[0].LikeCount)
}

func TestPostRepository_AddCommentMedia_GetCommentMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")

	// when
	id, err := repos.Post.AddCommentMedia(context.Background(), commentID, "/m.jpg", "image", "/t.jpg", 0)

	// then
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
	media, err := repos.Post.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/m.jpg", media[0].MediaURL)
}

func TestPostRepository_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")
	id, _ := repos.Post.AddCommentMedia(context.Background(), commentID, "/old.jpg", "image", "", 0)

	// when
	err := repos.Post.UpdateCommentMediaURL(context.Background(), id, "/new.jpg")

	// then
	require.NoError(t, err)
	media, _ := repos.Post.GetCommentMedia(context.Background(), commentID)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.jpg", media[0].MediaURL)
}

func TestPostRepository_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	commentID := createComment(t, repos, postID, user.ID, nil, "c")
	id, _ := repos.Post.AddCommentMedia(context.Background(), commentID, "/m.jpg", "image", "", 0)

	// when
	err := repos.Post.UpdateCommentMediaThumbnail(context.Background(), id, "/t.jpg")

	// then
	require.NoError(t, err)
	media, _ := repos.Post.GetCommentMedia(context.Background(), commentID)
	require.Len(t, media, 1)
	assert.Equal(t, "/t.jpg", media[0].ThumbnailURL)
}

func TestPostRepository_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	c1 := createComment(t, repos, postID, user.ID, nil, "c1")
	c2 := createComment(t, repos, postID, user.ID, nil, "c2")
	_, _ = repos.Post.AddCommentMedia(context.Background(), c1, "/1.jpg", "image", "", 0)
	_, _ = repos.Post.AddCommentMedia(context.Background(), c2, "/2.jpg", "image", "", 0)

	// when
	result, err := repos.Post.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, result[c1], 1)
	assert.Len(t, result[c2], 1)
}

func TestPostRepository_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Post.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPostRepository_CountUserPostsToday(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "general", "a")
	createPost(t, repos, user.ID, "general", "b")

	// when
	count, err := repos.Post.CountUserPostsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPostRepository_CountUserPostsToday_None(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	count, err := repos.Post.CountUserPostsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPostRepository_GetCornerCounts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createPost(t, repos, user.ID, "general", "a")
	createPost(t, repos, user.ID, "general", "b")
	createPost(t, repos, user.ID, "suggestions", "c")

	// when
	counts, err := repos.Post.GetCornerCounts(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, counts["general"])
	assert.Equal(t, 1, counts["suggestions"])
}

func TestPostRepository_IncrementShareCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Post.IncrementShareCount(context.Background(), "abc", "post")

	// then
	require.NoError(t, err)
	count, err := repos.Post.GetShareCount(context.Background(), "abc", "post")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPostRepository_IncrementShareCount_Multiple(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "abc", "post"))

	// when
	err := repos.Post.IncrementShareCount(context.Background(), "abc", "post")

	// then
	require.NoError(t, err)
	count, _ := repos.Post.GetShareCount(context.Background(), "abc", "post")
	assert.Equal(t, 2, count)
}

func TestPostRepository_DecrementShareCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "abc", "post"))
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "abc", "post"))

	// when
	err := repos.Post.DecrementShareCount(context.Background(), "abc", "post")

	// then
	require.NoError(t, err)
	count, _ := repos.Post.GetShareCount(context.Background(), "abc", "post")
	assert.Equal(t, 1, count)
}

func TestPostRepository_DecrementShareCount_ClampsToZero(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "abc", "post"))

	// when
	err := repos.Post.DecrementShareCount(context.Background(), "abc", "post")
	err2 := repos.Post.DecrementShareCount(context.Background(), "abc", "post")

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	count, _ := repos.Post.GetShareCount(context.Background(), "abc", "post")
	assert.Equal(t, 0, count)
}

func TestPostRepository_GetShareCount_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	count, err := repos.Post.GetShareCount(context.Background(), "missing", "post")

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPostRepository_GetShareCountsBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "a", "post"))
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "b", "post"))
	require.NoError(t, repos.Post.IncrementShareCount(context.Background(), "b", "post"))

	// when
	result, err := repos.Post.GetShareCountsBatch(context.Background(), []string{"a", "b", "c"}, "post")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
	_, hasC := result["c"]
	assert.False(t, hasC)
}

func TestPostRepository_GetShareCountsBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Post.GetShareCountsBatch(context.Background(), nil, "post")

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPostRepository_GetSharedContentFields_None(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "body")

	// when
	cid, ctype, err := repos.Post.GetSharedContentFields(context.Background(), postID)

	// then
	require.NoError(t, err)
	assert.Nil(t, cid)
	assert.Nil(t, ctype)
}

func TestPostRepository_GetSharedContentFields_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	cid, ctype, err := repos.Post.GetSharedContentFields(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, cid)
	assert.Nil(t, ctype)
}

func TestPostRepository_AddEmbed_GetEmbeds(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	err := repos.Post.AddEmbed(context.Background(), postID.String(), "post", "https://x.com", "link", "Title", "Desc", "/img.jpg", "X", "", 0)

	// then
	require.NoError(t, err)
	embeds, err := repos.Post.GetEmbeds(context.Background(), postID.String(), "post")
	require.NoError(t, err)
	require.Len(t, embeds, 1)
	assert.Equal(t, "Title", embeds[0].Title)
	assert.Equal(t, "https://x.com", embeds[0].URL)
}

func TestPostRepository_DeleteEmbeds(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.AddEmbed(context.Background(), postID.String(), "post", "https://x.com", "link", "T", "D", "", "", "", 0))

	// when
	err := repos.Post.DeleteEmbeds(context.Background(), postID.String(), "post")

	// then
	require.NoError(t, err)
	embeds, _ := repos.Post.GetEmbeds(context.Background(), postID.String(), "post")
	assert.Len(t, embeds, 0)
}

func TestPostRepository_UpdateEmbed(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.AddEmbed(context.Background(), postID.String(), "post", "https://x.com", "link", "old", "old", "", "", "", 0))
	embeds, _ := repos.Post.GetEmbeds(context.Background(), postID.String(), "post")
	require.Len(t, embeds, 1)

	// when
	err := repos.Post.UpdateEmbed(context.Background(), embeds[0].ID, "new title", "new desc", "/new.jpg", "NewSite")

	// then
	require.NoError(t, err)
	embeds, _ = repos.Post.GetEmbeds(context.Background(), postID.String(), "post")
	require.Len(t, embeds, 1)
	assert.Equal(t, "new title", embeds[0].Title)
	assert.Equal(t, "new desc", embeds[0].Desc)
	assert.Equal(t, "NewSite", embeds[0].SiteName)
}

func TestPostRepository_GetEmbedsBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	p1 := createPost(t, repos, user.ID, "general", "a")
	p2 := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.AddEmbed(context.Background(), p1.String(), "post", "https://a.com", "link", "A", "", "", "", "", 0))
	require.NoError(t, repos.Post.AddEmbed(context.Background(), p2.String(), "post", "https://b.com", "link", "B", "", "", "", "", 0))

	// when
	result, err := repos.Post.GetEmbedsBatch(context.Background(), []string{p1.String(), p2.String()}, "post")

	// then
	require.NoError(t, err)
	assert.Len(t, result[p1.String()], 1)
	assert.Len(t, result[p2.String()], 1)
}

func TestPostRepository_GetEmbedsBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Post.GetEmbedsBatch(context.Background(), nil, "post")

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPostRepository_GetStaleEmbeds(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.AddEmbed(context.Background(), postID.String(), "post", "https://x.com", "link", "T", "D", "", "", "", 0))
	db := repos.DB()
	_, err := db.Exec(`UPDATE embeds SET fetched_at = NOW() - INTERVAL '30 days'`)
	require.NoError(t, err)

	// when
	stale, err := repos.Post.GetStaleEmbeds(context.Background(), "-7 days", 10)

	// then
	require.NoError(t, err)
	assert.Len(t, stale, 1)
}

func TestPostRepository_GetStaleEmbeds_None(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.AddEmbed(context.Background(), postID.String(), "post", "https://x.com", "link", "T", "D", "", "", "", 0))

	// when
	stale, err := repos.Post.GetStaleEmbeds(context.Background(), "-7 days", 10)

	// then
	require.NoError(t, err)
	assert.Len(t, stale, 0)
}

func TestPostRepository_GetStaleEmbeds_SkipsYouTube(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	require.NoError(t, repos.Post.AddEmbed(context.Background(), postID.String(), "post", "https://youtu.be/abc", "youtube", "T", "D", "", "", "abc", 0))
	db := repos.DB()
	_, err := db.Exec(`UPDATE embeds SET fetched_at = NOW() - INTERVAL '30 days'`)
	require.NoError(t, err)

	// when
	stale, err := repos.Post.GetStaleEmbeds(context.Background(), "-7 days", 10)

	// then
	require.NoError(t, err)
	assert.Len(t, stale, 0)
}

func TestPostRepository_CreatePollWithOptions_GetPollByPostID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "body")
	pollID := uuid.New()
	expires := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")

	// when
	err := repos.Post.CreatePollWithOptions(context.Background(), pollID, postID, 86400, expires, []string{"yes", "no"})

	// then
	require.NoError(t, err)
	poll, opts, voted, err := repos.Post.GetPollByPostID(context.Background(), postID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, poll)
	assert.Equal(t, pollID.String(), poll.ID)
	assert.Len(t, opts, 2)
	assert.Nil(t, voted)
}

func TestPostRepository_GetPollByPostID_NoPoll(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")

	// when
	poll, opts, voted, err := repos.Post.GetPollByPostID(context.Background(), postID, user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, poll)
	assert.Nil(t, opts)
	assert.Nil(t, voted)
}

func TestPostRepository_VotePoll(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	pollID := uuid.New()
	expires := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	require.NoError(t, repos.Post.CreatePollWithOptions(context.Background(), pollID, postID, 86400, expires, []string{"yes", "no"}))
	_, opts, _, _ := repos.Post.GetPollByPostID(context.Background(), postID, user.ID)
	require.Len(t, opts, 2)

	// when
	err := repos.Post.VotePoll(context.Background(), pollID, user.ID, opts[0].ID)

	// then
	require.NoError(t, err)
	_, opts2, voted, err := repos.Post.GetPollByPostID(context.Background(), postID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, voted)
	assert.Equal(t, opts[0].ID, *voted)
	var total int
	for _, o := range opts2 {
		total += o.VoteCount
	}
	assert.Equal(t, 1, total)
}

func TestPostRepository_VotePoll_DuplicateRejected(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "b")
	pollID := uuid.New()
	expires := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	require.NoError(t, repos.Post.CreatePollWithOptions(context.Background(), pollID, postID, 86400, expires, []string{"yes", "no"}))
	_, opts, _, _ := repos.Post.GetPollByPostID(context.Background(), postID, user.ID)
	require.NoError(t, repos.Post.VotePoll(context.Background(), pollID, user.ID, opts[0].ID))

	// when
	err := repos.Post.VotePoll(context.Background(), pollID, user.ID, opts[1].ID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already voted")
}

func TestPostRepository_GetPollsByPostIDs(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	p1 := createPost(t, repos, user.ID, "general", "a")
	p2 := createPost(t, repos, user.ID, "general", "b")
	poll1 := uuid.New()
	poll2 := uuid.New()
	expires := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	require.NoError(t, repos.Post.CreatePollWithOptions(context.Background(), poll1, p1, 86400, expires, []string{"a", "b"}))
	require.NoError(t, repos.Post.CreatePollWithOptions(context.Background(), poll2, p2, 86400, expires, []string{"c", "d"}))

	// when
	polls, opts, votes, err := repos.Post.GetPollsByPostIDs(context.Background(), []uuid.UUID{p1, p2}, user.ID)

	// then
	require.NoError(t, err)
	assert.Len(t, polls, 2)
	assert.Len(t, opts[p1], 2)
	assert.Len(t, opts[p2], 2)
	assert.Len(t, votes, 0)
}

func TestPostRepository_GetPollsByPostIDs_WithVote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	postID := createPost(t, repos, user.ID, "general", "a")
	pollID := uuid.New()
	expires := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	require.NoError(t, repos.Post.CreatePollWithOptions(context.Background(), pollID, postID, 86400, expires, []string{"a", "b"}))
	_, opts, _, _ := repos.Post.GetPollByPostID(context.Background(), postID, user.ID)
	require.NoError(t, repos.Post.VotePoll(context.Background(), pollID, user.ID, opts[1].ID))

	// when
	_, _, votes, err := repos.Post.GetPollsByPostIDs(context.Background(), []uuid.UUID{postID}, user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, votes[postID])
	assert.Equal(t, opts[1].ID, *votes[postID])
}

func TestPostRepository_GetPollsByPostIDs_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	polls, opts, votes, err := repos.Post.GetPollsByPostIDs(context.Background(), nil, user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, polls)
	assert.Nil(t, opts)
	assert.Nil(t, votes)
}

func TestGetSharedContentPreviews_Post(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Sharer"))
	postID := createPost(t, repos, user.ID, "general", "shared body")
	_, _ = repos.Post.AddMedia(context.Background(), postID, "/m.jpg", "image", "", 0)

	// when
	result := repository.GetSharedContentPreviews(repos.DB(), []repository.SharedContentRef{
		{ID: postID.String(), Type: "post"},
	})

	// then
	key := "post:" + postID.String()
	preview := result[key]
	require.NotNil(t, preview)
	assert.Equal(t, "post", preview.ContentType)
	assert.Equal(t, "shared body", preview.Body)
	assert.False(t, preview.Deleted)
	assert.Len(t, preview.Media, 1)
}

func TestGetSharedContentPreviews_MissingContentFlaggedDeleted(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	missingID := uuid.New().String()

	// when
	result := repository.GetSharedContentPreviews(repos.DB(), []repository.SharedContentRef{
		{ID: missingID, Type: "post"},
	})

	// then
	preview := result["post:"+missingID]
	require.NotNil(t, preview)
	assert.True(t, preview.Deleted)
	assert.Equal(t, "/game-board/"+missingID, preview.URL)
}

func TestGetSharedContentPreviews_EmptyRefs(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result := repository.GetSharedContentPreviews(repos.DB(), nil)

	// then
	assert.Empty(t, result)
}

func TestGetSharedContentPreviews_UnknownType(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result := repository.GetSharedContentPreviews(repos.DB(), []repository.SharedContentRef{
		{ID: "xyz", Type: "nonsense"},
	})

	// then
	preview := result["nonsense:xyz"]
	require.NotNil(t, preview)
	assert.True(t, preview.Deleted)
	assert.Equal(t, "/", preview.URL)
}
