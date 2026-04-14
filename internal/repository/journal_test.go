package repository_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createJournal(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title, body, work string) uuid.UUID {
	t.Helper()
	id, err := repos.Journal.Create(context.Background(), userID, dto.CreateJournalRequest{
		Title: title,
		Body:  body,
		Work:  work,
	})
	require.NoError(t, err)
	return id
}

func createJournalComment(t *testing.T, repos *repository.Repositories, journalID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Journal.CreateComment(context.Background(), id, journalID, parentID, userID, body))
	return id
}

func defaultJournalListParams() params.ListParams {
	return params.NewListParams("new", "", uuid.Nil, "", false, 20, 0)
}

func TestJournalRepository_Create_AssignsDefaultWork(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	id, err := repos.Journal.Create(context.Background(), user.ID, dto.CreateJournalRequest{
		Title: "Hello",
		Body:  "World",
		Work:  "",
	})

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "general", got.Work)
}

func TestJournalRepository_GetByID_HappyPath(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author"))
	id := createJournal(t, repos, user.ID, "Title", "Body", "umineko")

	// when
	got, err := repos.Journal.GetByID(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Title", got.Title)
	assert.Equal(t, "Body", got.Body)
	assert.Equal(t, "umineko", got.Work)
	assert.Equal(t, user.ID, got.Author.ID)
	assert.Equal(t, "Author", got.Author.DisplayName)
	assert.False(t, got.IsArchived)
	assert.Nil(t, got.ArchivedAt)
	assert.Equal(t, 0, got.FollowerCount)
	assert.Equal(t, 0, got.CommentCount)
	assert.False(t, got.IsFollowing)
}

func TestJournalRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Journal.GetByID(context.Background(), uuid.New(), uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalRepository_GetByID_ViewerFollowingReflected(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")
	require.NoError(t, repos.Journal.Follow(context.Background(), viewer.ID, id))

	// when
	got, err := repos.Journal.GetByID(context.Background(), id, viewer.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsFollowing)
	assert.Equal(t, 1, got.FollowerCount)
}

func TestJournalRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	journals, total, err := repos.Journal.List(context.Background(), defaultJournalListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, journals)
}

func TestJournalRepository_List_FilterByWork(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	umineko := createJournal(t, repos, user.ID, "U", "body", "umineko")
	createJournal(t, repos, user.ID, "H", "body", "higurashi")

	// when
	p := defaultJournalListParams()
	p.Work = "umineko"
	journals, total, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, journals, 1)
	assert.Equal(t, umineko, journals[0].ID)
}

func TestJournalRepository_List_FilterByAuthor(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	authorA := repotest.CreateUser(t, repos)
	authorB := repotest.CreateUser(t, repos)
	aID := createJournal(t, repos, authorA.ID, "A", "body", "general")
	createJournal(t, repos, authorB.ID, "B", "body", "general")

	// when
	p := defaultJournalListParams()
	p.AuthorID = authorA.ID
	journals, total, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, journals, 1)
	assert.Equal(t, aID, journals[0].ID)
}

func TestJournalRepository_List_Search(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	matchTitle := createJournal(t, repos, user.ID, "Witches and Magic", "body", "general")
	matchBody := createJournal(t, repos, user.ID, "Other", "contains magic inside", "general")
	createJournal(t, repos, user.ID, "Unrelated", "other body", "general")

	// when
	p := defaultJournalListParams()
	p.Search = "magic"
	journals, total, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	ids := []uuid.UUID{journals[0].ID, journals[1].ID}
	assert.Contains(t, ids, matchTitle)
	assert.Contains(t, ids, matchBody)
}

func TestJournalRepository_List_ExcludesArchivedByDefault(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	active := createJournal(t, repos, user.ID, "Active", "body", "general")
	stale := createJournal(t, repos, user.ID, "Stale", "body", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	journals, total, err := repos.Journal.List(context.Background(), defaultJournalListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, journals)
	_ = active
	_ = stale
}

func TestJournalRepository_List_IncludeArchived(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createJournal(t, repos, user.ID, "One", "body", "general")
	createJournal(t, repos, user.ID, "Two", "body", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	p := defaultJournalListParams()
	p.IncludeArchived = true
	journals, total, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, journals, 2)
	for _, j := range journals {
		assert.True(t, j.IsArchived)
	}
}

func TestJournalRepository_List_SortOld(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	first := createJournal(t, repos, user.ID, "First", "b", "general")
	time.Sleep(1100 * time.Millisecond)
	second := createJournal(t, repos, user.ID, "Second", "b", "general")

	// when
	p := defaultJournalListParams()
	p.Sort = "old"
	journals, _, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, journals, 2)
	assert.Equal(t, first, journals[0].ID)
	assert.Equal(t, second, journals[1].ID)
}

func TestJournalRepository_List_SortMostFollowed(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	followerA := repotest.CreateUser(t, repos)
	followerB := repotest.CreateUser(t, repos)
	popular := createJournal(t, repos, author.ID, "Popular", "b", "general")
	quiet := createJournal(t, repos, author.ID, "Quiet", "b", "general")
	require.NoError(t, repos.Journal.Follow(context.Background(), followerA.ID, popular))
	require.NoError(t, repos.Journal.Follow(context.Background(), followerB.ID, popular))

	// when
	p := defaultJournalListParams()
	p.Sort = "most_followed"
	journals, _, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, journals, 2)
	assert.Equal(t, popular, journals[0].ID)
	assert.Equal(t, quiet, journals[1].ID)
}

func TestJournalRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createJournal(t, repos, user.ID, "j", "body", "general")
	}

	// when
	p := defaultJournalListParams()
	p.Limit = 1
	p.Offset = 1
	journals, total, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, journals, 1)
}

func TestJournalRepository_List_TruncatesBody(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	longBody := ""
	for i := 0; i < 400; i++ {
		longBody += "a"
	}
	createJournal(t, repos, user.ID, "T", longBody, "general")

	// when
	journals, _, err := repos.Journal.List(context.Background(), defaultJournalListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, journals, 1)
	assert.Len(t, journals[0].Body, 303)
	assert.Contains(t, journals[0].Body, "...")
}

func TestJournalRepository_List_ExcludesBlockedUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	visible := createJournal(t, repos, author.ID, "Visible", "b", "general")
	createJournal(t, repos, blocked.ID, "Hidden", "b", "general")

	// when
	journals, total, err := repos.Journal.List(context.Background(), defaultJournalListParams(), uuid.Nil, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, journals, 1)
	assert.Equal(t, visible, journals[0].ID)
}

func TestJournalRepository_Update_Owned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "Old", "OldBody", "general")

	// when
	err := repos.Journal.Update(context.Background(), id, user.ID, dto.CreateJournalRequest{
		Title: "New",
		Body:  "NewBody",
		Work:  "higurashi",
	})

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "New", got.Title)
	assert.Equal(t, "NewBody", got.Body)
	assert.Equal(t, "higurashi", got.Work)
	require.NotNil(t, got.UpdatedAt)
}

func TestJournalRepository_Update_NotOwned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, owner.ID, "T", "B", "general")

	// when
	err := repos.Journal.Update(context.Background(), id, other.ID, dto.CreateJournalRequest{
		Title: "Hacked",
		Body:  "Hacked",
		Work:  "general",
	})

	// then
	require.Error(t, err)
}

func TestJournalRepository_Update_UnarchivesJournal(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	err = repos.Journal.Update(context.Background(), id, user.ID, dto.CreateJournalRequest{
		Title: "T2",
		Body:  "B2",
		Work:  "general",
	})

	// then
	require.NoError(t, err)
	archived, err := repos.Journal.IsArchived(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, archived)
}

func TestJournalRepository_UpdateAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	err := repos.Journal.UpdateAsAdmin(context.Background(), id, dto.CreateJournalRequest{
		Title: "Admin Title",
		Body:  "Admin Body",
		Work:  "general",
	})

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Admin Title", got.Title)
	assert.Equal(t, "Admin Body", got.Body)
}

func TestJournalRepository_Delete_Owned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	err := repos.Journal.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalRepository_Delete_NotOwned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, owner.ID, "T", "B", "general")

	// when
	err := repos.Journal.Delete(context.Background(), id, other.ID)

	// then
	require.Error(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestJournalRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	err := repos.Journal.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalRepository_GetAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	got, err := repos.Journal.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestJournalRepository_GetAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalRepository_GetTitle(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "My Title", "B", "general")

	// when
	got, err := repos.Journal.GetTitle(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, "My Title", got)
}

func TestJournalRepository_GetTitle_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetTitle(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalRepository_IsArchived_False(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	archived, err := repos.Journal.IsArchived(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.False(t, archived)
}

func TestJournalRepository_IsArchived_True(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	archived, err := repos.Journal.IsArchived(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.True(t, archived)
}

func TestJournalRepository_CountUserJournalsToday(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createJournal(t, repos, user.ID, "A", "b", "general")
	createJournal(t, repos, user.ID, "B", "b", "general")
	createJournal(t, repos, other.ID, "C", "b", "general")

	// when
	count, err := repos.Journal.CountUserJournalsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestJournalRepository_CountUserJournalsToday_Zero(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	count, err := repos.Journal.CountUserJournalsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestJournalRepository_UpdateLastAuthorActivity_Unarchives(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	err = repos.Journal.UpdateLastAuthorActivity(context.Background(), id)

	// then
	require.NoError(t, err)
	archived, err := repos.Journal.IsArchived(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, archived)
}

func TestJournalRepository_ArchiveStale_ReturnsIDsAndMarks(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	stale := createJournal(t, repos, user.ID, "Stale", "b", "general")

	// when
	ids, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))

	// then
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, stale, ids[0])
	archived, err := repos.Journal.IsArchived(context.Background(), stale)
	require.NoError(t, err)
	assert.True(t, archived)
}

func TestJournalRepository_ArchiveStale_NoneStale(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createJournal(t, repos, user.ID, "Fresh", "b", "general")

	// when
	ids, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(-time.Hour))

	// then
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestJournalRepository_FollowAndUnfollow(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	follower := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	require.NoError(t, repos.Journal.Follow(context.Background(), follower.ID, id))
	require.NoError(t, repos.Journal.Follow(context.Background(), follower.ID, id))
	isFollowerAfter, err := repos.Journal.IsFollower(context.Background(), follower.ID, id)
	require.NoError(t, err)
	countAfter, err := repos.Journal.GetFollowerCount(context.Background(), id)
	require.NoError(t, err)
	require.NoError(t, repos.Journal.Unfollow(context.Background(), follower.ID, id))
	isFollowerFinal, err := repos.Journal.IsFollower(context.Background(), follower.ID, id)
	require.NoError(t, err)

	// then
	assert.True(t, isFollowerAfter)
	assert.Equal(t, 1, countAfter)
	assert.False(t, isFollowerFinal)
}

func TestJournalRepository_IsFollower_False(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	got, err := repos.Journal.IsFollower(context.Background(), other.ID, id)

	// then
	require.NoError(t, err)
	assert.False(t, got)
}

func TestJournalRepository_GetFollowerIDs(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	followerA := repotest.CreateUser(t, repos)
	followerB := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")
	require.NoError(t, repos.Journal.Follow(context.Background(), followerA.ID, id))
	require.NoError(t, repos.Journal.Follow(context.Background(), followerB.ID, id))

	// when
	ids, err := repos.Journal.GetFollowerIDs(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{followerA.ID, followerB.ID}, ids)
}

func TestJournalRepository_GetFollowerIDs_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	ids, err := repos.Journal.GetFollowerIDs(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestJournalRepository_GetFollowerCount_Zero(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	count, err := repos.Journal.GetFollowerCount(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestJournalRepository_ListFollowedByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	follower := repotest.CreateUser(t, repos)
	a := createJournal(t, repos, author.ID, "A", "b", "general")
	b := createJournal(t, repos, author.ID, "B", "b", "general")
	createJournal(t, repos, author.ID, "C", "b", "general")
	require.NoError(t, repos.Journal.Follow(context.Background(), follower.ID, a))
	require.NoError(t, repos.Journal.Follow(context.Background(), follower.ID, b))

	// when
	journals, total, err := repos.Journal.ListFollowedByUser(context.Background(), follower.ID, follower.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, journals, 2)
	ids := []uuid.UUID{journals[0].ID, journals[1].ID}
	assert.Contains(t, ids, a)
	assert.Contains(t, ids, b)
	for _, j := range journals {
		assert.True(t, j.IsFollowing)
	}
}

func TestJournalRepository_ListFollowedByUser_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	journals, total, err := repos.Journal.ListFollowedByUser(context.Background(), user.ID, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, journals)
}

func TestJournalRepository_CreateComment_Flat(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	commentID := createJournalComment(t, repos, journalID, commenter.ID, nil, "hello")

	// then
	comments, total, err := repos.Journal.GetComments(context.Background(), journalID, commenter.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Nil(t, comments[0].ParentID)
	assert.Equal(t, "hello", comments[0].Body)
}

func TestJournalRepository_CreateComment_Threaded(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	parentID := createJournalComment(t, repos, journalID, commenter.ID, nil, "parent")

	// when
	childID := createJournalComment(t, repos, journalID, commenter.ID, &parentID, "child")

	// then
	comments, total, err := repos.Journal.GetComments(context.Background(), journalID, commenter.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	var child repository.JournalCommentRow
	for _, c := range comments {
		if c.ID == childID {
			child = c
		}
	}
	require.NotNil(t, child.ParentID)
	assert.Equal(t, parentID, *child.ParentID)
}

func TestJournalRepository_UpdateComment_OwnedAndNotOwned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "old")

	// when
	ownErr := repos.Journal.UpdateComment(context.Background(), commentID, author.ID, "new")
	notOwnedErr := repos.Journal.UpdateComment(context.Background(), commentID, other.ID, "evil")

	// then
	require.NoError(t, ownErr)
	require.Error(t, notOwnedErr)
	comments, _, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new", comments[0].Body)
	require.NotNil(t, comments[0].UpdatedAt)
}

func TestJournalRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "original")

	// when
	err := repos.Journal.UpdateCommentAsAdmin(context.Background(), commentID, "admin-edit")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin-edit", comments[0].Body)
}

func TestJournalRepository_DeleteComment_OwnedAndNotOwned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	notOwnedErr := repos.Journal.DeleteComment(context.Background(), commentID, other.ID)
	ownedErr := repos.Journal.DeleteComment(context.Background(), commentID, author.ID)

	// then
	require.Error(t, notOwnedErr)
	require.NoError(t, ownedErr)
	_, total, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestJournalRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	err := repos.Journal.DeleteCommentAsAdmin(context.Background(), commentID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestJournalRepository_GetComments_PaginationOrderingAndExclusion(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	commenterA := repotest.CreateUser(t, repos, repotest.WithDisplayName("A"))
	commenterB := repotest.CreateUser(t, repos, repotest.WithDisplayName("B"))
	blocked := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	first := createJournalComment(t, repos, journalID, commenterA.ID, nil, "first")
	second := createJournalComment(t, repos, journalID, commenterB.ID, nil, "second")
	createJournalComment(t, repos, journalID, blocked.ID, nil, "blocked-comment")

	// when
	all, total, err := repos.Journal.GetComments(context.Background(), journalID, commenterA.ID, 10, 0, nil)
	excluded, exclTotal, exclErr := repos.Journal.GetComments(context.Background(), journalID, commenterA.ID, 10, 0, []uuid.UUID{blocked.ID})
	page, _, pageErr := repos.Journal.GetComments(context.Background(), journalID, commenterA.ID, 1, 1, nil)

	// then
	require.NoError(t, err)
	require.NoError(t, exclErr)
	require.NoError(t, pageErr)
	assert.Equal(t, 3, total)
	require.Len(t, all, 3)
	assert.Equal(t, first, all[0].ID)
	assert.Equal(t, second, all[1].ID)
	assert.Equal(t, "A", all[0].AuthorDisplayName)
	assert.Equal(t, 2, exclTotal)
	for _, c := range excluded {
		assert.NotEqual(t, blocked.ID, c.UserID)
	}
	require.Len(t, page, 1)
	assert.Equal(t, second, page[0].ID)
}

func TestJournalRepository_GetComments_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	comments, total, err := repos.Journal.GetComments(context.Background(), journalID, user.ID, 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, comments)
}

func TestJournalRepository_GetCommentJournalID_AndAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	gotJournalID, journalErr := repos.Journal.GetCommentJournalID(context.Background(), commentID)
	gotAuthorID, authorErr := repos.Journal.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, journalErr)
	require.NoError(t, authorErr)
	assert.Equal(t, journalID, gotJournalID)
	assert.Equal(t, author.ID, gotAuthorID)
}

func TestJournalRepository_GetCommentJournalID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetCommentJournalID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalRepository_GetCommentAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetCommentAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalRepository_LikeAndUnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	require.NoError(t, repos.Journal.LikeComment(context.Background(), liker.ID, commentID))
	require.NoError(t, repos.Journal.LikeComment(context.Background(), liker.ID, commentID))
	likedComments, _, err := repos.Journal.GetComments(context.Background(), journalID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	require.NoError(t, repos.Journal.UnlikeComment(context.Background(), liker.ID, commentID))
	unlikedComments, _, err := repos.Journal.GetComments(context.Background(), journalID, liker.ID, 10, 0, nil)
	require.NoError(t, err)

	// then
	require.Len(t, likedComments, 1)
	assert.Equal(t, 1, likedComments[0].LikeCount)
	assert.True(t, likedComments[0].UserLiked)
	require.Len(t, unlikedComments, 1)
	assert.Equal(t, 0, unlikedComments[0].LikeCount)
	assert.False(t, unlikedComments[0].UserLiked)
}

func TestJournalRepository_AddCommentMedia_AndBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentA := createJournalComment(t, repos, journalID, author.ID, nil, "a")
	commentB := createJournalComment(t, repos, journalID, author.ID, nil, "b")
	commentC := createJournalComment(t, repos, journalID, author.ID, nil, "c")

	// when
	idA1, err := repos.Journal.AddCommentMedia(context.Background(), commentA, "url-a-1", "image", "thumb-a-1", 1)
	require.NoError(t, err)
	idA0, err := repos.Journal.AddCommentMedia(context.Background(), commentA, "url-a-0", "image", "thumb-a-0", 0)
	require.NoError(t, err)
	idB, err := repos.Journal.AddCommentMedia(context.Background(), commentB, "url-b", "video", "thumb-b", 0)
	require.NoError(t, err)
	batch, batchErr := repos.Journal.GetCommentMediaBatch(context.Background(), []uuid.UUID{commentA, commentB, commentC})

	// then
	require.NoError(t, batchErr)
	assert.Greater(t, idA1, int64(0))
	assert.Greater(t, idA0, int64(0))
	assert.Greater(t, idB, int64(0))
	require.Len(t, batch[commentA], 2)
	assert.Equal(t, "url-a-0", batch[commentA][0].MediaURL)
	assert.Equal(t, "url-a-1", batch[commentA][1].MediaURL)
	require.Len(t, batch[commentB], 1)
	assert.Equal(t, "url-b", batch[commentB][0].MediaURL)
	assert.Equal(t, "video", batch[commentB][0].MediaType)
	assert.NotContains(t, batch, commentC)
}

func TestJournalRepository_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Journal.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestJournalRepository_UpdateCommentMediaURLAndThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")
	mediaID, err := repos.Journal.AddCommentMedia(context.Background(), commentID, "old-url", "image", "old-thumb", 0)
	require.NoError(t, err)

	// when
	require.NoError(t, repos.Journal.UpdateCommentMediaURL(context.Background(), mediaID, "new-url"))
	require.NoError(t, repos.Journal.UpdateCommentMediaThumbnail(context.Background(), mediaID, "new-thumb"))

	// then
	batch, err := repos.Journal.GetCommentMediaBatch(context.Background(), []uuid.UUID{commentID})
	require.NoError(t, err)
	require.Len(t, batch[commentID], 1)
	assert.Equal(t, "new-url", batch[commentID][0].MediaURL)
	assert.Equal(t, "new-thumb", batch[commentID][0].ThumbnailURL)
}

func TestJournalRepository_CommentCountReflectedInJournal(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	createJournalComment(t, repos, journalID, author.ID, nil, "one")
	createJournalComment(t, repos, journalID, author.ID, nil, "two")

	// when
	got, err := repos.Journal.GetByID(context.Background(), journalID, uuid.Nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 2, got.CommentCount)
}
