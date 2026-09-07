package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createArt(t *testing.T, repos *repository.Repositories, userID uuid.UUID, corner, artType, title string, tags []string, spoiler bool) uuid.UUID {
	t.Helper()
	created, err := repos.Art.CreateWithTags(context.Background(), spec.NewArtWithTags{
		NewArt: spec.NewArt{
			UserID:       userID,
			Corner:       corner,
			ArtType:      artType,
			Title:        title,
			Description:  "desc",
			ImageURL:     "https://example.com/img.png",
			ThumbnailURL: "https://example.com/thumb.png",
			IsSpoiler:    spoiler,
		},
		Tags: tags,
	})
	require.NoError(t, err)
	return created.ID
}

func createGallery(t *testing.T, repos *repository.Repositories, userID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	created, err := repos.Art.CreateGallery(context.Background(), spec.NewGallery{
		UserID:      userID,
		Name:        name,
		Description: "desc",
	})
	require.NoError(t, err)
	return created.ID
}

func createArtComment(t *testing.T, repos *repository.Repositories, artID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Comments.ByID[string(mention.KindArtComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{
		TargetID: artID,
		ParentID: parentID,
		UserID:   userID,
		Body:     body,
	})
	require.NoError(t, err)
	return created.ID
}

func addArtCommentMedia(t *testing.T, repos *repository.Repositories, commentID uuid.UUID, mediaURL, thumbnailURL string) {
	t.Helper()
	_, err := repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{
		TargetID:     commentID,
		MediaURL:     mediaURL,
		MediaType:    "image",
		ThumbnailURL: thumbnailURL,
	})
	require.NoError(t, err)
}

func TestArtDAO_CreateWithTags_GetByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Artist"))

	// when
	id := createArt(t, repos, user.ID, "general", "drawing", "My Art", []string{"tagA", "TagB", "  "}, false)

	// then
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: user.ID})
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

func TestArtDAO_CreateWithTags_SpoilerTrue(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	id := createArt(t, repos, user.ID, "general", "drawing", "S", nil, true)

	// then
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsSpoiler)
}

func TestArtDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: uuid.New(), ViewerID: uuid.New()})

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtDAO_GetTags_TrimsAndLowercases(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", []string{"FooBar", " baz ", ""}, false)

	// when
	tags, err := repos.Art.GetTags(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"foobar", "baz"}, tags)
}

func TestArtDAO_UpdateWithTags_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "Old", []string{"a"}, false)

	// when
	err := repos.Art.UpdateWithTags(context.Background(), spec.ArtUpdateWithTags{
		ArtUpdate: spec.ArtUpdate{
			ID: id, UserID: user.ID, Title: "New Title", Description: "New Desc", IsSpoiler: true,
		},
		Tags: []string{"b", "c"},
	})

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Equal(t, "New Title", row.Title)
	assert.Equal(t, "New Desc", row.Description)
	assert.True(t, row.IsSpoiler)
	tags, err := repos.Art.GetTags(context.Background(), id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"b", "c"}, tags)
}

func TestArtDAO_UpdateWithTags_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.UpdateWithTags(context.Background(), spec.ArtUpdateWithTags{
		ArtUpdate: spec.ArtUpdate{
			ID: id, UserID: other.ID, Title: "Hack", Description: "Hack",
		},
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_UpdateWithTags_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.UpdateWithTags(context.Background(), spec.ArtUpdateWithTags{
		ArtUpdate: spec.ArtUpdate{
			ID: id, UserID: admin.ID, Title: "Admin Title", Description: "d", AsAdmin: true,
		},
		Tags: []string{"x"},
	})

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	assert.Equal(t, "Admin Title", row.Title)
}

func TestArtDAO_Delete_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtDAO_Delete_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: other.ID})

	// then
	require.Error(t, err)
}

func TestArtDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtDAO_DeleteWithImage_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	paths, err := repos.Art.DeleteWithImage(context.Background(), spec.ArtDelete{
		ID:     id,
		UserID: user.ID,
		Audit: audit.NewEntry{
			ActorID:    user.ID,
			Action:     audit.ActionArtDelete,
			TargetType: audit.TargetArt,
			TargetID:   id.String(),
			SubjectID:  user.ID,
		},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/img.png", "https://example.com/thumb.png"}, paths)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtDAO_DeleteWithImage_ReturnsArtAndCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)
	commentID := createArtComment(t, repos, id, user.ID, nil, "hi")
	addArtCommentMedia(t, repos, commentID, "https://example.com/c1.png", "https://example.com/c1-thumb.png")
	replyID := createArtComment(t, repos, id, user.ID, &commentID, "reply")
	addArtCommentMedia(t, repos, replyID, "https://example.com/c2.png", "")

	// when
	paths, err := repos.Art.DeleteWithImage(context.Background(), spec.ArtDelete{
		ID:     id,
		UserID: user.ID,
		Audit: audit.NewEntry{
			ActorID:    user.ID,
			Action:     audit.ActionArtDelete,
			TargetType: audit.TargetArt,
			TargetID:   id.String(),
			SubjectID:  user.ID,
		},
	})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"https://example.com/img.png",
		"https://example.com/thumb.png",
		"https://example.com/c1.png",
		"https://example.com/c1-thumb.png",
		"https://example.com/c2.png",
	}, paths)
}

func TestArtDAO_DeleteWithImage_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	paths, err := repos.Art.DeleteWithImage(context.Background(), spec.ArtDelete{
		ID:     id,
		UserID: other.ID,
		Audit: audit.NewEntry{
			ActorID:    other.ID,
			Action:     audit.ActionArtDeleteAdmin,
			TargetType: audit.TargetArt,
			TargetID:   id.String(),
			SubjectID:  owner.ID,
		},
	})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
}

func TestArtDAO_DeleteWithImage_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	paths, err := repos.Art.DeleteWithImage(context.Background(), spec.ArtDelete{
		ID:      id,
		UserID:  admin.ID,
		AsAdmin: true,
		Audit: audit.NewEntry{
			ActorID:    admin.ID,
			Action:     audit.ActionArtDeleteAdmin,
			TargetType: audit.TargetArt,
			TargetID:   id.String(),
			SubjectID:  owner.ID,
		},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/img.png", "https://example.com/thumb.png"}, paths)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: owner.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtDAO_DeleteWithImage_UnknownArt_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	unknownID := uuid.New()

	// when
	_, err := repos.Art.DeleteWithImage(context.Background(), spec.ArtDelete{
		ID:     unknownID,
		UserID: user.ID,
		Audit: audit.NewEntry{
			ActorID:    user.ID,
			Action:     audit.ActionArtDelete,
			TargetType: audit.TargetArt,
			TargetID:   unknownID.String(),
			SubjectID:  user.ID,
		},
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_GetArtAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	authorID, err := repos.Art.GetArtAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, authorID)
}

func TestArtDAO_GetArtAuthorID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Art.GetArtAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtDAO_GetImageURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	url, err := repos.Art.GetImageURL(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/img.png", url)
}

func TestArtDAO_GetImageURL_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Art.GetImageURL(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtDAO_LikeAndUnlike(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: liker.ID, TargetID: id}))
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: liker.ID, TargetID: id}))

	// then
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: liker.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, row.LikeCount)
	assert.True(t, row.UserLiked)

	require.NoError(t, repos.Art.Unlike(context.Background(), spec.Like{UserID: liker.ID, TargetID: id}))
	row, err = repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: liker.ID})
	require.NoError(t, err)
	assert.Equal(t, 0, row.LikeCount)
	assert.False(t, row.UserLiked)
}

func TestArtDAO_GetLikedBy(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: a.ID, TargetID: id}))
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: b.ID, TargetID: id}))

	// when
	users, err := repos.Art.GetLikedBy(context.Background(), spec.LikedByQuery{TargetID: id, ExcludeUserIDs: nil})

	// then
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestArtDAO_GetLikedBy_ExcludesUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: a.ID, TargetID: id}))
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: b.ID, TargetID: id}))

	// when
	users, err := repos.Art.GetLikedBy(context.Background(), spec.LikedByQuery{TargetID: id, ExcludeUserIDs: []uuid.UUID{a.ID}})

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, b.ID, users[0].ID)
}

func TestArtDAO_RecordView(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	firstNew, err := repos.Art.RecordView(context.Background(), spec.ViewRecord{TargetID: id, ViewerHash: "hash1"})
	require.NoError(t, err)
	dupNew, err := repos.Art.RecordView(context.Background(), spec.ViewRecord{TargetID: id, ViewerHash: "hash1"})
	require.NoError(t, err)
	secondNew, err := repos.Art.RecordView(context.Background(), spec.ViewRecord{TargetID: id, ViewerHash: "hash2"})
	require.NoError(t, err)

	// then
	assert.True(t, firstNew)
	assert.False(t, dupNew)
	assert.True(t, secondNew)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: id, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, row.ViewCount)
}

func TestArtDAO_GetTagsBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestArtDAO_GetTagsBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	result, err := repos.Art.GetTagsBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArtDAO_GetPopularTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"common", "rare"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", []string{"common"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "C", []string{"common"}, false)

	// when
	tags, err := repos.Art.GetPopularTags(context.Background(), spec.PopularTagFilter{Corner: "general", Limit: 10})

	// then
	require.NoError(t, err)
	require.NotEmpty(t, tags)
	assert.Equal(t, "common", tags[0].Tag)
	assert.Equal(t, 3, tags[0].Count)
}

func TestArtDAO_GetPopularTags_NoCornerFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"foo"}, false)
	createArt(t, repos, user.ID, "other", "drawing", "B", []string{"foo"}, false)

	// when
	tags, err := repos.Art.GetPopularTags(context.Background(), spec.PopularTagFilter{Corner: "", Limit: 10})

	// then
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, 2, tags[0].Count)
}

func TestArtDAO_GetCornerCounts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestArtDAO_CountUserArtToday(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, other.ID, "general", "drawing", "C", nil, false)

	// when
	count, err := repos.Art.CountUserArtToday(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestArtDAO_ListAll_Basic(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, user.ID, "alt", "drawing", "C", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, arts, 2)
}

func TestArtDAO_ListAll_FilterByArtType(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "photo", "B", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", ArtType: "photo", Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, "B", arts[0].Title)
}

func TestArtDAO_ListAll_Search(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "UniqueApple", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "Banana", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Search: "Apple", Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, "UniqueApple", arts[0].Title)
}

func TestArtDAO_ListAll_FilterByTag(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"red"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", []string{"blue"}, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Tag: "red", Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, "A", arts[0].Title)
}

func TestArtDAO_ListAll_SortPopular(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	idA := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	idB := createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	require.NoError(t, repos.Art.Like(context.Background(), spec.Like{UserID: liker.ID, TargetID: idB}))

	// when
	arts, _, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Sort: "popular", Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	require.Len(t, arts, 2)
	assert.Equal(t, idB, arts[0].ID)
	assert.Equal(t, idA, arts[1].ID)
}

func TestArtDAO_ListAll_SortViews(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	idA := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	idB := createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	_, err := repos.Art.RecordView(context.Background(), spec.ViewRecord{TargetID: idB, ViewerHash: "h1"})
	require.NoError(t, err)
	_, err = repos.Art.RecordView(context.Background(), spec.ViewRecord{TargetID: idB, ViewerHash: "h2"})
	require.NoError(t, err)
	_, err = repos.Art.RecordView(context.Background(), spec.ViewRecord{TargetID: idA, ViewerHash: "h3"})
	require.NoError(t, err)

	// when
	arts, _, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Sort: "views", Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	require.Len(t, arts, 2)
	assert.Equal(t, idB, arts[0].ID)
	assert.Equal(t, idA, arts[1].ID)
}

func TestArtDAO_ListAll_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 5 {
		createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	}

	// when
	page1, total, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Limit: 2, Offset: 0,
	})
	page2, _, err2 := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: user.ID, Corner: "general", Limit: 2, Offset: 2,
	})

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
}

func TestArtDAO_ListAll_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	createArt(t, repos, userA.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, userB.ID, "general", "drawing", "B", nil, false)

	// when
	arts, total, err := repos.Art.ListAll(context.Background(), spec.ArtFilter{
		ViewerID: userA.ID, Corner: "general", Limit: 10, Offset: 0, ExcludeUserIDs: []uuid.UUID{userB.ID},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, arts, 1)
	assert.Equal(t, userA.ID, arts[0].UserID)
}

func TestArtDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	createArt(t, repos, userA.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, userA.ID, "general", "drawing", "B", nil, false)
	createArt(t, repos, userB.ID, "general", "drawing", "C", nil, false)

	// when
	arts, total, err := repos.Art.ListByUser(context.Background(), spec.ArtUserFilter{
		UserID: userA.ID, ViewerID: userA.ID, Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, arts, 2)
}

func TestArtDAO_ListByUser_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)
	}

	// when
	arts, total, err := repos.Art.ListByUser(context.Background(), spec.ArtUserFilter{
		UserID: user.ID, ViewerID: user.ID, Limit: 2, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, arts, 2)
}

func TestArtDAO_CreateComment_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)

	// when
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hello")

	// then
	comments, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Equal(t, "hello", comments[0].Body)
	assert.Nil(t, comments[0].ParentID)
}

func TestArtDAO_CreateComment_Threaded(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	parent := createArtComment(t, repos, artID, user.ID, nil, "parent")

	// when
	child := createArtComment(t, repos, artID, user.ID, &parent, "child")

	// then
	comments, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
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

func TestArtDAO_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	createArtComment(t, repos, artID, owner.ID, nil, "ok")
	createArtComment(t, repos, artID, other.ID, nil, "hide")

	// when
	comments, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: owner.ID, Limit: 10, Offset: 0, ExcludeUserIDs: []uuid.UUID{other.ID},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, comments, 1)
	assert.Equal(t, "ok", comments[0].Body)
}

func TestArtDAO_UpdateComment_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "old")

	// when
	err := repos.Art.UpdateComment(context.Background(), spec.CommentUpdate{
		CommentID: commentID, UserID: user.ID, Body: "new body",
	})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new body", comments[0].Body)
}

func TestArtDAO_UpdateComment_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "old")

	// when
	err := repos.Art.UpdateComment(context.Background(), spec.CommentUpdate{
		CommentID: commentID, UserID: other.ID, Body: "hack",
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "old")

	// when
	err := repos.Art.UpdateComment(context.Background(), spec.CommentUpdate{
		CommentID: commentID, Body: "admin body", AsAdmin: true,
	})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: owner.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin body", comments[0].Body)
}

func TestArtDAO_UpdateCommentAsAdmin_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	err := repos.Art.UpdateComment(context.Background(), spec.CommentUpdate{
		CommentID: uuid.New(), Body: "body", AsAdmin: true,
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_DeleteComment_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	err := repos.Art.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: user.ID})

	// then
	require.NoError(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestArtDAO_DeleteComment_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "hi")

	// when
	err := repos.Art.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: other.ID})

	// then
	require.Error(t, err)
}

func TestArtDAO_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	err := repos.Art.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, AsAdmin: true})

	// then
	require.NoError(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestArtDAO_UpdateCommentWithDetails_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "old")

	// when
	err := repos.Art.UpdateCommentWithDetails(context.Background(), spec.CommentUpdate{
		CommentID: commentID, UserID: admin.ID, Body: "admin body", AsAdmin: true,
	})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: owner.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin body", comments[0].Body)
}

func TestArtDAO_DeleteCommentWithAudit_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	_, err := repos.Art.DeleteCommentWithAudit(context.Background(), spec.ArtCommentDeletion{
		CommentDeletion: spec.CommentDeletion{
			CommentID: commentID,
			UserID:    user.ID,
		},
		Audit: audit.NewEntry{
			ActorID:    user.ID,
			Action:     audit.ActionArtCommentDelete,
			TargetType: audit.TargetArtComment,
			TargetID:   commentID.String(),
		},
	})

	// then
	require.NoError(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	entries, auditTotal, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{
		Action: audit.ActionArtCommentDelete,
		Page:   bounds.NewPage(10, 0),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, auditTotal)
	require.Len(t, entries, 1)
	assert.Equal(t, commentID.String(), entries[0].TargetID)
}

func TestArtDAO_DeleteCommentWithAudit_ReturnsThreadMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")
	addArtCommentMedia(t, repos, commentID, "https://example.com/root.png", "https://example.com/root-thumb.png")
	replyID := createArtComment(t, repos, artID, user.ID, &commentID, "reply")
	addArtCommentMedia(t, repos, replyID, "https://example.com/reply.png", "")
	otherID := createArtComment(t, repos, artID, user.ID, nil, "unrelated")
	addArtCommentMedia(t, repos, otherID, "https://example.com/other.png", "")

	// when
	paths, err := repos.Art.DeleteCommentWithAudit(context.Background(), spec.ArtCommentDeletion{
		CommentDeletion: spec.CommentDeletion{
			CommentID: commentID,
			UserID:    user.ID,
		},
		Audit: audit.NewEntry{
			ActorID:    user.ID,
			Action:     audit.ActionArtCommentDelete,
			TargetType: audit.TargetArtComment,
			TargetID:   commentID.String(),
		},
	})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"https://example.com/root.png",
		"https://example.com/root-thumb.png",
		"https://example.com/reply.png",
	}, paths)
}

func TestArtDAO_DeleteCommentWithAudit_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "hi")

	// when
	_, err := repos.Art.DeleteCommentWithAudit(context.Background(), spec.ArtCommentDeletion{
		CommentDeletion: spec.CommentDeletion{
			CommentID: commentID,
			UserID:    admin.ID,
			AsAdmin:   true,
		},
		Audit: audit.NewEntry{
			ActorID:    admin.ID,
			Action:     audit.ActionArtCommentDeleteAdmin,
			TargetType: audit.TargetArtComment,
			TargetID:   commentID.String(),
		},
	})

	// then
	require.NoError(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: owner.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	entries, auditTotal, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{
		Action: audit.ActionArtCommentDeleteAdmin,
		Page:   bounds.NewPage(10, 0),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, auditTotal)
	require.Len(t, entries, 1)
	assert.Equal(t, admin.ID, entries[0].ActorID)
}

func TestArtDAO_DeleteCommentWithAudit_NotOwner_WritesNoAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, owner.ID, nil, "hi")

	// when
	_, err := repos.Art.DeleteCommentWithAudit(context.Background(), spec.ArtCommentDeletion{
		CommentDeletion: spec.CommentDeletion{
			CommentID: commentID,
			UserID:    other.ID,
		},
		Audit: audit.NewEntry{
			ActorID:    other.ID,
			Action:     audit.ActionArtCommentDelete,
			TargetType: audit.TargetArtComment,
			TargetID:   commentID.String(),
		},
	})

	// then
	require.Error(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: owner.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	_, auditTotal, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{
		Action: audit.ActionArtCommentDelete,
		Page:   bounds.NewPage(10, 0),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, auditTotal)
}

func TestArtDAO_DeleteCommentWithAudit_BadActor_RollsBackDelete(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	_, err := repos.Art.DeleteCommentWithAudit(context.Background(), spec.ArtCommentDeletion{
		CommentDeletion: spec.CommentDeletion{
			CommentID: commentID,
			UserID:    user.ID,
		},
		Audit: audit.NewEntry{
			ActorID:    uuid.New(),
			Action:     audit.ActionArtCommentDelete,
			TargetType: audit.TargetArtComment,
			TargetID:   commentID.String(),
		},
	})

	// then
	require.Error(t, err)
	_, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestArtDAO_GetCommentEntityID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	got, err := repos.Art.GetCommentEntityID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, artID, got)
}

func TestArtDAO_GetCommentEntityID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Art.GetCommentEntityID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtDAO_GetCommentAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	got, err := repos.Art.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestArtDAO_GetCommentAuthorID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Art.GetCommentAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestArtDAO_LikeAndUnlikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	require.NoError(t, repos.Art.LikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID}))
	require.NoError(t, repos.Art.LikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID}))

	// then
	comments, _, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: liker.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)

	require.NoError(t, repos.Art.UnlikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID}))
	comments, _, err = repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: liker.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].LikeCount)
	assert.False(t, comments[0].UserLiked)
}

func TestArtDAO_AddCommentMedia_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")

	// when
	mediaID, err := repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{
		TargetID:     commentID,
		MediaURL:     "https://example.com/m.png",
		MediaType:    "image",
		ThumbnailURL: "https://example.com/m-thumb.png",
	})

	// then
	require.NoError(t, err)
	require.Greater(t, mediaID, int64(0))
	media, err := repos.Art.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "https://example.com/m.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
}

func TestArtDAO_GetCommentMedia_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	media, err := repos.Art.GetCommentMedia(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Empty(t, media)
}

func TestArtDAO_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	c1 := createArtComment(t, repos, artID, user.ID, nil, "one")
	c2 := createArtComment(t, repos, artID, user.ID, nil, "two")
	_, err := repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: c1, MediaURL: "u1", MediaType: "image", ThumbnailURL: "t1"})
	require.NoError(t, err)
	_, err = repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: c1, MediaURL: "u2", MediaType: "image", ThumbnailURL: "t2", SortOrder: 1})
	require.NoError(t, err)
	_, err = repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: c2, MediaURL: "u3", MediaType: "video", ThumbnailURL: "t3"})
	require.NoError(t, err)

	// when
	result, err := repos.Art.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, result[c1], 2)
	assert.Len(t, result[c2], 1)
}

func TestArtDAO_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	result, err := repos.Art.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArtDAO_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")
	mediaID, err := repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "old", MediaType: "image", ThumbnailURL: "t"})
	require.NoError(t, err)

	// when
	err = repos.Art.UpdateCommentMediaURL(context.Background(), spec.MediaURLUpdate{ID: mediaID, URL: "new"})

	// then
	require.NoError(t, err)
	media, err := repos.Art.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "new", media[0].MediaURL)
}

func TestArtDAO_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")
	mediaID, err := repos.Art.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "u", MediaType: "image", ThumbnailURL: "old"})
	require.NoError(t, err)

	// when
	err = repos.Art.UpdateCommentMediaThumbnail(context.Background(), spec.MediaURLUpdate{ID: mediaID, URL: "new"})

	// then
	require.NoError(t, err)
	media, err := repos.Art.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "new", media[0].ThumbnailURL)
}

func TestArtDAO_CreateGallery_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("GalleryOwner"))

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

func TestArtDAO_GetGalleryByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	row, err := repos.Art.GetGalleryByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestArtDAO_UpdateGallery_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createGallery(t, repos, user.ID, "Old")

	// when
	err := repos.Art.UpdateGallery(context.Background(), spec.GalleryUpdate{
		ID: id, UserID: user.ID, Name: "New Name", Description: "New Desc",
	})

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "New Name", row.Name)
	assert.Equal(t, "New Desc", row.Description)
}

func TestArtDAO_UpdateGallery_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createGallery(t, repos, owner.ID, "Old")

	// when
	err := repos.Art.UpdateGallery(context.Background(), spec.GalleryUpdate{
		ID: id, UserID: other.ID, Name: "Hack", Description: "",
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_SetGalleryCover(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))

	// when
	err := repos.Art.SetGalleryCover(context.Background(), spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: user.ID, CoverArtID: &artID})

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.NotNil(t, row.CoverArtID)
	assert.Equal(t, artID, *row.CoverArtID)
	assert.Equal(t, "https://example.com/img.png", row.CoverImageURL)
}

func TestArtDAO_SetGalleryCover_Clear(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))
	require.NoError(t, repos.Art.SetGalleryCover(context.Background(), spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: user.ID, CoverArtID: &artID}))

	// when
	err := repos.Art.SetGalleryCover(context.Background(), spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: user.ID, CoverArtID: nil})

	// then
	require.NoError(t, err)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Nil(t, row.CoverArtID)
}

func TestArtDAO_SetGalleryCover_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, owner.ID, "G")

	// when
	err := repos.Art.SetGalleryCover(context.Background(), spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: other.ID, CoverArtID: nil})

	// then
	require.Error(t, err)
}

func TestArtDAO_SetGallery_AndClear(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)

	// when
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))

	// then
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: artID, ViewerID: user.ID})
	require.NoError(t, err)
	require.NotNil(t, row.GalleryID)
	assert.Equal(t, galleryID, *row.GalleryID)

	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: nil}))
	row, err = repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: artID, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, row.GalleryID)
}

func TestArtDAO_SetGallery_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	err := repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{
		ArtID:     artID,
		UserID:    other.ID,
		GalleryID: new(createGallery(t, repos, owner.ID, "G")),
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_DeleteGallery_Owner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))

	// when
	paths, err := repos.Art.DeleteGallery(context.Background(), spec.GalleryRef{GalleryID: galleryID, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/img.png", "https://example.com/thumb.png"}, paths)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	assert.Nil(t, row)
	art, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: artID, ViewerID: user.ID})
	require.NoError(t, err)
	assert.Nil(t, art)
}

func TestArtDAO_DeleteGallery_ReturnsArtAndCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))
	commentID := createArtComment(t, repos, artID, user.ID, nil, "hi")
	addArtCommentMedia(t, repos, commentID, "https://example.com/c1.png", "https://example.com/c1-thumb.png")

	outsideID := createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	outsideComment := createArtComment(t, repos, outsideID, user.ID, nil, "hi")
	addArtCommentMedia(t, repos, outsideComment, "https://example.com/outside.png", "")

	// when
	paths, err := repos.Art.DeleteGallery(context.Background(), spec.GalleryRef{GalleryID: galleryID, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"https://example.com/img.png",
		"https://example.com/thumb.png",
		"https://example.com/c1.png",
		"https://example.com/c1-thumb.png",
	}, paths)
}

func TestArtDAO_DeleteGallery_NotOwner_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, owner.ID, "G")

	// when
	_, err := repos.Art.DeleteGallery(context.Background(), spec.GalleryRef{GalleryID: galleryID, UserID: other.ID})

	// then
	require.Error(t, err)
}

func TestArtDAO_ListGalleriesByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	createGallery(t, repos, userA.ID, "A1")
	createGallery(t, repos, userA.ID, "A2")
	createGallery(t, repos, userB.ID, "B1")

	// when
	galleries, err := repos.Art.ListGalleriesByUser(context.Background(), userA.ID)

	// then
	require.NoError(t, err)
	assert.Len(t, galleries, 2)
}

func TestArtDAO_ListAllGalleries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createGallery(t, repos, user.ID, "G1")
	createGallery(t, repos, user.ID, "G2")

	// when
	galleries, err := repos.Art.ListAllGalleries(context.Background(), "")

	// then
	require.NoError(t, err)
	assert.Len(t, galleries, 2)
}

func TestArtDAO_ListAllGalleries_FilterByCorner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	g1 := createGallery(t, repos, user.ID, "G1")
	a1 := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	a2 := createArt(t, repos, user.ID, "alt", "drawing", "B", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: a1, UserID: user.ID, GalleryID: &g1}))
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{
		ArtID:     a2,
		UserID:    user.ID,
		GalleryID: new(createGallery(t, repos, user.ID, "G2")),
	}))

	// when
	galleries, err := repos.Art.ListAllGalleries(context.Background(), "general")

	// then
	require.NoError(t, err)
	require.Len(t, galleries, 1)
	assert.Equal(t, g1, galleries[0].ID)
}

func TestArtDAO_GetGalleryPreviewImages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	for range 3 {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))
	}

	// when
	imgs, err := repos.Art.GetGalleryPreviewImages(context.Background(), spec.GalleryPreviewFilter{GalleryID: galleryID, Limit: 2})

	// then
	require.NoError(t, err)
	assert.Len(t, imgs, 2)
	assert.Equal(t, "https://example.com/img.png", imgs[0].ImageURL)
	assert.Equal(t, "https://example.com/thumb.png", imgs[0].ThumbnailURL)
}

func TestArtDAO_GetGalleryPreviewImages_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	imgs, err := repos.Art.GetGalleryPreviewImages(context.Background(), spec.GalleryPreviewFilter{GalleryID: uuid.New(), Limit: 5})

	// then
	require.NoError(t, err)
	assert.Empty(t, imgs)
}

func TestArtDAO_ListArtInGallery(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	for range 3 {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))
	}
	otherArt := createArt(t, repos, user.ID, "general", "drawing", "X", nil, false)
	require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{
		ArtID:     otherArt,
		UserID:    user.ID,
		GalleryID: new(createGallery(t, repos, user.ID, "H")),
	}))

	// when
	arts, total, err := repos.Art.ListArtInGallery(context.Background(), spec.GalleryArtFilter{
		GalleryID: galleryID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, arts, 3)
}

func TestArtDAO_ListArtInGallery_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	for range 4 {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))
	}

	// when
	arts, total, err := repos.Art.ListArtInGallery(context.Background(), spec.GalleryArtFilter{
		GalleryID: galleryID, ViewerID: user.ID, Limit: 2, Offset: 1,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, arts, 2)
}

func TestArtDAO_GetComments_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	for range 5 {
		createArtComment(t, repos, artID, user.ID, nil, "c")
	}

	// when
	comments, total, err := repos.Art.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID: artID, ViewerID: user.ID, Limit: 2, Offset: 2,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, comments, 2)
}

func TestArtDAO_CreateComment_UnknownArt_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	_, err := repos.Comments.ByID[string(mention.KindArtComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{
		TargetID: uuid.New(),
		ParentID: nil,
		UserID:   user.ID,
		Body:     "body",
	})

	// then
	require.Error(t, err)
}

func TestArtDAO_DeleteComment_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.Art.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: uuid.New(), UserID: user.ID})

	// then
	require.Error(t, err)
}

func TestArtDAO_SetGallery_ForeignGallery_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	attacker := daotest.CreateUser(t, repos)
	victim := daotest.CreateUser(t, repos)
	victimGalleryID := createGallery(t, repos, victim.ID, "Victim Gallery")
	artID := createArt(t, repos, attacker.ID, "general", "drawing", "Intruder", nil, false)

	// when
	err := repos.Art.SetGallery(context.Background(), spec.ArtGalleryAssignment{ArtID: artID, UserID: attacker.ID, GalleryID: &victimGalleryID})

	// then
	require.ErrorIs(t, err, dao.ErrArtNotOwned)
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: artID, ViewerID: attacker.ID})
	require.NoError(t, err)
	assert.Nil(t, row.GalleryID)
	listed, total, err := repos.Art.ListArtInGallery(context.Background(), spec.GalleryArtFilter{
		GalleryID: victimGalleryID, ViewerID: attacker.ID, Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, listed)
}

func TestArtDAO_SetGalleryCover_ForeignArt_Fails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, owner.ID, "G")
	foreignArtID := createArt(t, repos, other.ID, "general", "drawing", "Not Mine", nil, false)

	// when
	err := repos.Art.SetGalleryCover(context.Background(), spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: owner.ID, CoverArtID: &foreignArtID})

	// then
	require.ErrorIs(t, err, dao.ErrArtNotOwned)
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Nil(t, row.CoverArtID)
}
