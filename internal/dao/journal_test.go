package dao_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createJournal(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title, _body, work string) uuid.UUID {
	t.Helper()
	created, err := repos.Journal.Create(context.Background(), userID, dto.CreateJournalRequest{
		Title: title,
		Work:  work,
	})
	require.NoError(t, err)
	return created.ID
}

func createJournalComment(t *testing.T, repos *repository.Repositories, journalID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Journal.CreateComment(context.Background(), repository.NewJournalComment{
		JournalID: journalID,
		ParentID:  parentID,
		UserID:    userID,
		Body:      body,
	})
	require.NoError(t, err)
	return created.ID
}

func defaultJournalListParams() params.ListParams {
	return params.NewListParams("new", "", uuid.Nil, "", false, 20, 0)
}

func TestJournalDAO_Create_AssignsDefaultWork(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Journal.Create(context.Background(), user.ID, dto.CreateJournalRequest{
		Title: "Hello",
		Work:  "",
	})

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), created.ID, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "general", got.Work)
}

func TestJournalDAO_GetByID_HappyPath(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author"))
	id := createJournal(t, repos, user.ID, "Title", "Body", "umineko")

	// when
	got, err := repos.Journal.GetByID(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Title", got.Title)
	assert.Equal(t, "umineko", got.Work)
	assert.Equal(t, user.ID, got.Author.ID)
	assert.Equal(t, "Author", got.Author.DisplayName)
	assert.False(t, got.IsArchived)
	assert.Nil(t, got.ArchivedAt)
	assert.Equal(t, 0, got.FollowerCount)
	assert.Equal(t, 0, got.CommentCount)
	assert.False(t, got.IsFollowing)
}

func TestJournalDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Journal.GetByID(context.Background(), uuid.New(), uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalDAO_GetByID_ViewerFollowingReflected(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
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

func TestJournalDAO_List_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	journals, total, err := repos.Journal.List(context.Background(), defaultJournalListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, journals)
}

func TestJournalDAO_List_FilterByWork(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestJournalDAO_List_FilterByAuthor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	authorA := daotest.CreateUser(t, repos)
	authorB := daotest.CreateUser(t, repos)
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

func TestJournalDAO_List_SearchByTitle(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	matchTitle := createJournal(t, repos, user.ID, "Witches and Magic", "", "general")
	createJournal(t, repos, user.ID, "Unrelated", "", "general")

	// when
	p := defaultJournalListParams()
	p.Search = "magic"
	journals, total, err := repos.Journal.List(context.Background(), p, uuid.Nil, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, journals, 1)
	assert.Equal(t, matchTitle, journals[0].ID)
}

func TestJournalDAO_List_ExcludesArchivedByDefault(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestJournalDAO_List_IncludeArchived(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestJournalDAO_List_SortOld(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	first := createJournal(t, repos, user.ID, "First", "b", "general")
	second := createJournal(t, repos, user.ID, "Second", "b", "general")
	base := time.Now().UTC()

	for i, id := range []uuid.UUID{first, second} {
		_, err := repos.DB().ExecContext(context.Background(),
			`UPDATE journals SET created_at = $1 WHERE id = $2`,
			base.Add(time.Duration(i)*time.Minute), id,
		)
		require.NoError(t, err)
	}

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

func TestJournalDAO_List_SortMostFollowed(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	followerA := daotest.CreateUser(t, repos)
	followerB := daotest.CreateUser(t, repos)
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

func TestJournalDAO_List_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
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

func TestJournalDAO_List_TruncatesLatestEntryExcerpt(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "", "general")
	var longBody strings.Builder
	for range 400 {
		longBody.WriteString("a")
	}
	_, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: id, EntryNumber: 1, Body: longBody.String(), WordCount: 1})
	require.NoError(t, err)

	// when
	journals, _, err := repos.Journal.List(context.Background(), defaultJournalListParams(), uuid.Nil, nil)

	// then
	require.NoError(t, err)
	require.Len(t, journals, 1)
	assert.Len(t, journals[0].LatestEntryExcerpt, 303)
}

func TestJournalDAO_List_ExcludesBlockedUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
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

func TestJournalDAO_Update_Owned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "Old", "OldBody", "general")

	// when
	err := repos.Journal.Update(context.Background(), repository.JournalUpdate{
		ID:     id,
		UserID: user.ID,
		Title:  "New",
		Work:   "higurashi",
	})

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "New", got.Title)
	assert.Equal(t, "higurashi", got.Work)
	require.NotNil(t, got.UpdatedAt)
}

func TestJournalDAO_Update_NotOwned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, owner.ID, "T", "B", "general")

	// when
	err := repos.Journal.Update(context.Background(), repository.JournalUpdate{
		ID:     id,
		UserID: other.ID,
		Title:  "Hacked",
		Work:   "general",
	})

	// then
	require.Error(t, err)
}

func TestJournalDAO_Update_UnarchivesJournal(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	err = repos.Journal.Update(context.Background(), repository.JournalUpdate{
		ID:     id,
		UserID: user.ID,
		Title:  "T2",
		Work:   "general",
	})

	// then
	require.NoError(t, err)
	archived, err := repos.Journal.IsArchived(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, archived)
}

func TestJournalDAO_UpdateAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	err := repos.Journal.Update(context.Background(), repository.JournalUpdate{
		ID:      id,
		Title:   "Admin Title",
		Work:    "general",
		AsAdmin: true,
	})

	// then
	require.NoError(t, err)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Admin Title", got.Title)
}

func TestJournalDAO_Delete_Owned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	paths, err := repos.Journal.Delete(context.Background(), id, user.ID, false)

	// then
	require.NoError(t, err)
	assert.Empty(t, paths)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalDAO_Delete_NotOwned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, owner.ID, "T", "B", "general")

	// when
	paths, err := repos.Journal.Delete(context.Background(), id, other.ID, false)

	// then
	require.Error(t, err)
	assert.Nil(t, paths)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestJournalDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	paths, err := repos.Journal.Delete(context.Background(), id, uuid.Nil, true)

	// then
	require.NoError(t, err)
	assert.Empty(t, paths)
	got, err := repos.Journal.GetByID(context.Background(), id, uuid.Nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalDAO_GetAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	got, err := repos.Journal.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestJournalDAO_GetAuthorID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalDAO_GetTitle(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "My Title", "B", "general")

	// when
	got, err := repos.Journal.GetTitle(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, "My Title", got)
}

func TestJournalDAO_GetTitle_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetTitle(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalDAO_IsArchived_False(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	archived, err := repos.Journal.IsArchived(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.False(t, archived)
}

func TestJournalDAO_IsArchived_True(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, user.ID, "T", "B", "general")
	_, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	// when
	archived, err := repos.Journal.IsArchived(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.True(t, archived)
}

func TestJournalDAO_CountUserJournalsToday(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	createJournal(t, repos, user.ID, "A", "b", "general")
	createJournal(t, repos, user.ID, "B", "b", "general")
	createJournal(t, repos, other.ID, "C", "b", "general")

	// when
	count, err := repos.Journal.CountUserJournalsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestJournalDAO_CountUserJournalsToday_Zero(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	count, err := repos.Journal.CountUserJournalsToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestJournalDAO_UpdateLastAuthorActivity_Unarchives(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestJournalDAO_ArchiveStale_ReturnsIDsAndMarks(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestJournalDAO_ArchiveStale_NoneStale(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createJournal(t, repos, user.ID, "Fresh", "b", "general")

	// when
	ids, err := repos.Journal.ArchiveStale(context.Background(), time.Now().Add(-time.Hour))

	// then
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestJournalDAO_FollowAndUnfollow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	follower := daotest.CreateUser(t, repos)
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

func TestJournalDAO_IsFollower_False(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	got, err := repos.Journal.IsFollower(context.Background(), other.ID, id)

	// then
	require.NoError(t, err)
	assert.False(t, got)
}

func TestJournalDAO_GetFollowerIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	followerA := daotest.CreateUser(t, repos)
	followerB := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")
	require.NoError(t, repos.Journal.Follow(context.Background(), followerA.ID, id))
	require.NoError(t, repos.Journal.Follow(context.Background(), followerB.ID, id))

	// when
	ids, err := repos.Journal.GetFollowerIDs(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{followerA.ID, followerB.ID}, ids)
}

func TestJournalDAO_GetFollowerIDs_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	ids, err := repos.Journal.GetFollowerIDs(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestJournalDAO_GetFollowerCount_Zero(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	id := createJournal(t, repos, author.ID, "T", "B", "general")

	// when
	count, err := repos.Journal.GetFollowerCount(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestJournalDAO_ListFollowedByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	follower := daotest.CreateUser(t, repos)
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

func TestJournalDAO_ListFollowedByUser_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	journals, total, err := repos.Journal.ListFollowedByUser(context.Background(), user.ID, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, journals)
}

func TestJournalDAO_CreateComment_Flat(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
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

func TestJournalDAO_CreateComment_Threaded(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	parentID := createJournalComment(t, repos, journalID, commenter.ID, nil, "parent")

	// when
	childID := createJournalComment(t, repos, journalID, commenter.ID, &parentID, "child")

	// then
	comments, total, err := repos.Journal.GetComments(context.Background(), journalID, commenter.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	var child repository.CommentRow
	for _, c := range comments {
		if c.ID == childID {
			child = c
		}
	}
	require.NotNil(t, child.ParentID)
	assert.Equal(t, parentID, *child.ParentID)
}

func TestJournalDAO_UpdateComment_OwnedAndNotOwned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "old")

	// when
	ownErr := repos.Journal.UpdateComment(context.Background(), repository.JournalCommentUpdate{ID: commentID, UserID: author.ID, Body: "new"})
	notOwnedErr := repos.Journal.UpdateComment(context.Background(), repository.JournalCommentUpdate{ID: commentID, UserID: other.ID, Body: "evil"})

	// then
	require.NoError(t, ownErr)
	require.Error(t, notOwnedErr)
	comments, _, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new", comments[0].Body)
	require.NotNil(t, comments[0].UpdatedAt)
}

func TestJournalDAO_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "original")

	// when
	err := repos.Journal.UpdateComment(context.Background(), repository.JournalCommentUpdate{ID: commentID, Body: "admin-edit", AsAdmin: true})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin-edit", comments[0].Body)
}

func TestJournalDAO_DeleteComment_OwnedAndNotOwned(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	_, notOwnedErr := repos.Journal.DeleteComment(context.Background(), commentID, other.ID, false)
	_, ownedErr := repos.Journal.DeleteComment(context.Background(), commentID, author.ID, false)

	// then
	require.Error(t, notOwnedErr)
	require.NoError(t, ownedErr)
	_, total, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestJournalDAO_DeleteComment_NotOwnedLeavesNoAuditRow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	_, err := repos.Journal.DeleteComment(context.Background(), commentID, other.ID, false)

	// then
	require.Error(t, err)
	entries, total, listErr := repos.AuditLog.List(context.Background(), repository.AuditActionJournalCommentDelete, 10, 0)
	require.NoError(t, listErr)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestJournalDAO_DeleteComment_AsAdminWritesAuditRow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	_, err := repos.Journal.DeleteComment(context.Background(), commentID, admin.ID, true)

	// then
	require.NoError(t, err)
	_, total, err := repos.Journal.GetComments(context.Background(), journalID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	entries, auditTotal, auditErr := repos.AuditLog.List(context.Background(), repository.AuditActionJournalCommentDeleteAdmin, 10, 0)
	require.NoError(t, auditErr)
	assert.Equal(t, 1, auditTotal)
	require.Len(t, entries, 1)
	assert.Equal(t, admin.ID, entries[0].ActorID)
	assert.Equal(t, repository.AuditTargetJournalComment, entries[0].TargetType)
	assert.Equal(t, commentID.String(), entries[0].TargetID)
}

func TestJournalDAO_GetComments_PaginationOrderingAndExclusion(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	commenterA := daotest.CreateUser(t, repos, daotest.WithDisplayName("A"))
	commenterB := daotest.CreateUser(t, repos, daotest.WithDisplayName("B"))
	blocked := daotest.CreateUser(t, repos)
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

func TestJournalDAO_GetComments_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "T", "B", "general")

	// when
	comments, total, err := repos.Journal.GetComments(context.Background(), journalID, user.ID, 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, comments)
}

func TestJournalDAO_GetCommentEntityID_AndAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")

	// when
	gotJournalID, journalErr := repos.Journal.GetCommentEntityID(context.Background(), commentID)
	gotAuthorID, authorErr := repos.Journal.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, journalErr)
	require.NoError(t, authorErr)
	assert.Equal(t, journalID, gotJournalID)
	assert.Equal(t, author.ID, gotAuthorID)
}

func TestJournalDAO_GetCommentEntityID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetCommentEntityID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalDAO_GetCommentAuthorID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Journal.GetCommentAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestJournalDAO_LikeAndUnlikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
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

func TestJournalDAO_AddCommentMedia_AndBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentA := createJournalComment(t, repos, journalID, author.ID, nil, "a")
	commentB := createJournalComment(t, repos, journalID, author.ID, nil, "b")
	commentC := createJournalComment(t, repos, journalID, author.ID, nil, "c")

	// when
	idA0, err := repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: commentA, MediaURL: "url-a-0", MediaType: "image", ThumbnailURL: "thumb-a-0"})
	require.NoError(t, err)
	idA1, err := repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: commentA, MediaURL: "url-a-1", MediaType: "image", ThumbnailURL: "thumb-a-1"})
	require.NoError(t, err)
	idB, err := repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: commentB, MediaURL: "url-b", MediaType: "video", ThumbnailURL: "thumb-b"})
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

func TestJournalDAO_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	result, err := repos.Journal.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestJournalDAO_UpdateCommentMediaURLAndThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, author.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, author.ID, nil, "x")
	mediaID, err := repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: commentID, MediaURL: "old-url", MediaType: "image", ThumbnailURL: "old-thumb"})
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

func TestJournalDAO_CommentCountReflectedInJournal(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
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

func TestJournalDAO_CreateAndGetEntry(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	jid := createJournal(t, repos, user.ID, "Title", "", "general")
	entry, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: 1, Title: new("Day 1"), Body: "the body", WordCount: 2})
	require.NoError(t, err)
	entryID := entry.ID

	// when
	got, err := repos.Journal.GetEntry(context.Background(), jid, 1)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entryID, got.ID)
	assert.Equal(t, 1, got.EntryNumber)
	require.NotNil(t, got.Title)
	assert.Equal(t, "Day 1", *got.Title)
	assert.Equal(t, "the body", got.Body)
	assert.False(t, got.HasPrev)
	assert.False(t, got.HasNext)
}

func TestJournalDAO_GetEntry_PrevNext(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	jid := createJournal(t, repos, user.ID, "Title", "", "general")
	for i := 1; i <= 3; i++ {
		_, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: i, Body: "body", WordCount: 1})
		require.NoError(t, err)
	}

	// when
	first, _ := repos.Journal.GetEntry(context.Background(), jid, 1)
	middle, _ := repos.Journal.GetEntry(context.Background(), jid, 2)
	last, _ := repos.Journal.GetEntry(context.Background(), jid, 3)

	// then
	assert.False(t, first.HasPrev)
	assert.True(t, first.HasNext)
	assert.True(t, middle.HasPrev)
	assert.True(t, middle.HasNext)
	assert.True(t, last.HasPrev)
	assert.False(t, last.HasNext)
}

func TestJournalDAO_ListEntries_NewestFirst(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	jid := createJournal(t, repos, user.ID, "Title", "", "general")
	for i := 1; i <= 3; i++ {
		_, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: i, Body: "b", WordCount: 1})
		require.NoError(t, err)
	}

	// when
	entries, err := repos.Journal.ListEntries(context.Background(), jid)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, 3, entries[0].EntryNumber)
	assert.Equal(t, 2, entries[1].EntryNumber)
	assert.Equal(t, 1, entries[2].EntryNumber)
}

func TestJournalDAO_GetNextEntryNumber(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	jid := createJournal(t, repos, user.ID, "Title", "", "general")
	next, err := repos.Journal.GetNextEntryNumber(context.Background(), jid)
	require.NoError(t, err)
	assert.Equal(t, 1, next)

	_, err = repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: 1, Body: "b", WordCount: 1})
	require.NoError(t, err)
	_, err = repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: 2, Body: "b", WordCount: 1})
	require.NoError(t, err)

	// when
	next, err = repos.Journal.GetNextEntryNumber(context.Background(), jid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, next)
}

func TestJournalDAO_GetByID_PopulatesLatestEntry(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	jid := createJournal(t, repos, user.ID, "Title", "", "general")
	_, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: 1, Body: "first body", WordCount: 2})
	require.NoError(t, err)
	_, err = repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: 2, Title: new("Latest"), Body: "newest body", WordCount: 2})
	require.NoError(t, err)

	// when
	got, err := repos.Journal.GetByID(context.Background(), jid, uuid.Nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.LatestEntryNumber)
	assert.Equal(t, 2, *got.LatestEntryNumber)
	require.NotNil(t, got.LatestEntryTitle)
	assert.Equal(t, "Latest", *got.LatestEntryTitle)
	assert.Equal(t, "newest body", got.LatestEntryExcerpt)
	assert.Equal(t, 2, got.EntryCount)
}

func TestJournalDAO_EntryComments_ScopedSeparately(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	jid := createJournal(t, repos, user.ID, "Title", "", "general")
	entry, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: jid, EntryNumber: 1, Body: "b", WordCount: 1})
	require.NoError(t, err)
	entryID := entry.ID

	// when
	topLevelCommentRow, err := repos.Journal.CreateComment(context.Background(), repository.NewJournalComment{JournalID: jid, UserID: user.ID, Body: "on journal"})
	require.NoError(t, err)
	topLevelComment := topLevelCommentRow.ID
	entryCommentRow, err := repos.Journal.CreateComment(context.Background(), repository.NewJournalComment{JournalID: jid, EntryID: &entryID, UserID: user.ID, Body: "on entry"})
	require.NoError(t, err)
	entryComment := entryCommentRow.ID

	jrComments, _, err := repos.Journal.GetComments(context.Background(), jid, uuid.Nil, 100, 0, nil)
	require.NoError(t, err)
	enComments, _, err := repos.Journal.GetEntryComments(context.Background(), entryID, uuid.Nil, 100, 0, nil)
	require.NoError(t, err)

	// then: title-page query only returns the journal-level comment
	require.Len(t, jrComments, 1)
	assert.Equal(t, topLevelComment, jrComments[0].ID)
	// entry query only returns the entry-scoped comment
	require.Len(t, enComments, 1)
	assert.Equal(t, entryComment, enComments[0].ID)
	require.NotNil(t, enComments[0].EntryID)
	assert.Equal(t, entryID, *enComments[0].EntryID)
}

func TestJournalDAO_Delete_ReturnsEveryEntryAndCommentMediaPath(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "T", "B", "general")
	entry, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "b", WordCount: 1})
	require.NoError(t, err)
	_, err = repos.Journal.AddMedia(context.Background(), repository.NewJournalEntryMedia{EntryID: entry.ID, MediaURL: "/uploads/journal/entry.png", MediaType: "image", ThumbnailURL: "/uploads/journal/entry_thumb.png"})
	require.NoError(t, err)
	_, err = repos.Journal.AddMedia(context.Background(), repository.NewJournalEntryMedia{EntryID: entry.ID, MediaURL: "/uploads/journal/entry_no_thumb.gif", MediaType: "image"})
	require.NoError(t, err)
	journalComment := createJournalComment(t, repos, journalID, user.ID, nil, "on journal")
	_, err = repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: journalComment, MediaURL: "/uploads/journal/comment.png", MediaType: "image", ThumbnailURL: "/uploads/journal/comment_thumb.png"})
	require.NoError(t, err)
	entryComment, err := repos.Journal.CreateComment(context.Background(), repository.NewJournalComment{JournalID: journalID, EntryID: &entry.ID, UserID: user.ID, Body: "on entry"})
	require.NoError(t, err)
	_, err = repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: entryComment.ID, MediaURL: "/uploads/journal/entry_comment.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	paths, err := repos.Journal.Delete(context.Background(), journalID, user.ID, false)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/journal/entry.png",
		"/uploads/journal/entry_thumb.png",
		"/uploads/journal/entry_no_thumb.gif",
		"/uploads/journal/comment.png",
		"/uploads/journal/comment_thumb.png",
		"/uploads/journal/entry_comment.png",
	}, paths)
	assert.NotContains(t, paths, "")
	got, err := repos.Journal.GetByID(context.Background(), journalID, uuid.Nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalDAO_DeleteEntry_ReturnsOnlyThatEntrysMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "T", "B", "general")
	entry, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: journalID, EntryNumber: 1, Body: "b", WordCount: 1})
	require.NoError(t, err)
	survivingEntry, err := repos.Journal.CreateEntry(context.Background(), repository.NewJournalEntry{JournalID: journalID, EntryNumber: 2, Body: "b", WordCount: 1})
	require.NoError(t, err)
	_, err = repos.Journal.AddMedia(context.Background(), repository.NewJournalEntryMedia{EntryID: entry.ID, MediaURL: "/uploads/journal/entry.png", MediaType: "image", ThumbnailURL: "/uploads/journal/entry_thumb.png"})
	require.NoError(t, err)
	_, err = repos.Journal.AddMedia(context.Background(), repository.NewJournalEntryMedia{EntryID: survivingEntry.ID, MediaURL: "/uploads/journal/kept_entry.png", MediaType: "image"})
	require.NoError(t, err)
	entryComment, err := repos.Journal.CreateComment(context.Background(), repository.NewJournalComment{JournalID: journalID, EntryID: &entry.ID, UserID: user.ID, Body: "on entry"})
	require.NoError(t, err)
	_, err = repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: entryComment.ID, MediaURL: "/uploads/journal/entry_comment.png", MediaType: "image", ThumbnailURL: "/uploads/journal/entry_comment_thumb.png"})
	require.NoError(t, err)
	reply, err := repos.Journal.CreateComment(context.Background(), repository.NewJournalComment{JournalID: journalID, ParentID: &entryComment.ID, UserID: user.ID, Body: "reply"})
	require.NoError(t, err)
	_, err = repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: reply.ID, MediaURL: "/uploads/journal/reply.png", MediaType: "image"})
	require.NoError(t, err)
	journalComment := createJournalComment(t, repos, journalID, user.ID, nil, "on journal")
	_, err = repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: journalComment, MediaURL: "/uploads/journal/kept_comment.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	paths, err := repos.Journal.DeleteEntry(context.Background(), entry.ID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/journal/entry.png",
		"/uploads/journal/entry_thumb.png",
		"/uploads/journal/entry_comment.png",
		"/uploads/journal/entry_comment_thumb.png",
		"/uploads/journal/reply.png",
	}, paths)
	assert.NotContains(t, paths, "")
	keptEntryMedia, err := repos.Journal.GetMediaBatch(context.Background(), []uuid.UUID{survivingEntry.ID})
	require.NoError(t, err)
	require.Len(t, keptEntryMedia[survivingEntry.ID], 1)
	assert.Equal(t, "/uploads/journal/kept_entry.png", keptEntryMedia[survivingEntry.ID][0].MediaURL)
	keptCommentMedia, err := repos.Journal.GetCommentMediaBatch(context.Background(), []uuid.UUID{journalComment})
	require.NoError(t, err)
	require.Len(t, keptCommentMedia[journalComment], 1)
	assert.Equal(t, "/uploads/journal/kept_comment.png", keptCommentMedia[journalComment][0].MediaURL)
}

func TestJournalDAO_DeleteComment_ReturnsOnlyThatCommentsMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	journalID := createJournal(t, repos, user.ID, "T", "B", "general")
	commentID := createJournalComment(t, repos, journalID, user.ID, nil, "mine")
	_, err := repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: commentID, MediaURL: "/uploads/journal/comment.png", MediaType: "image", ThumbnailURL: "/uploads/journal/comment_thumb.png"})
	require.NoError(t, err)
	otherCommentID := createJournalComment(t, repos, journalID, user.ID, nil, "theirs")
	_, err = repos.Journal.AddCommentMedia(context.Background(), repository.NewJournalCommentMedia{CommentID: otherCommentID, MediaURL: "/uploads/journal/kept.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	paths, err := repos.Journal.DeleteComment(context.Background(), commentID, user.ID, false)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/journal/comment.png", "/uploads/journal/comment_thumb.png"}, paths)
	assert.NotContains(t, paths, "")
	keptMedia, err := repos.Journal.GetCommentMediaBatch(context.Background(), []uuid.UUID{otherCommentID})
	require.NoError(t, err)
	require.Len(t, keptMedia[otherCommentID], 1)
	assert.Equal(t, "/uploads/journal/kept.png", keptMedia[otherCommentID][0].MediaURL)
}
