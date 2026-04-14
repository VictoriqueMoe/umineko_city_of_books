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

func createAnnouncement(t *testing.T, repos *repository.Repositories, authorID uuid.UUID, title, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Announcement.Create(context.Background(), id, authorID, title, body))
	return id
}

func createAnnouncementComment(t *testing.T, repos *repository.Repositories, announcementID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Announcement.CreateComment(context.Background(), id, announcementID, parentID, userID, body))
	return id
}

func TestAnnouncementRepository_CreateAndGetByID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author Name"))

	// when
	id := createAnnouncement(t, repos, user.ID, "Hello", "World body")

	// then
	row, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, id, row.ID)
	assert.Equal(t, "Hello", row.Title)
	assert.Equal(t, "World body", row.Body)
	assert.Equal(t, user.ID, row.AuthorID)
	assert.Equal(t, user.Username, row.AuthorUsername)
	assert.Equal(t, "Author Name", row.AuthorDisplayName)
	assert.False(t, row.Pinned)
	assert.Equal(t, "", row.AuthorRole)
}

func TestAnnouncementRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	row, err := repos.Announcement.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestAnnouncementRepository_Update(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createAnnouncement(t, repos, user.ID, "Old", "Old body")

	// when
	err := repos.Announcement.Update(context.Background(), id, "New Title", "New body")

	// then
	require.NoError(t, err)
	row, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "New Title", row.Title)
	assert.Equal(t, "New body", row.Body)
}

func TestAnnouncementRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createAnnouncement(t, repos, user.ID, "T", "B")

	// when
	err := repos.Announcement.Delete(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestAnnouncementRepository_List_PaginationAndOrdering(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	first := createAnnouncement(t, repos, user.ID, "First", "1")
	second := createAnnouncement(t, repos, user.ID, "Second", "2")
	third := createAnnouncement(t, repos, user.ID, "Third", "3")
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), second, true))

	// when
	rows, total, err := repos.Announcement.List(context.Background(), 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, rows, 3)
	assert.Equal(t, second, rows[0].ID)
	assert.True(t, rows[0].Pinned)
	assert.Contains(t, []uuid.UUID{first, third}, rows[1].ID)
	assert.Contains(t, []uuid.UUID{first, third}, rows[2].ID)

	pageRows, pageTotal, err := repos.Announcement.List(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, pageTotal)
	assert.Len(t, pageRows, 1)
}

func TestAnnouncementRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, total, err := repos.Announcement.List(context.Background(), 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestAnnouncementRepository_GetLatest(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createAnnouncement(t, repos, user.ID, "First", "a")
	pinned := createAnnouncement(t, repos, user.ID, "Pinned", "b")
	createAnnouncement(t, repos, user.ID, "Third", "c")
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), pinned, true))

	// when
	latest, err := repos.Announcement.GetLatest(context.Background())

	// then
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, pinned, latest.ID)
}

func TestAnnouncementRepository_GetLatest_None(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	latest, err := repos.Announcement.GetLatest(context.Background())

	// then
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func TestAnnouncementRepository_SetPinned_Toggle(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createAnnouncement(t, repos, user.ID, "T", "B")

	// when
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), id, true))
	pinnedRow, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), id, false))
	unpinnedRow, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)

	// then
	assert.True(t, pinnedRow.Pinned)
	assert.False(t, unpinnedRow.Pinned)
}

func TestAnnouncementRepository_CreateComment_WithParent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	parentID := createAnnouncementComment(t, repos, annID, commenter.ID, nil, "parent")

	// when
	childID := createAnnouncementComment(t, repos, annID, commenter.ID, &parentID, "child")

	// then
	comments, total, err := repos.Announcement.GetComments(context.Background(), annID, commenter.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, comments, 2)
	var found bool
	for _, c := range comments {
		if c.ID == childID {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parentID, *c.ParentID)
			found = true
		}
	}
	assert.True(t, found)
}

func TestAnnouncementRepository_UpdateComment_OwnedAndNotOwned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "old")

	// when
	ownErr := repos.Announcement.UpdateComment(context.Background(), commentID, author.ID, "new")
	notOwnedErr := repos.Announcement.UpdateComment(context.Background(), commentID, other.ID, "evil")

	// then
	require.NoError(t, ownErr)
	require.Error(t, notOwnedErr)
	comments, _, err := repos.Announcement.GetComments(context.Background(), annID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new", comments[0].Body)
	require.NotNil(t, comments[0].UpdatedAt)
}

func TestAnnouncementRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "original")

	// when
	err := repos.Announcement.UpdateCommentAsAdmin(context.Background(), commentID, "admin-edit")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Announcement.GetComments(context.Background(), annID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin-edit", comments[0].Body)
}

func TestAnnouncementRepository_DeleteComment_OwnedAndNotOwned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "x")

	// when
	notOwnedErr := repos.Announcement.DeleteComment(context.Background(), commentID, other.ID)
	ownedErr := repos.Announcement.DeleteComment(context.Background(), commentID, author.ID)

	// then
	require.Error(t, notOwnedErr)
	require.NoError(t, ownedErr)
	_, total, err := repos.Announcement.GetComments(context.Background(), annID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestAnnouncementRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "x")

	// when
	err := repos.Announcement.DeleteCommentAsAdmin(context.Background(), commentID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Announcement.GetComments(context.Background(), annID, author.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestAnnouncementRepository_GetComments_PaginationOrderingAndExclusion(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author"))
	commenterA := repotest.CreateUser(t, repos, repotest.WithDisplayName("A"))
	commenterB := repotest.CreateUser(t, repos, repotest.WithDisplayName("B"))
	blocked := repotest.CreateUser(t, repos, repotest.WithDisplayName("Blocked"))
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	first := createAnnouncementComment(t, repos, annID, commenterA.ID, nil, "first")
	second := createAnnouncementComment(t, repos, annID, commenterB.ID, nil, "second")
	createAnnouncementComment(t, repos, annID, blocked.ID, nil, "blocked-comment")

	// when
	all, total, err := repos.Announcement.GetComments(context.Background(), annID, commenterA.ID, 10, 0, nil)
	excluded, exclTotal, exclErr := repos.Announcement.GetComments(context.Background(), annID, commenterA.ID, 10, 0, []uuid.UUID{blocked.ID})
	page, _, pageErr := repos.Announcement.GetComments(context.Background(), annID, commenterA.ID, 1, 1, nil)

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
	require.Len(t, excluded, 2)
	for _, c := range excluded {
		assert.NotEqual(t, blocked.ID, c.UserID)
	}
	require.Len(t, page, 1)
	assert.Equal(t, second, page[0].ID)
}

func TestAnnouncementRepository_GetCommentAnnouncementID_AndAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "x")

	// when
	gotAnnID, annErr := repos.Announcement.GetCommentAnnouncementID(context.Background(), commentID)
	gotAuthorID, authorErr := repos.Announcement.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, annErr)
	require.NoError(t, authorErr)
	assert.Equal(t, annID, gotAnnID)
	assert.Equal(t, author.ID, gotAuthorID)
}

func TestAnnouncementRepository_GetCommentAnnouncementID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Announcement.GetCommentAnnouncementID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestAnnouncementRepository_LikeAndUnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "x")

	// when
	require.NoError(t, repos.Announcement.LikeComment(context.Background(), liker.ID, commentID))
	require.NoError(t, repos.Announcement.LikeComment(context.Background(), liker.ID, commentID))
	likedComments, _, err := repos.Announcement.GetComments(context.Background(), annID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	require.NoError(t, repos.Announcement.UnlikeComment(context.Background(), liker.ID, commentID))
	unlikedComments, _, err := repos.Announcement.GetComments(context.Background(), annID, liker.ID, 10, 0, nil)
	require.NoError(t, err)

	// then
	require.Len(t, likedComments, 1)
	assert.Equal(t, 1, likedComments[0].LikeCount)
	assert.True(t, likedComments[0].UserLiked)
	require.Len(t, unlikedComments, 1)
	assert.Equal(t, 0, unlikedComments[0].LikeCount)
	assert.False(t, unlikedComments[0].UserLiked)
}

func TestAnnouncementRepository_AddCommentMedia_AndBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentA := createAnnouncementComment(t, repos, annID, author.ID, nil, "a")
	commentB := createAnnouncementComment(t, repos, annID, author.ID, nil, "b")
	commentC := createAnnouncementComment(t, repos, annID, author.ID, nil, "c")

	// when
	idA1, err := repos.Announcement.AddCommentMedia(context.Background(), commentA, "url-a-1", "image", "thumb-a-1", 1)
	require.NoError(t, err)
	idA0, err := repos.Announcement.AddCommentMedia(context.Background(), commentA, "url-a-0", "image", "thumb-a-0", 0)
	require.NoError(t, err)
	idB, err := repos.Announcement.AddCommentMedia(context.Background(), commentB, "url-b", "video", "thumb-b", 0)
	require.NoError(t, err)
	batch, batchErr := repos.Announcement.GetCommentMediaBatch(context.Background(), []uuid.UUID{commentA, commentB, commentC})

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

func TestAnnouncementRepository_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Announcement.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestAnnouncementRepository_UpdateCommentMediaURLAndThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, author.ID, nil, "x")
	mediaID, err := repos.Announcement.AddCommentMedia(context.Background(), commentID, "old-url", "image", "old-thumb", 0)
	require.NoError(t, err)

	// when
	require.NoError(t, repos.Announcement.UpdateCommentMediaURL(context.Background(), mediaID, "new-url"))
	require.NoError(t, repos.Announcement.UpdateCommentMediaThumbnail(context.Background(), mediaID, "new-thumb"))

	// then
	batch, err := repos.Announcement.GetCommentMediaBatch(context.Background(), []uuid.UUID{commentID})
	require.NoError(t, err)
	require.Len(t, batch[commentID], 1)
	assert.Equal(t, "new-url", batch[commentID][0].MediaURL)
	assert.Equal(t, "new-thumb", batch[commentID][0].ThumbnailURL)
}

func TestAnnouncementRepository_GetByID_WithRoleJoin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, user.ID, "T", "B")
	commentID := createAnnouncementComment(t, repos, annID, user.ID, nil, "c")

	// when
	annRow, err := repos.Announcement.GetByID(context.Background(), annID)
	require.NoError(t, err)
	comments, _, err := repos.Announcement.GetComments(context.Background(), annID, user.ID, 10, 0, nil)
	require.NoError(t, err)

	// then
	require.NotNil(t, annRow)
	assert.Equal(t, "", annRow.AuthorRole)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Equal(t, "", comments[0].AuthorRole)
	assert.Equal(t, user.Username, comments[0].AuthorUsername)
}
