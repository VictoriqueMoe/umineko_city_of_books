package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createOC(t *testing.T, repos *repository.Repositories, userID uuid.UUID, name, series, customSeries string) uuid.UUID {
	t.Helper()
	created, err := repos.OC.Create(context.Background(), repository.NewOC{UserID: userID, Name: name, Description: "desc", Series: series, CustomSeriesName: customSeries})
	require.NoError(t, err)
	return created.ID
}

func TestOCDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.OC.Create(context.Background(), repository.NewOC{UserID: user.ID, Name: "Linda", Description: "the OC bio", Series: "umineko"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), created.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Linda", row.Name)
	assert.Equal(t, "umineko", row.Series)
	assert.Equal(t, "the OC bio", row.Description)
	assert.Equal(t, user.ID, row.UserID)
}

func TestOCDAO_CreateCustomSeries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.OC.Create(context.Background(), repository.NewOC{UserID: user.ID, Name: "Linda", Series: "custom", CustomSeriesName: "Higanbana"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), created.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "custom", row.Series)
	assert.Equal(t, "Higanbana", row.CustomSeriesName)
}

func TestOCDAO_HasOC_CaseInsensitive(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestOCDAO_Update_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Update(context.Background(), repository.OCUpdate{ID: id, UserID: user.ID, Name: "Linda Renamed", Description: "new bio", Series: "ciconia"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Linda Renamed", row.Name)
	assert.Equal(t, "new bio", row.Description)
	assert.Equal(t, "ciconia", row.Series)
}

func TestOCDAO_Update_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Update(context.Background(), repository.OCUpdate{ID: id, UserID: stranger.ID, Name: "Hijacked", Series: "umineko"})

	// then
	require.Error(t, err)
}

func TestOCDAO_Update_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Update(context.Background(), repository.OCUpdate{ID: id, UserID: admin.ID, Name: "Modded", Series: "umineko", AsAdmin: true})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "Modded", row.Name)
}

func TestOCDAO_UpdateImage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestOCDAO_Delete_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_Delete_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.Delete(context.Background(), id, stranger.ID)

	// then
	require.Error(t, err)
}

func TestOCDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_DeleteOC_ReturnsImageGalleryAndCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	require.NoError(t, repos.OC.UpdateImage(context.Background(), id, "/uploads/ocs/portrait.png", "/uploads/ocs/portrait_thumb.png"))
	_, err := repos.OC.AddGalleryImage(context.Background(), id, "/uploads/ocs/gallery.png", "/uploads/ocs/gallery_thumb.png", "First", 0)
	require.NoError(t, err)
	_, err = repos.OC.AddGalleryImage(context.Background(), id, "/uploads/ocs/gallery_two.png", "", "Second", 1)
	require.NoError(t, err)
	comment, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "nice")
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), comment.ID, "/uploads/ocs/comment.png", "image", "/uploads/ocs/comment_thumb.png", 0)
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), comment.ID, "/uploads/ocs/comment_two.gif", "image", "", 1)
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteOC(context.Background(), repository.OCDeletion{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/ocs/portrait.png",
		"/uploads/ocs/portrait_thumb.png",
		"/uploads/ocs/gallery.png",
		"/uploads/ocs/gallery_thumb.png",
		"/uploads/ocs/gallery_two.png",
		"/uploads/ocs/comment.png",
		"/uploads/ocs/comment_thumb.png",
		"/uploads/ocs/comment_two.gif",
	}, paths)
	row, err := repos.OC.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_DeleteOC_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")
	require.NoError(t, repos.OC.UpdateImage(context.Background(), id, "/uploads/ocs/portrait.png", ""))

	// when
	paths, err := repos.OC.DeleteOC(context.Background(), repository.OCDeletion{ID: id, UserID: stranger.ID})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	row, err := repos.OC.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.NotNil(t, row)
}

func TestOCDAO_DeleteOC_AsAdmin_ReturnsGalleryPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")
	_, err := repos.OC.AddGalleryImage(context.Background(), id, "/uploads/ocs/mod.png", "/uploads/ocs/mod_thumb.png", "", 0)
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteOC(context.Background(), repository.OCDeletion{ID: id, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/ocs/mod.png", "/uploads/ocs/mod_thumb.png"}, paths)
	row, err := repos.OC.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_DeleteCommentWithMedia_ReturnsOnlyThatCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	target, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "target")
	require.NoError(t, err)
	other, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "other")
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), target.ID, "/uploads/ocs/target.png", "image", "/uploads/ocs/target_thumb.png", 0)
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), target.ID, "/uploads/ocs/target_two.gif", "image", "", 1)
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), other.ID, "/uploads/ocs/other.png", "image", "", 0)
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteCommentWithMedia(context.Background(), repository.OCCommentDeletion{CommentID: target.ID, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/ocs/target.png",
		"/uploads/ocs/target_thumb.png",
		"/uploads/ocs/target_two.gif",
	}, paths)
	remaining, err := repos.OC.GetCommentMedia(context.Background(), other.ID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, "/uploads/ocs/other.png", remaining[0].MediaURL)
}

func TestOCDAO_DeleteCommentWithMedia_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	comment, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "spam")
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), comment.ID, "/uploads/ocs/spam.png", "image", "/uploads/ocs/spam_thumb.png", 0)
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteCommentWithMedia(context.Background(), repository.OCCommentDeletion{CommentID: comment.ID, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/ocs/spam.png", "/uploads/ocs/spam_thumb.png"}, paths)
	_, total, err := repos.OC.GetComments(context.Background(), id, user.ID, 20, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestOCDAO_DeleteCommentWithMedia_AsAdminWritesAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	comment, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "spam")
	require.NoError(t, err)

	// when
	_, err = repos.OC.DeleteCommentWithMedia(context.Background(), repository.OCCommentDeletion{CommentID: comment.ID, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), repository.AuditActionOCCommentDeleteAdmin, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, moderator.ID, entries[0].ActorID)
	assert.Equal(t, repository.AuditTargetOCComment, entries[0].TargetType)
	assert.Equal(t, comment.ID.String(), entries[0].TargetID)
	require.NotNil(t, entries[0].SubjectID)
	assert.Equal(t, user.ID, *entries[0].SubjectID)
	assert.Empty(t, entries[0].Details)
}

func TestOCDAO_DeleteCommentWithMedia_AsOwnerWritesAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	comment, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "mine")
	require.NoError(t, err)

	// when
	_, err = repos.OC.DeleteCommentWithMedia(context.Background(), repository.OCCommentDeletion{CommentID: comment.ID, UserID: user.ID})

	// then
	require.NoError(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), repository.AuditActionOCCommentDelete, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, user.ID, entries[0].ActorID)
	require.NotNil(t, entries[0].SubjectID)
	assert.Equal(t, user.ID, *entries[0].SubjectID)
}

func TestOCDAO_DeleteCommentWithMedia_NotOwnedWritesNoAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	comment, err := repos.OC.CreateComment(context.Background(), id, nil, user.ID, "mine")
	require.NoError(t, err)

	// when
	_, err = repos.OC.DeleteCommentWithMedia(context.Background(), repository.OCCommentDeletion{CommentID: comment.ID, UserID: stranger.ID})

	// then
	require.Error(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), repository.AuditActionOCCommentDeleteAdmin, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
	_, total, err := repos.OC.GetComments(context.Background(), id, user.ID, 20, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestOCDAO_List_FiltersBySeries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestOCDAO_List_FiltersByCustomSeriesName(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestOCDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
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

func TestOCDAO_ListSummariesByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestOCDAO_GalleryRoundTrip(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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
	require.NoError(t, repos.OC.UpdateGalleryImage(context.Background(), first, id, new("Updated"), nil))

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

func TestOCDAO_VoteRoundTrip(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
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

func TestOCDAO_FavouriteRoundTrip(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	fan := daotest.CreateUser(t, repos)
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

func TestOCDAO_CommentsRoundTrip(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")

	// when (create)
	comment, err := repos.OC.CreateComment(context.Background(), id, nil, commenter.ID, "great oc")
	require.NoError(t, err)
	commentID := comment.ID

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
