package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	artImageURL     = "https://example.com/img.png"
	artThumbnailURL = "https://example.com/thumb.png"
)

func createArt(t *testing.T, repos *repository.Repositories, userID uuid.UUID, corner, artType, title string, tags []string, spoiler bool) uuid.UUID {
	t.Helper()
	created, err := repos.Art.CreateWithTags(context.Background(), spec.NewArtWithTags{
		UserID:       userID,
		Corner:       corner,
		ArtType:      artType,
		Title:        title,
		Description:  "desc",
		ImageURL:     artImageURL,
		ThumbnailURL: artThumbnailURL,
		IsSpoiler:    spoiler,
		Tags:         tags,
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

func artBackdate(t *testing.T, repos *repository.Repositories, artID uuid.UUID, createdAt string) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE art SET created_at = $1 WHERE id = $2`, createdAt, artID)
	require.NoError(t, err)
}

func artGetByID(t *testing.T, repos *repository.Repositories, artID, viewerID uuid.UUID) *model.ArtRow {
	t.Helper()
	row, err := repos.Art.GetByID(context.Background(), spec.ArtLookup{ID: artID, ViewerID: viewerID})
	require.NoError(t, err)

	return row
}

func artGallery(t *testing.T, repos *repository.Repositories, galleryID uuid.UUID) *model.GalleryRow {
	t.Helper()
	row, err := repos.Art.GetGalleryByID(context.Background(), galleryID)
	require.NoError(t, err)

	return row
}

func artRowIDs(rows []model.ArtRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func artGalleryIDs(rows []model.GalleryRow) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func TestArtDAO_CreateWithTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		spoiler  bool
		wantTags []string
	}{
		{name: "tags are trimmed and lowercased and blank tags are dropped", tags: []string{"FooBar", " baz ", "", "  "}, spoiler: false, wantTags: []string{"foobar", "baz"}},
		{name: "the spoiler flag is stored", tags: nil, spoiler: true, wantTags: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Artist"))

			// when
			id := createArt(t, repos, user.ID, "general", "drawing", "My Art", tc.tags, tc.spoiler)

			// then
			row := artGetByID(t, repos, id, user.ID)
			require.NotNil(t, row)
			assert.Equal(t, id, row.ID)
			assert.Equal(t, user.ID, row.UserID)
			assert.Equal(t, "general", row.Corner)
			assert.Equal(t, "drawing", row.ArtType)
			assert.Equal(t, "My Art", row.Title)
			assert.Equal(t, "Artist", row.AuthorDisplayName)
			assert.Equal(t, 0, row.LikeCount)
			assert.Equal(t, 0, row.CommentCount)
			assert.Equal(t, tc.spoiler, row.IsSpoiler)

			tags, err := repos.Art.GetTags(context.Background(), id)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantTags, tags)
		})
	}
}

func TestArtDAO_UpdateWithTags(t *testing.T) {
	tests := []struct {
		name            string
		byOwner         bool
		asAdmin         bool
		wantErr         bool
		wantTitle       string
		wantDescription string
		wantSpoiler     bool
		wantTags        []string
	}{
		{name: "the owner replaces the fields and tags", byOwner: true, asAdmin: false, wantErr: false, wantTitle: "New Title", wantDescription: "New Desc", wantSpoiler: true, wantTags: []string{"b", "c"}},
		{name: "an admin edits someone else's art", byOwner: false, asAdmin: true, wantErr: false, wantTitle: "New Title", wantDescription: "New Desc", wantSpoiler: true, wantTags: []string{"b", "c"}},
		{name: "another user cannot edit it and nothing changes", byOwner: false, asAdmin: false, wantErr: true, wantTitle: "Old", wantDescription: "desc", wantSpoiler: false, wantTags: []string{"a"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			id := createArt(t, repos, owner.ID, "general", "drawing", "Old", []string{"a"}, false)

			editor := other.ID
			if tc.byOwner {
				editor = owner.ID
			}

			// when
			err := repos.Art.UpdateWithTags(context.Background(), spec.ArtUpdateWithTags{
				ID: id, UserID: editor, Title: "New Title", Description: "New Desc", IsSpoiler: true, AsAdmin: tc.asAdmin,
				Tags: []string{"b", "c"},
			})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			row := artGetByID(t, repos, id, owner.ID)
			require.NotNil(t, row)
			assert.Equal(t, tc.wantTitle, row.Title)
			assert.Equal(t, tc.wantDescription, row.Description)
			assert.Equal(t, tc.wantSpoiler, row.IsSpoiler)

			tags, err := repos.Art.GetTags(context.Background(), id)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantTags, tags)
		})
	}
}

func TestArtDAO_DeleteWithImage(t *testing.T) {
	tests := []struct {
		name       string
		byOwner    bool
		asAdmin    bool
		missingArt bool
		wantErr    bool
	}{
		{name: "the owner deletes the art and gets back its image, thumbnail and every comment media path", byOwner: true, asAdmin: false, missingArt: false, wantErr: false},
		{name: "an admin deletes someone else's art and gets back the same paths", byOwner: false, asAdmin: true, missingArt: false, wantErr: false},
		{name: "another user cannot delete it and gets no paths", byOwner: false, asAdmin: false, missingArt: false, wantErr: true},
		{name: "an unknown art id fails without paths", byOwner: true, asAdmin: false, missingArt: true, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			artID := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)
			commentTarget := spec.CommentTarget{Kind: spec.CommentKindArt, ID: artID}
			commentID := createComment(t, repos, commentTarget, owner.ID, nil, "hi")
			addCommentMedia(t, repos, spec.CommentKindArt, commentID, "https://example.com/c1.png", "https://example.com/c1-thumb.png")
			replyID := createComment(t, repos, commentTarget, owner.ID, &commentID, "reply")
			addCommentMedia(t, repos, spec.CommentKindArt, replyID, "https://example.com/c2.png", "")

			actor := other.ID
			action := audit.ActionArtDeleteAdmin
			if tc.byOwner {
				actor = owner.ID
				action = audit.ActionArtDelete
			}

			target := artID
			if tc.missingArt {
				target = uuid.New()
			}

			// when
			paths, err := repos.Art.DeleteWithImage(context.Background(), spec.ArtDelete{
				ID:      target,
				UserID:  actor,
				AsAdmin: tc.asAdmin,
				Audit: audit.NewEntry{
					ActorID:    actor,
					Action:     action,
					TargetType: audit.TargetArt,
					TargetID:   target.String(),
					SubjectID:  owner.ID,
				},
			})

			// then
			row := artGetByID(t, repos, artID, owner.ID)
			if tc.wantErr {
				require.Error(t, err)
				assert.Empty(t, paths)
				assert.NotNil(t, row)
			} else {
				require.NoError(t, err)
				require.Len(t, paths, 5)
				assert.Equal(t, []string{artImageURL, artThumbnailURL}, paths[:2])
				assert.ElementsMatch(t, []string{
					"https://example.com/c1.png",
					"https://example.com/c1-thumb.png",
					"https://example.com/c2.png",
				}, paths[2:])
				assert.Nil(t, row)
			}
		})
	}
}

func TestArtDAO_Lookups(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	artist := daotest.CreateUser(t, repos)
	artID := createArt(t, repos, artist.ID, "general", "drawing", "A", nil, false)

	// when
	authorID, authorErr := repos.Art.GetArtAuthorID(ctx, artID)
	imageURL, imageErr := repos.Art.GetImageURL(ctx, artID)

	// then
	require.NoError(t, authorErr)
	assert.Equal(t, artist.ID, authorID)

	require.NoError(t, imageErr)
	assert.Equal(t, artImageURL, imageURL)
}

func TestArtDAO_UnknownIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	unknownID := uuid.New()

	// when
	art, artErr := repos.Art.GetByID(ctx, spec.ArtLookup{ID: unknownID, ViewerID: user.ID})
	gallery, galleryErr := repos.Art.GetGalleryByID(ctx, unknownID)
	_, authorErr := repos.Art.GetArtAuthorID(ctx, unknownID)
	_, imageErr := repos.Art.GetImageURL(ctx, unknownID)
	_, commentErr := repos.Comment.CreateComment(ctx, spec.NewComment{
		Target: spec.CommentTarget{Kind: spec.CommentKindArt, ID: unknownID},
		UserID: user.ID,
		Body:   "body",
	})

	// then
	require.NoError(t, artErr)
	assert.Nil(t, art)

	require.NoError(t, galleryErr)
	assert.Nil(t, gallery)

	assert.ErrorIs(t, authorErr, dao.ErrNotFound)
	assert.Error(t, imageErr)
	assert.Error(t, commentErr, "a comment on art that does not exist is refused by the database")
}

func TestArtDAO_LikeAndUnlike(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)

	// when
	require.NoError(t, repos.Art.Like(ctx, spec.Like{UserID: liker.ID, TargetID: id}))
	require.NoError(t, repos.Art.Like(ctx, spec.Like{UserID: liker.ID, TargetID: id}))

	// then
	row := artGetByID(t, repos, id, liker.ID)
	require.NotNil(t, row)
	assert.Equal(t, 1, row.LikeCount)
	assert.True(t, row.UserLiked)

	require.NoError(t, repos.Art.Unlike(ctx, spec.Like{UserID: liker.ID, TargetID: id}))
	row = artGetByID(t, repos, id, liker.ID)
	require.NotNil(t, row)
	assert.Equal(t, 0, row.LikeCount)
	assert.False(t, row.UserLiked)
}

func TestArtDAO_GetLikedBy(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	id := createArt(t, repos, owner.ID, "general", "drawing", "T", nil, false)
	require.NoError(t, repos.Art.Like(ctx, spec.Like{UserID: a.ID, TargetID: id}))
	require.NoError(t, repos.Art.Like(ctx, spec.Like{UserID: b.ID, TargetID: id}))

	// when
	everyone, everyoneErr := repos.Art.GetLikedBy(ctx, spec.LikedByQuery{TargetID: id, ExcludeUserIDs: nil})
	filtered, filteredErr := repos.Art.GetLikedBy(ctx, spec.LikedByQuery{TargetID: id, ExcludeUserIDs: []uuid.UUID{a.ID}})

	// then
	require.NoError(t, everyoneErr)
	require.Len(t, everyone, 2)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, []uuid.UUID{everyone[0].ID, everyone[1].ID})

	require.NoError(t, filteredErr)
	require.Len(t, filtered, 1)
	assert.Equal(t, b.ID, filtered[0].ID)
}

func TestArtDAO_RecordView(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	id := createArt(t, repos, user.ID, "general", "drawing", "T", nil, false)

	// when
	firstNew, err := repos.Art.RecordView(ctx, spec.ViewRecord{TargetID: id, ViewerHash: "hash1"})
	require.NoError(t, err)
	dupNew, err := repos.Art.RecordView(ctx, spec.ViewRecord{TargetID: id, ViewerHash: "hash1"})
	require.NoError(t, err)
	secondNew, err := repos.Art.RecordView(ctx, spec.ViewRecord{TargetID: id, ViewerHash: "hash2"})
	require.NoError(t, err)

	// then
	assert.True(t, firstNew)
	assert.False(t, dupNew)
	assert.True(t, secondNew)
	row := artGetByID(t, repos, id, user.ID)
	require.NotNil(t, row)
	assert.Equal(t, 2, row.ViewCount)
}

func TestArtDAO_GetTagsBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	id1 := createArt(t, repos, user.ID, "general", "drawing", "A", []string{"x", "y"}, false)
	id2 := createArt(t, repos, user.ID, "general", "drawing", "B", []string{"z"}, false)
	id3 := createArt(t, repos, user.ID, "general", "drawing", "C", nil, false)

	// when
	result, err := repos.Art.GetTagsBatch(ctx, []uuid.UUID{id1, id2, id3})
	empty, emptyErr := repos.Art.GetTagsBatch(ctx, nil)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"x", "y"}, result[id1])
	assert.ElementsMatch(t, []string{"z"}, result[id2])
	assert.Empty(t, result[id3])

	require.NoError(t, emptyErr)
	assert.Nil(t, empty)
}

func TestArtDAO_GetPopularTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", []string{"common", "rare"}, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", []string{"common"}, false)
	createArt(t, repos, user.ID, "other", "drawing", "C", []string{"common"}, false)

	tests := []struct {
		name   string
		corner string
		want   []model.TagCount
	}{
		{name: "a corner counts only its own art, most used first", corner: "general", want: []model.TagCount{{Tag: "common", Count: 2}, {Tag: "rare", Count: 1}}},
		{name: "an empty corner counts across every corner", corner: "", want: []model.TagCount{{Tag: "common", Count: 3}, {Tag: "rare", Count: 1}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			tags, err := repos.Art.GetPopularTags(context.Background(), spec.PopularTagFilter{Corner: tc.corner, Limit: 10})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, tags)
		})
	}
}

func TestArtDAO_Counts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	createArt(t, repos, user.ID, "general", "drawing", "B", nil, false)
	stale := createArt(t, repos, user.ID, "general", "drawing", "Old", nil, false)
	artBackdate(t, repos, stale, "2020-01-01T00:00:00Z")
	createArt(t, repos, other.ID, "alt", "drawing", "C", nil, false)

	// when
	counts, countsErr := repos.Art.GetCornerCounts(ctx)
	today, todayErr := repos.Art.CountUserArtToday(ctx, user.ID)

	// then
	require.NoError(t, countsErr)
	assert.Equal(t, map[string]int{"general": 3, "alt": 1}, counts)

	require.NoError(t, todayErr)
	assert.Equal(t, 2, today)
}

func TestArtDAO_ListAll(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	viewer := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	apple := createArt(t, repos, viewer.ID, "general", "drawing", "UniqueApple", []string{"red"}, false)
	banana := createArt(t, repos, viewer.ID, "general", "photo", "Banana", []string{"blue"}, false)
	cherry := createArt(t, repos, stranger.ID, "general", "drawing", "Cherry", nil, false)
	createArt(t, repos, viewer.ID, "alt", "drawing", "Date", nil, false)
	artBackdate(t, repos, apple, "2026-01-01T00:00:00Z")
	artBackdate(t, repos, banana, "2026-01-02T00:00:00Z")
	artBackdate(t, repos, cherry, "2026-01-03T00:00:00Z")

	require.NoError(t, repos.Art.Like(ctx, spec.Like{UserID: liker.ID, TargetID: apple}))
	for _, view := range []spec.ViewRecord{{TargetID: banana, ViewerHash: "h1"}, {TargetID: banana, ViewerHash: "h2"}, {TargetID: apple, ViewerHash: "h3"}} {
		_, err := repos.Art.RecordView(ctx, view)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		filter    spec.ArtFilter
		want      []uuid.UUID
		wantTotal int
	}{
		{name: "a corner lists only its own art, newest first", filter: spec.ArtFilter{Corner: "general", Limit: 10}, want: []uuid.UUID{cherry, banana, apple}, wantTotal: 3},
		{name: "art type narrows the list", filter: spec.ArtFilter{Corner: "general", ArtType: "photo", Limit: 10}, want: []uuid.UUID{banana}, wantTotal: 1},
		{name: "search matches part of a title", filter: spec.ArtFilter{Corner: "general", Search: "Apple", Limit: 10}, want: []uuid.UUID{apple}, wantTotal: 1},
		{name: "a tag narrows the list", filter: spec.ArtFilter{Corner: "general", Tag: "red", Limit: 10}, want: []uuid.UUID{apple}, wantTotal: 1},
		{name: "excluded users are left out of the page and the total", filter: spec.ArtFilter{Corner: "general", ExcludeUserIDs: []uuid.UUID{stranger.ID}, Limit: 10}, want: []uuid.UUID{banana, apple}, wantTotal: 2},
		{name: "popular sorts by likes then newest", filter: spec.ArtFilter{Corner: "general", Sort: "popular", Limit: 10}, want: []uuid.UUID{apple, cherry, banana}, wantTotal: 3},
		{name: "views sorts by view count", filter: spec.ArtFilter{Corner: "general", Sort: "views", Limit: 10}, want: []uuid.UUID{banana, apple, cherry}, wantTotal: 3},
		{name: "the first page is capped at the limit while the total counts everything", filter: spec.ArtFilter{Corner: "general", Limit: 2}, want: []uuid.UUID{cherry, banana}, wantTotal: 3},
		{name: "a later page starts at the offset", filter: spec.ArtFilter{Corner: "general", Limit: 2, Offset: 2}, want: []uuid.UUID{apple}, wantTotal: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			filter := tc.filter
			filter.ViewerID = viewer.ID

			// when
			arts, total, err := repos.Art.ListAll(ctx, filter)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)
			assert.Equal(t, tc.want, artRowIDs(arts))
		})
	}
}

func TestArtDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	oldest := createArt(t, repos, author.ID, "general", "drawing", "A", nil, false)
	middle := createArt(t, repos, author.ID, "alt", "photo", "B", nil, false)
	newest := createArt(t, repos, author.ID, "general", "drawing", "C", nil, false)
	createArt(t, repos, other.ID, "general", "drawing", "D", nil, false)
	artBackdate(t, repos, oldest, "2026-01-01T00:00:00Z")
	artBackdate(t, repos, middle, "2026-01-02T00:00:00Z")
	artBackdate(t, repos, newest, "2026-01-03T00:00:00Z")

	tests := []struct {
		name   string
		limit  int
		offset int
		want   []uuid.UUID
	}{
		{name: "only the author's art is listed, newest first, across corners", limit: 10, offset: 0, want: []uuid.UUID{newest, middle, oldest}},
		{name: "the first page is capped at the limit while the total counts everything", limit: 2, offset: 0, want: []uuid.UUID{newest, middle}},
		{name: "a later page starts at the offset", limit: 2, offset: 2, want: []uuid.UUID{oldest}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			arts, total, err := repos.Art.ListByUser(context.Background(), spec.ArtUserFilter{
				UserID: author.ID, ViewerID: author.ID, Limit: tc.limit, Offset: tc.offset,
			})

			// then
			require.NoError(t, err)
			assert.Equal(t, 3, total)
			assert.Equal(t, tc.want, artRowIDs(arts))
		})
	}
}

func TestArtDAO_CreateGallery_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("GalleryOwner"))

	// when
	id := createGallery(t, repos, user.ID, "My Gallery")

	// then
	row := artGallery(t, repos, id)
	require.NotNil(t, row)
	assert.Equal(t, "My Gallery", row.Name)
	assert.Equal(t, "desc", row.Description)
	assert.Equal(t, "GalleryOwner", row.AuthorDisplayName)
	assert.Equal(t, 0, row.ArtCount)
}

func TestArtDAO_UpdateGallery(t *testing.T) {
	tests := []struct {
		name            string
		byOwner         bool
		wantErr         bool
		wantName        string
		wantDescription string
	}{
		{name: "the owner renames and redescribes it", byOwner: true, wantErr: false, wantName: "New Name", wantDescription: "New Desc"},
		{name: "another user cannot edit it", byOwner: false, wantErr: true, wantName: "Old", wantDescription: "desc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			id := createGallery(t, repos, owner.ID, "Old")

			editor := other.ID
			if tc.byOwner {
				editor = owner.ID
			}

			// when
			err := repos.Art.UpdateGallery(context.Background(), spec.GalleryUpdate{
				ID: id, UserID: editor, Name: "New Name", Description: "New Desc",
			})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			row := artGallery(t, repos, id)
			require.NotNil(t, row)
			assert.Equal(t, tc.wantName, row.Name)
			assert.Equal(t, tc.wantDescription, row.Description)
		})
	}
}

func TestArtDAO_SetGalleryCover_AndClear(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
	require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))

	// when
	err := repos.Art.SetGalleryCover(ctx, spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: user.ID, CoverArtID: &artID})

	// then
	require.NoError(t, err)
	row := artGallery(t, repos, galleryID)
	require.NotNil(t, row)
	require.NotNil(t, row.CoverArtID)
	assert.Equal(t, artID, *row.CoverArtID)
	assert.Equal(t, artImageURL, row.CoverImageURL)

	require.NoError(t, repos.Art.SetGalleryCover(ctx, spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: user.ID, CoverArtID: nil}))
	row = artGallery(t, repos, galleryID)
	require.NotNil(t, row)
	assert.Nil(t, row.CoverArtID)
}

func TestArtDAO_SetGalleryCover_Rejected(t *testing.T) {
	tests := []struct {
		name         string
		byOwner      bool
		foreignCover bool
	}{
		{name: "another user cannot clear the cover", byOwner: false, foreignCover: false},
		{name: "art the gallery owner does not own cannot become the cover", byOwner: true, foreignCover: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			galleryID := createGallery(t, repos, owner.ID, "G")
			coverID := createArt(t, repos, owner.ID, "general", "drawing", "Mine", nil, false)
			foreignArtID := createArt(t, repos, other.ID, "general", "drawing", "Not Mine", nil, false)
			require.NoError(t, repos.Art.SetGalleryCover(ctx, spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: owner.ID, CoverArtID: &coverID}))

			editor := other.ID
			if tc.byOwner {
				editor = owner.ID
			}

			var cover *uuid.UUID
			if tc.foreignCover {
				cover = &foreignArtID
			}

			// when
			err := repos.Art.SetGalleryCover(ctx, spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: editor, CoverArtID: cover})

			// then
			require.ErrorIs(t, err, dao.ErrArtNotOwned)
			row := artGallery(t, repos, galleryID)
			require.NotNil(t, row)
			require.NotNil(t, row.CoverArtID)
			assert.Equal(t, coverID, *row.CoverArtID)
		})
	}
}

func TestArtDAO_SetGallery_AndClear(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")
	artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)

	// when
	require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))

	// then
	row := artGetByID(t, repos, artID, user.ID)
	require.NotNil(t, row)
	require.NotNil(t, row.GalleryID)
	assert.Equal(t, galleryID, *row.GalleryID)

	require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: nil}))
	row = artGetByID(t, repos, artID, user.ID)
	require.NotNil(t, row)
	assert.Nil(t, row.GalleryID)
}

func TestArtDAO_SetGallery_Rejected(t *testing.T) {
	tests := []struct {
		name           string
		byOwner        bool
		foreignGallery bool
	}{
		{name: "another user cannot file the owner's art", byOwner: false, foreignGallery: false},
		{name: "the owner cannot file their art into someone else's gallery", byOwner: true, foreignGallery: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			artID := createArt(t, repos, owner.ID, "general", "drawing", "Intruder", nil, false)

			galleryOwner := owner.ID
			if tc.foreignGallery {
				galleryOwner = other.ID
			}
			galleryID := createGallery(t, repos, galleryOwner, "G")

			actor := other.ID
			if tc.byOwner {
				actor = owner.ID
			}

			// when
			err := repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: artID, UserID: actor, GalleryID: &galleryID})

			// then
			require.ErrorIs(t, err, dao.ErrArtNotOwned)
			row := artGetByID(t, repos, artID, owner.ID)
			require.NotNil(t, row)
			assert.Nil(t, row.GalleryID)

			listed, total, err := repos.Art.ListArtInGallery(ctx, spec.GalleryArtFilter{
				GalleryID: galleryID, ViewerID: actor, Limit: 10, Offset: 0,
			})
			require.NoError(t, err)
			assert.Equal(t, 0, total)
			assert.Empty(t, listed)
		})
	}
}

func TestArtDAO_DeleteGallery(t *testing.T) {
	tests := []struct {
		name    string
		byOwner bool
		wantErr bool
	}{
		{name: "the owner deletes the gallery with its art and gets back the art's image and comment media paths", byOwner: true, wantErr: false},
		{name: "another user cannot delete it and nothing is removed", byOwner: false, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			galleryID := createGallery(t, repos, owner.ID, "G")
			artID := createArt(t, repos, owner.ID, "general", "drawing", "A", nil, false)
			require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: artID, UserID: owner.ID, GalleryID: &galleryID}))
			commentID := createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindArt, ID: artID}, owner.ID, nil, "hi")
			addCommentMedia(t, repos, spec.CommentKindArt, commentID, "https://example.com/c1.png", "https://example.com/c1-thumb.png")

			outsideID := createArt(t, repos, owner.ID, "general", "drawing", "B", nil, false)
			outsideComment := createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindArt, ID: outsideID}, owner.ID, nil, "hi")
			addCommentMedia(t, repos, spec.CommentKindArt, outsideComment, "https://example.com/outside.png", "")

			actor := other.ID
			if tc.byOwner {
				actor = owner.ID
			}

			// when
			paths, err := repos.Art.DeleteGallery(ctx, spec.GalleryRef{GalleryID: galleryID, UserID: actor})

			// then
			gallery := artGallery(t, repos, galleryID)
			art := artGetByID(t, repos, artID, owner.ID)
			assert.NotNil(t, artGetByID(t, repos, outsideID, owner.ID))

			if tc.wantErr {
				require.Error(t, err)
				assert.Empty(t, paths)
				assert.NotNil(t, gallery)
				assert.NotNil(t, art)
			} else {
				require.NoError(t, err)
				assert.Equal(t, []string{
					artImageURL,
					artThumbnailURL,
					"https://example.com/c1.png",
					"https://example.com/c1-thumb.png",
				}, paths)
				assert.Nil(t, gallery)
				assert.Nil(t, art)
			}
		})
	}
}

func TestArtDAO_ListGalleries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	generalGallery := createGallery(t, repos, userA.ID, "A1")
	altGallery := createGallery(t, repos, userA.ID, "A2")
	b1Gallery := createGallery(t, repos, userB.ID, "B1")
	generalArt := createArt(t, repos, userA.ID, "general", "drawing", "A", nil, false)
	altArt := createArt(t, repos, userA.ID, "alt", "drawing", "B", nil, false)
	require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: generalArt, UserID: userA.ID, GalleryID: &generalGallery}))
	require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: altArt, UserID: userA.ID, GalleryID: &altGallery}))

	// when
	byUser, byUserErr := repos.Art.ListGalleriesByUser(ctx, userA.ID)
	all, allErr := repos.Art.ListAllGalleries(ctx, "")
	general, generalErr := repos.Art.ListAllGalleries(ctx, "general")

	// then
	require.NoError(t, byUserErr)
	assert.ElementsMatch(t, []uuid.UUID{generalGallery, altGallery}, artGalleryIDs(byUser))

	require.NoError(t, allErr)
	assert.ElementsMatch(t, []uuid.UUID{generalGallery, altGallery, b1Gallery}, artGalleryIDs(all))

	require.NoError(t, generalErr)
	assert.Equal(t, []uuid.UUID{generalGallery}, artGalleryIDs(general))
}

func TestArtDAO_GalleryContents(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	galleryID := createGallery(t, repos, user.ID, "G")

	var inGallery []uuid.UUID
	for range 4 {
		artID := createArt(t, repos, user.ID, "general", "drawing", "A", nil, false)
		require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{ArtID: artID, UserID: user.ID, GalleryID: &galleryID}))
		inGallery = append(inGallery, artID)
	}

	otherArt := createArt(t, repos, user.ID, "general", "drawing", "X", nil, false)
	require.NoError(t, repos.Art.SetGallery(ctx, spec.ArtGalleryAssignment{
		ArtID:     otherArt,
		UserID:    user.ID,
		GalleryID: new(createGallery(t, repos, user.ID, "H")),
	}))

	// when
	arts, total, err := repos.Art.ListArtInGallery(ctx, spec.GalleryArtFilter{
		GalleryID: galleryID, ViewerID: user.ID, Limit: 10, Offset: 0,
	})
	page, pageTotal, pageErr := repos.Art.ListArtInGallery(ctx, spec.GalleryArtFilter{
		GalleryID: galleryID, ViewerID: user.ID, Limit: 2, Offset: 1,
	})
	previews, previewsErr := repos.Art.GetGalleryPreviewImages(ctx, spec.GalleryPreviewFilter{GalleryID: galleryID, Limit: 2})

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.ElementsMatch(t, inGallery, artRowIDs(arts))

	require.NoError(t, pageErr)
	assert.Equal(t, 4, pageTotal)
	assert.Len(t, page, 2)

	require.NoError(t, previewsErr)
	require.Len(t, previews, 2)
	assert.Equal(t, artImageURL, previews[0].ImageURL)
	assert.Equal(t, artThumbnailURL, previews[0].ThumbnailURL)
}
