package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createOC(t *testing.T, repos *repository.Repositories, userID uuid.UUID, name, series, customSeries string) uuid.UUID {
	t.Helper()
	created, err := repos.OC.Create(context.Background(), spec.NewOC{UserID: userID, Name: name, Description: "desc", Series: series, CustomSeriesName: customSeries})
	require.NoError(t, err)
	return created.ID
}

func TestOCDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.OC.Create(context.Background(), spec.NewOC{UserID: user.ID, Name: "Linda", Description: "the OC bio", Series: "umineko"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: created.ID, ViewerID: user.ID})
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
	created, err := repos.OC.Create(context.Background(), spec.NewOC{UserID: user.ID, Name: "Linda", Series: "custom", CustomSeriesName: "Higanbana"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: created.ID, ViewerID: user.ID})
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
	got1, err := repos.OC.HasOC(context.Background(), spec.OCNameLookup{UserID: user.ID, Name: "linda"})
	require.NoError(t, err)
	got2, err := repos.OC.HasOC(context.Background(), spec.OCNameLookup{UserID: user.ID, Name: "Other"})
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
	err := repos.OC.Update(context.Background(), spec.OCUpdate{ID: id, UserID: user.ID, Name: "Linda Renamed", Description: "new bio", Series: "ciconia"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: user.ID})
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
	err := repos.OC.Update(context.Background(), spec.OCUpdate{ID: id, UserID: stranger.ID, Name: "Hijacked", Series: "umineko"})

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
	err := repos.OC.Update(context.Background(), spec.OCUpdate{ID: id, UserID: admin.ID, Name: "Modded", Series: "umineko", AsAdmin: true})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	assert.Equal(t, "Modded", row.Name)
}

func TestOCDAO_UpdateImage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")

	// when
	err := repos.OC.UpdateImage(context.Background(), spec.OCImageUpdate{ID: id, ImageURL: "/uploads/ocs/x.png", ThumbnailURL: "/uploads/ocs/x_thumb.png"})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: user.ID})
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
	err := repos.OC.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: user.ID})
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
	err := repos.OC.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: stranger.ID})

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
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_DeleteOC_ReturnsImageGalleryAndCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	require.NoError(t, repos.OC.UpdateImage(context.Background(), spec.OCImageUpdate{ID: id, ImageURL: "/uploads/ocs/portrait.png", ThumbnailURL: "/uploads/ocs/portrait_thumb.png"}))
	_, err := repos.OC.AddGalleryImage(context.Background(), spec.NewOCGalleryImage{OCID: id, ImageURL: "/uploads/ocs/gallery.png", ThumbnailURL: "/uploads/ocs/gallery_thumb.png", Caption: "First", SortOrder: 0})
	require.NoError(t, err)
	_, err = repos.OC.AddGalleryImage(context.Background(), spec.NewOCGalleryImage{OCID: id, ImageURL: "/uploads/ocs/gallery_two.png", Caption: "Second", SortOrder: 1})
	require.NoError(t, err)
	comment, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "nice"})
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: comment.ID, MediaURL: "/uploads/ocs/comment.png", MediaType: "image", ThumbnailURL: "/uploads/ocs/comment_thumb.png", Filename: "comment.png", SortOrder: 0})
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: comment.ID, MediaURL: "/uploads/ocs/comment_two.gif", MediaType: "image", Filename: "comment_two.gif", SortOrder: 1})
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteOC(context.Background(), spec.OCDeletion{ID: id, UserID: user.ID})

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
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_DeleteOC_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")
	require.NoError(t, repos.OC.UpdateImage(context.Background(), spec.OCImageUpdate{ID: id, ImageURL: "/uploads/ocs/portrait.png"}))

	// when
	paths, err := repos.OC.DeleteOC(context.Background(), spec.OCDeletion{ID: id, UserID: stranger.ID})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	assert.NotNil(t, row)
}

func TestOCDAO_DeleteOC_AsAdmin_ReturnsGalleryPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createOC(t, repos, owner.ID, "Linda", "umineko", "")
	_, err := repos.OC.AddGalleryImage(context.Background(), spec.NewOCGalleryImage{OCID: id, ImageURL: "/uploads/ocs/mod.png", ThumbnailURL: "/uploads/ocs/mod_thumb.png", SortOrder: 0})
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteOC(context.Background(), spec.OCDeletion{ID: id, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/ocs/mod.png", "/uploads/ocs/mod_thumb.png"}, paths)
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestOCDAO_DeleteCommentWithMedia_ReturnsOnlyThatCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	target, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "target"})
	require.NoError(t, err)
	other, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "other"})
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: target.ID, MediaURL: "/uploads/ocs/target.png", MediaType: "image", ThumbnailURL: "/uploads/ocs/target_thumb.png", Filename: "target.png", SortOrder: 0})
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: target.ID, MediaURL: "/uploads/ocs/target_two.gif", MediaType: "image", Filename: "target_two.gif", SortOrder: 1})
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: other.ID, MediaURL: "/uploads/ocs/other.png", MediaType: "image", Filename: "other.png", SortOrder: 0})
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteCommentWithMedia(context.Background(), spec.CommentDeletion{CommentID: target.ID, UserID: user.ID})

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
	comment, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "spam"})
	require.NoError(t, err)
	_, err = repos.OC.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: comment.ID, MediaURL: "/uploads/ocs/spam.png", MediaType: "image", ThumbnailURL: "/uploads/ocs/spam_thumb.png", Filename: "spam.png", SortOrder: 0})
	require.NoError(t, err)

	// when
	paths, err := repos.OC.DeleteCommentWithMedia(context.Background(), spec.CommentDeletion{CommentID: comment.ID, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/ocs/spam.png", "/uploads/ocs/spam_thumb.png"}, paths)
	_, total, err := repos.OC.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: user.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestOCDAO_DeleteCommentWithMedia_AsAdminWritesAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createOC(t, repos, user.ID, "Linda", "umineko", "")
	comment, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "spam"})
	require.NoError(t, err)

	// when
	_, err = repos.OC.DeleteCommentWithMedia(context.Background(), spec.CommentDeletion{CommentID: comment.ID, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: audit.ActionOCCommentDeleteAdmin, Page: bounds.NewPage(10, 0)})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, moderator.ID, entries[0].ActorID)
	assert.Equal(t, audit.TargetOCComment, entries[0].TargetType)
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
	comment, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "mine"})
	require.NoError(t, err)

	// when
	_, err = repos.OC.DeleteCommentWithMedia(context.Background(), spec.CommentDeletion{CommentID: comment.ID, UserID: user.ID})

	// then
	require.NoError(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: audit.ActionOCCommentDelete, Page: bounds.NewPage(10, 0)})
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
	comment, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: user.ID, Body: "mine"})
	require.NoError(t, err)

	// when
	_, err = repos.OC.DeleteCommentWithMedia(context.Background(), spec.CommentDeletion{CommentID: comment.ID, UserID: stranger.ID})

	// then
	require.Error(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: audit.ActionOCCommentDeleteAdmin, Page: bounds.NewPage(10, 0)})
	require.NoError(t, err)
	assert.Empty(t, entries)
	_, total, err := repos.OC.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: user.ID, Limit: 20, Offset: 0})
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
	rows, total, err := repos.OC.List(context.Background(), spec.OCListFilter{ViewerID: uuid.Nil, Sort: "new", CrackOCsOnly: false, Series: "umineko", OwnerID: uuid.Nil, Limit: 20, Offset: 0})

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
	rows, total, err := repos.OC.List(context.Background(), spec.OCListFilter{ViewerID: uuid.Nil, Sort: "new", CrackOCsOnly: false, Series: "custom", CustomSeriesName: "higanbana", OwnerID: uuid.Nil, Limit: 20, Offset: 0})

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
	rows, total, err := repos.OC.ListByUser(context.Background(), spec.OCUserListFilter{UserID: owner.ID, ViewerID: owner.ID, Limit: 20, Offset: 0})

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
	first, err := repos.OC.AddGalleryImage(context.Background(), spec.NewOCGalleryImage{OCID: id, ImageURL: "/uploads/ocs/a.png", Caption: "First", SortOrder: 0})
	require.NoError(t, err)
	_, err = repos.OC.AddGalleryImage(context.Background(), spec.NewOCGalleryImage{OCID: id, ImageURL: "/uploads/ocs/b.png", Caption: "Second", SortOrder: 1})
	require.NoError(t, err)

	images, err := repos.OC.GetGallery(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Len(t, images, 2)

	// when (update first caption)
	require.NoError(t, repos.OC.UpdateGalleryImage(context.Background(), spec.OCGalleryImageUpdate{ID: first, OCID: id, Caption: new("Updated")}))

	// then
	got, err := repos.OC.GetGallery(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Updated", got[0].Caption)

	// when (delete second)
	require.NoError(t, repos.OC.DeleteGalleryImage(context.Background(), spec.MediaDeletion{ID: got[1].ID, TargetID: id}))

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
	require.NoError(t, repos.OC.Vote(context.Background(), spec.Vote{UserID: voter.ID, TargetID: id, Value: 1}))
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: voter.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, row.VoteScore)
	assert.Equal(t, 1, row.UserVote)

	// when (downvote replaces)
	require.NoError(t, repos.OC.Vote(context.Background(), spec.Vote{UserID: voter.ID, TargetID: id, Value: -1}))
	row, err = repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: voter.ID})
	require.NoError(t, err)
	assert.Equal(t, -1, row.VoteScore)

	// when (clear)
	require.NoError(t, repos.OC.Vote(context.Background(), spec.Vote{UserID: voter.ID, TargetID: id, Value: 0}))
	row, err = repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: voter.ID})
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
	require.NoError(t, repos.OC.Favourite(context.Background(), spec.Like{UserID: fan.ID, TargetID: id}))
	row, err := repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: fan.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, row.FavouriteCount)
	assert.True(t, row.UserFavourited)

	// when (idempotent)
	require.NoError(t, repos.OC.Favourite(context.Background(), spec.Like{UserID: fan.ID, TargetID: id}))
	row, err = repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: fan.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, row.FavouriteCount)

	// when (unfavourite)
	require.NoError(t, repos.OC.Unfavourite(context.Background(), spec.Like{UserID: fan.ID, TargetID: id}))
	row, err = repos.OC.GetByID(context.Background(), spec.OCByID{ID: id, ViewerID: fan.ID})
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
	comment, err := repos.Comments.ByID[string(mention.KindOCComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{TargetID: id, UserID: commenter.ID, Body: "great oc"})
	require.NoError(t, err)
	commentID := comment.ID

	rows, total, err := repos.OC.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "great oc", rows[0].Body)

	// when (update)
	require.NoError(t, repos.OC.UpdateComment(context.Background(), spec.CommentUpdate{CommentID: commentID, UserID: commenter.ID, Body: "edited"}))
	rows, _, err = repos.OC.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, "edited", rows[0].Body)

	// when (like)
	require.NoError(t, repos.OC.LikeComment(context.Background(), spec.CommentLike{UserID: commenter.ID, CommentID: commentID}))
	rows, _, err = repos.OC.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, rows[0].LikeCount)
	assert.True(t, rows[0].UserLiked)

	// when (delete)
	require.NoError(t, repos.OC.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: commenter.ID}))
	_, total, err = repos.OC.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}
