package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createAnnouncement(t *testing.T, repos *repository.Repositories, authorID uuid.UUID, title, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Announcement.Create(context.Background(), spec.NewAnnouncement{AuthorID: authorID, Title: title, Body: body})
	require.NoError(t, err)
	return created.ID
}

func TestAnnouncementDAO_CreateAndGetByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author Name"))

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

func TestAnnouncementDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	row, err := repos.Announcement.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestAnnouncementDAO_Update(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createAnnouncement(t, repos, user.ID, "Old", "Old body")

	// when
	err := repos.Announcement.Update(context.Background(), spec.AnnouncementUpdate{ID: id, Title: "New Title", Body: "New body"})

	// then
	require.NoError(t, err)
	row, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "New Title", row.Title)
	assert.Equal(t, "New body", row.Body)
}

func TestAnnouncementDAO_Delete(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createAnnouncement(t, repos, user.ID, "T", "B")

	// when
	err := repos.Announcement.Delete(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestAnnouncementDAO_List_PaginationAndOrdering(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	first := createAnnouncement(t, repos, user.ID, "First", "1")
	second := createAnnouncement(t, repos, user.ID, "Second", "2")
	third := createAnnouncement(t, repos, user.ID, "Third", "3")
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), spec.AnnouncementPinUpdate{ID: second, Pinned: true}))

	// when
	rows, total, err := repos.Announcement.List(context.Background(), spec.AnnouncementListQuery{Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, rows, 3)
	assert.Equal(t, second, rows[0].ID)
	assert.True(t, rows[0].Pinned)
	assert.Contains(t, []uuid.UUID{first, third}, rows[1].ID)
	assert.Contains(t, []uuid.UUID{first, third}, rows[2].ID)

	pageRows, pageTotal, err := repos.Announcement.List(context.Background(), spec.AnnouncementListQuery{Limit: 1, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, 3, pageTotal)
	assert.Len(t, pageRows, 1)
}

func TestAnnouncementDAO_List_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	rows, total, err := repos.Announcement.List(context.Background(), spec.AnnouncementListQuery{Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestAnnouncementDAO_GetLatest(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createAnnouncement(t, repos, user.ID, "First", "a")
	pinned := createAnnouncement(t, repos, user.ID, "Pinned", "b")
	createAnnouncement(t, repos, user.ID, "Third", "c")
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), spec.AnnouncementPinUpdate{ID: pinned, Pinned: true}))

	// when
	latest, err := repos.Announcement.GetLatest(context.Background())

	// then
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, pinned, latest.ID)
}

func TestAnnouncementDAO_GetLatest_None(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	latest, err := repos.Announcement.GetLatest(context.Background())

	// then
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func TestAnnouncementDAO_SetPinned_Toggle(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createAnnouncement(t, repos, user.ID, "T", "B")

	// when
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), spec.AnnouncementPinUpdate{ID: id, Pinned: true}))
	pinnedRow, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NoError(t, repos.Announcement.SetPinned(context.Background(), spec.AnnouncementPinUpdate{ID: id, Pinned: false}))
	unpinnedRow, err := repos.Announcement.GetByID(context.Background(), id)
	require.NoError(t, err)

	// then
	assert.True(t, pinnedRow.Pinned)
	assert.False(t, unpinnedRow.Pinned)
}

func TestAnnouncementDAO_DeleteWithMedia_ReturnsEveryCommentMediaPath(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	annID := createAnnouncement(t, repos, author.ID, "T", "B")
	target := spec.CommentTarget{Kind: spec.CommentKindAnnouncement, ID: annID}
	firstComment := createComment(t, repos, target, author.ID, nil, "first")
	secondComment := createComment(t, repos, target, commenter.ID, nil, "second")
	addCommentMedia(t, repos, spec.CommentKindAnnouncement, firstComment, "/uploads/announcements/one.png", "/uploads/announcements/one-thumb.png")
	addCommentMedia(t, repos, spec.CommentKindAnnouncement, firstComment, "/uploads/announcements/two.mp4", "")
	addCommentMedia(t, repos, spec.CommentKindAnnouncement, secondComment, "/uploads/announcements/three.png", "/uploads/announcements/three-thumb.png")

	otherAnnID := createAnnouncement(t, repos, author.ID, "Other", "B")
	otherComment := createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindAnnouncement, ID: otherAnnID}, author.ID, nil, "elsewhere")
	addCommentMedia(t, repos, spec.CommentKindAnnouncement, otherComment, "/uploads/announcements/untouched.png", "")

	// when
	paths, err := repos.Announcement.DeleteWithMedia(context.Background(), annID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/announcements/one.png",
		"/uploads/announcements/one-thumb.png",
		"/uploads/announcements/two.mp4",
		"/uploads/announcements/three.png",
		"/uploads/announcements/three-thumb.png",
	}, paths)

	row, err := repos.Announcement.GetByID(context.Background(), annID)
	require.NoError(t, err)
	assert.Nil(t, row)
}
