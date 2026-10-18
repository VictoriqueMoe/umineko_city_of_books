package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
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
	commentID := createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindOC, ID: id}, user.ID, nil, "nice")
	addCommentMedia(t, repos, spec.CommentKindOC, commentID, "/uploads/ocs/comment.png", "/uploads/ocs/comment_thumb.png")
	addCommentMedia(t, repos, spec.CommentKindOC, commentID, "/uploads/ocs/comment_two.gif", "")

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

func ocSetCreatedAt(t *testing.T, repos *repository.Repositories, id uuid.UUID, createdAt string) {
	t.Helper()

	_, err := repos.DB().ExecContext(context.Background(), "UPDATE ocs SET created_at = $1 WHERE id = $2", createdAt, id)
	require.NoError(t, err)
}

func ocCastVotes(t *testing.T, repos *repository.Repositories, ocID uuid.UUID, values ...int) {
	t.Helper()

	for _, value := range values {
		voter := daotest.CreateUser(t, repos)
		require.NoError(t, repos.OC.Vote(context.Background(), spec.Vote{UserID: voter.ID, TargetID: ocID, Value: value}))
	}
}

func ocFavourite(t *testing.T, repos *repository.Repositories, ocID uuid.UUID) uuid.UUID {
	t.Helper()

	fan := daotest.CreateUser(t, repos)
	require.NoError(t, repos.OC.Favourite(context.Background(), spec.Like{UserID: fan.ID, TargetID: ocID}))

	return fan.ID
}

func ocComment(t *testing.T, repos *repository.Repositories, ocID, userID uuid.UUID) {
	t.Helper()

	createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindOC, ID: ocID}, userID, nil, "lovely")
}

func TestOCDAO_List_SortsByEngagement(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	flipper := daotest.CreateUser(t, repos)
	oldest := createOC(t, repos, owner.ID, "Oldest", "umineko", "")
	middle := createOC(t, repos, owner.ID, "Middle", "umineko", "")
	recent := createOC(t, repos, owner.ID, "Recent", "umineko", "")
	newest := createOC(t, repos, owner.ID, "Newest", "umineko", "")

	ocSetCreatedAt(t, repos, oldest, "2020-01-01 00:00:00")
	ocSetCreatedAt(t, repos, middle, "2021-01-01 00:00:00")
	ocSetCreatedAt(t, repos, recent, "2022-01-01 00:00:00")
	ocSetCreatedAt(t, repos, newest, "2023-01-01 00:00:00")

	ocCastVotes(t, repos, oldest, -1, -1, -1, -1)
	ocCastVotes(t, repos, middle, 1, 1)
	require.NoError(t, repos.OC.Vote(ctx, spec.Vote{UserID: flipper.ID, TargetID: newest, Value: 1}))
	require.NoError(t, repos.OC.Vote(ctx, spec.Vote{UserID: flipper.ID, TargetID: newest, Value: -1}))

	ocFavourite(t, repos, middle)
	leaver := ocFavourite(t, repos, middle)
	require.NoError(t, repos.OC.Unfavourite(ctx, spec.Like{UserID: leaver, TargetID: middle}))
	ocFavourite(t, repos, recent)
	ocFavourite(t, repos, recent)

	ocComment(t, repos, recent, owner.ID)
	ocComment(t, repos, newest, owner.ID)
	ocComment(t, repos, newest, owner.ID)

	cases := []struct {
		name      string
		sort      string
		crackOnly bool
		want      []uuid.UUID
	}{
		{name: "top orders by score, ties newest first", sort: "top", want: []uuid.UUID{middle, recent, newest, oldest}},
		{name: "crack orders by lowest score, ties newest first", sort: "crack", want: []uuid.UUID{oldest, newest, recent, middle}},
		{name: "favourites orders by favourite count, ties newest first", sort: "favourites", want: []uuid.UUID{recent, middle, newest, oldest}},
		{name: "comments orders by comment count, ties newest first", sort: "comments", want: []uuid.UUID{newest, recent, middle, oldest}},
		{name: "crack only keeps OCs scored at minus three or below", sort: "new", crackOnly: true, want: []uuid.UUID{oldest}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.OC.List(ctx, spec.OCListFilter{ViewerID: owner.ID, Sort: tc.sort, CrackOCsOnly: tc.crackOnly, Limit: 20, Offset: 0})

			// then
			require.NoError(t, err)
			assert.Equal(t, len(tc.want), total)
			got := make([]uuid.UUID, 0, len(rows))
			for _, row := range rows {
				got = append(got, row.ID)
			}
			assert.Equal(t, tc.want, got)
		})
	}
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

func TestOCDAO_GalleryImageWritesOnAMissingImageReadAsNotFound(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	ocID := createOC(t, repos, user.ID, "Linda", "umineko", "")
	otherOCID := createOC(t, repos, user.ID, "Ange", "umineko", "")
	foreignImage, err := repos.OC.AddGalleryImage(context.Background(), spec.NewOCGalleryImage{OCID: otherOCID, ImageURL: "/uploads/ocs/a.png"})
	require.NoError(t, err)

	cases := []struct {
		name  string
		write func(imageID int64) error
	}{
		{
			name: "update",
			write: func(imageID int64) error {
				return repos.OC.UpdateGalleryImage(context.Background(), spec.OCGalleryImageUpdate{ID: imageID, OCID: ocID, Caption: new("x")})
			},
		},
		{
			name: "delete",
			write: func(imageID int64) error {
				return repos.OC.DeleteGalleryImage(context.Background(), spec.MediaDeletion{ID: imageID, TargetID: ocID})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			missingErr := tc.write(999999)
			foreignErr := tc.write(foreignImage)

			// then
			require.ErrorIs(t, missingErr, dao.ErrNotFound)
			require.ErrorIs(t, foreignErr, dao.ErrNotFound)
		})
	}
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
