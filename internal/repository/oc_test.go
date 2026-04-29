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

func createOC(t *testing.T, repos *repository.Repositories, userID uuid.UUID, name, series, customSeries string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.OC.Create(context.Background(), id, userID, name, "desc", series, customSeries))
	return id
}

func TestOCRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.OC.Create(context.Background(), id, user.ID, "Linda", "the OC bio", "umineko", "")

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Linda", row.Name)
	assert.Equal(t, "umineko", row.Series)
	assert.Equal(t, "the OC bio", row.Description)
	assert.Equal(t, user.ID, row.UserID)
}

func TestOCRepository_CreateCustomSeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.OC.Create(context.Background(), id, user.ID, "Linda", "", "custom", "Higanbana")

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "custom", row.Series)
	assert.Equal(t, "Higanbana", row.CustomSeriesName)
}

func TestOCRepository_HasOC_CaseInsensitive(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	got1, err := repos.OC.HasOC(context.Background(), user.ID, "linda")
	require.NoError(t, err)
	got2, err := repos.OC.HasOC(context.Background(), user.ID, "Other")
	require.NoError(t, err)

	// then
	assert.True(t, got1)
	assert.False(t, got2)
}

func TestOCRepository_Update_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Update(context.Background(), id, user.ID, "Linda Renamed", "new bio", "ciconia", "", false)

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Linda Renamed", row.Name)
	assert.Equal(t, "new bio", row.Description)
	assert.Equal(t, "ciconia", row.Series)
}

func TestOCRepository_Update_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Update(context.Background(), id, stranger.ID, "Hijacked", "", "umineko", "", false)

	// then
	require.Error(t, err)
}

func TestOCRepository_Update_AsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	admin := repotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Update(context.Background(), id, admin.ID, "Modded", "", "umineko", "", true)

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "Modded", row.Name)
}

func TestOCRepository_UpdateImage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.UpdateImage(context.Background(), id, "/uploads/ocs/x.png", "/uploads/ocs/x_thumb.png")

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/uploads/ocs/x.png", row.ImageURL)
	assert.Equal(t, "/uploads/ocs/x_thumb.png", row.ThumbnailURL)
}

func TestOCRepository_Delete_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCRepository_Delete_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Delete(context.Background(), id, stranger.ID)

	// then
	require.Error(t, err)
}

func TestOCRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCRepository_List_FiltersBySeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createOC(t, repos, user.ID, "Linda", "umineko", "")
	createOC(t, repos, user.ID, "Rena", "higurashi", "")

	// when
	rows, total, err := repos.OC.List(context.Background(), uuid.Nil, "new", false, "umineko", "", uuid.Nil, 20, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Linda", rows[0].Name)
}

func TestOCRepository_List_FiltersByCustomSeriesName(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createOC(t, repos, user.ID, "A", "custom", "Higanbana")
	createOC(t, repos, user.ID, "B", "custom", "Roseguns")

	// when
	rows, total, err := repos.OC.List(context.Background(), uuid.Nil, "new", false, "custom", "higanbana", uuid.Nil, 20, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "A", rows[0].Name)
}

func TestOCRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createOC(t, repos, owner.ID, "Linda", "umineko", "")
	createOC(t, repos, owner.ID, "Beatrice", "umineko", "")
	createOC(t, repos, other.ID, "Rena", "higurashi", "")

	// when
	rows, total, err := repos.OC.ListByUser(context.Background(), owner.ID, owner.ID, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestOCRepository_ListSummariesByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createOC(t, repos, user.ID, "Zelda", "umineko", "")
	createOC(t, repos, user.ID, "Aria", "higurashi", "")

	// when
	summaries, err := repos.OC.ListSummariesByUser(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "Aria", summaries[0].Name)
	assert.Equal(t, "Zelda", summaries[1].Name)
}

func TestOCRepository_GalleryRoundTrip(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	first, err := repos.OC.AddGalleryImage(context.Background(), id, "/uploads/ocs/a.png", "", "First", 0)
	require.NoError(t, err)
	_, err = repos.OC.AddGalleryImage(context.Background(), id, "/uploads/ocs/b.png", "", "Second", 1)
	require.NoError(t, err)

	images, err := repos.OC.GetGallery(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Len(t, images, 2)

	// when (update first caption)
	caption := "Updated"
	require.NoError(t, repos.OC.UpdateGalleryImage(context.Background(), first, id, &caption, nil))

	// then
	got, err := repos.OC.GetGallery(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Updated", got[0].Caption)

	// when (delete second)
	require.NoError(t, repos.OC.DeleteGalleryImage(context.Background(), got[1].ID, id))

	got, err = repos.OC.GetGallery(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestOCRepository_VoteRoundTrip(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when (upvote)
	require.NoError(t, repos.OC.Vote(context.Background(), voter.ID, id, 1))
	row, err := repos.OC.GetByID(context.Background(), id, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.VoteScore)
	assert.Equal(t, 1, row.UserVote)

	// when (downvote replaces)
	require.NoError(t, repos.OC.Vote(context.Background(), voter.ID, id, -1))
	row, err = repos.OC.GetByID(context.Background(), id, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, row.VoteScore)

	// when (clear)
	require.NoError(t, repos.OC.Vote(context.Background(), voter.ID, id, 0))
	row, err = repos.OC.GetByID(context.Background(), id, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.VoteScore)
}

func TestOCRepository_FavouriteRoundTrip(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	fan := repotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when (favourite)
	require.NoError(t, repos.OC.Favourite(context.Background(), fan.ID, id))
	row, err := repos.OC.GetByID(context.Background(), id, fan.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.FavouriteCount)
	assert.True(t, row.UserFavourited)

	// when (idempotent)
	require.NoError(t, repos.OC.Favourite(context.Background(), fan.ID, id))
	row, err = repos.OC.GetByID(context.Background(), id, fan.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.FavouriteCount)

	// when (unfavourite)
	require.NoError(t, repos.OC.Unfavourite(context.Background(), fan.ID, id))
	row, err = repos.OC.GetByID(context.Background(), id, fan.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.FavouriteCount)
	assert.False(t, row.UserFavourited)
}

func TestOCRepository_CommentsRoundTrip(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when (create)
	commentID := uuid.New()
	require.NoError(t, repos.OC.CreateComment(context.Background(), commentID, id, nil, commenter.ID, "great oc"))

	rows, total, err := repos.OC.GetComments(context.Background(), id, commenter.ID, 20, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "great oc", rows[0].Body)

	// when (update)
	require.NoError(t, repos.OC.UpdateComment(context.Background(), commentID, commenter.ID, "edited"))
	rows, _, err = repos.OC.GetComments(context.Background(), id, commenter.ID, 20, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, "edited", rows[0].Body)

	// when (like)
	require.NoError(t, repos.OC.LikeComment(context.Background(), commenter.ID, commentID))
	rows, _, err = repos.OC.GetComments(context.Background(), id, commenter.ID, 20, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, rows[0].LikeCount)
	assert.True(t, rows[0].UserLiked)

	// when (delete)
	require.NoError(t, repos.OC.DeleteComment(context.Background(), commentID, commenter.ID))
	_, total, err = repos.OC.GetComments(context.Background(), id, commenter.ID, 20, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}
