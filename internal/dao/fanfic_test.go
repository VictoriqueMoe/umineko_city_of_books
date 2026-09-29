package dao_test

import (
	"cmp"
	"context"
	"database/sql"
	"slices"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	fanficparams "umineko_city_of_books/internal/fanfic/params"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeFanficChars() []dto.FanficCharacter {
	return []dto.FanficCharacter{
		{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"},
		{Series: "Umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
	}
}

func fanficInsert(t *testing.T, repos *repository.Repositories, s spec.NewFanficWithDetails) uuid.UUID {
	t.Helper()

	s.Series = cmp.Or(s.Series, "Umineko")
	s.Rating = cmp.Or(s.Rating, "K")
	s.Language = cmp.Or(s.Language, "English")
	s.Status = cmp.Or(s.Status, "in_progress")

	created, err := repos.Fanfic.CreateWithDetails(context.Background(), s)
	require.NoError(t, err)

	return created.ID
}

func createFanfic(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string) uuid.UUID {
	t.Helper()

	return fanficInsert(t, repos, spec.NewFanficWithDetails{
		NewFanfic:  spec.NewFanfic{UserID: userID, Title: title, Summary: "summary"},
		Genres:     []string{"Drama", "Mystery"},
		Tags:       []string{"angst", "fluff"},
		Characters: makeFanficChars(),
	})
}

func createFanficComment(t *testing.T, repos *repository.Repositories, fanficID uuid.UUID, parentID *uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()

	created, err := repos.Comments.ByID[string(mention.KindFanficComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{
		TargetID: fanficID,
		ParentID: parentID,
		UserID:   userID,
		Body:     body,
	})
	require.NoError(t, err)

	return created.ID
}

func createFanficChapter(t *testing.T, repos *repository.Repositories, fanficID uuid.UUID, chapterNumber int) uuid.UUID {
	t.Helper()

	created, err := repos.Fanfic.CreateChapter(context.Background(), spec.NewChapter{
		FanficID:  fanficID,
		Number:    chapterNumber,
		Title:     "Chapter",
		Body:      "body text",
		WordCount: 100,
	})
	require.NoError(t, err)

	return created.ID
}

func fanficAddCommentMedia(t *testing.T, repos *repository.Repositories, commentID uuid.UUID, mediaURL, thumbnailURL string) int64 {
	t.Helper()

	id, err := repos.Fanfic.AddCommentMedia(context.Background(), spec.NewMedia{
		TargetID:     commentID,
		MediaURL:     mediaURL,
		MediaType:    "image",
		ThumbnailURL: thumbnailURL,
	})
	require.NoError(t, err)

	return id
}

func fanficRow(t *testing.T, repos *repository.Repositories, id, viewerID uuid.UUID) *model.FanficRow {
	t.Helper()

	row, err := repos.Fanfic.GetByID(context.Background(), spec.FanficLookup{ID: id, ViewerID: viewerID})
	require.NoError(t, err)
	require.NotNil(t, row)

	return row
}

func fanficChapter(t *testing.T, repos *repository.Repositories, fanficID uuid.UUID, chapterNumber int) *model.FanficChapterRow {
	t.Helper()

	chapter, err := repos.Fanfic.GetChapter(context.Background(), spec.FanficChapterLookup{FanficID: fanficID, ChapterNumber: chapterNumber})
	require.NoError(t, err)

	return chapter
}

func fanficComments(t *testing.T, repos *repository.Repositories, fanficID, viewerID uuid.UUID) []model.CommentRow {
	t.Helper()

	rows, _, err := repos.Fanfic.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: fanficID, ViewerID: viewerID, Limit: 500})
	require.NoError(t, err)

	return rows
}

func fanficCommentByID(t *testing.T, rows []model.CommentRow, id uuid.UUID) model.CommentRow {
	t.Helper()

	i := slices.IndexFunc(rows, func(row model.CommentRow) bool {
		return row.ID == id
	})
	require.NotEqual(t, -1, i)

	return rows[i]
}

func fanficCommentBodies(rows []model.CommentRow) []string {
	var bodies []string
	for _, row := range rows {
		bodies = append(bodies, row.Body)
	}

	return bodies
}

func fanficTitles(rows []model.FanficRow) []string {
	var titles []string
	for _, row := range rows {
		titles = append(titles, row.Title)
	}

	return titles
}

func TestFanficDAO_CreateWithDetails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Fanfic.CreateWithDetails(ctx, spec.NewFanficWithDetails{
		NewFanfic: spec.NewFanfic{
			UserID:    user.ID,
			Title:     "Title",
			Summary:   "Summary",
			Series:    "Umineko",
			Rating:    "T",
			Language:  "English",
			Status:    "in_progress",
			IsOneshot: true,
			IsPairing: true,
		},
		Genres: []string{"Drama"},
		Tags:   []string{"  ", "", "keep"},
		Characters: []dto.FanficCharacter{
			{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"},
			{Series: "Umineko", CharacterID: "x", CharacterName: "  Padded  "},
		},
	})

	// then
	require.NoError(t, err)

	row := fanficRow(t, repos, created.ID, user.ID)
	assert.Equal(t, "Title", row.Title)
	assert.Equal(t, "Summary", row.Summary)
	assert.True(t, row.IsOneshot)
	assert.False(t, row.ContainsLemons)
	assert.True(t, row.IsPairing)

	tags, err := repos.Fanfic.GetTags(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"keep"}, tags, "blank tags are skipped")

	chars, err := repos.Fanfic.GetCharacters(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, chars, 2)
	assert.Equal(t, "Battler", chars[0].CharacterName)
	assert.Equal(t, "Padded", chars[1].CharacterName, "character names are trimmed")
}

func TestFanficDAO_UpdateWithDetails_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Old")

	// when
	err := repos.Fanfic.UpdateWithDetails(ctx, spec.FanficUpdateWithDetails{
		FanficUpdate: spec.FanficUpdate{
			ID:             id,
			UserID:         user.ID,
			Title:          "New",
			Summary:        "newsum",
			Series:         "Higurashi",
			Rating:         "M",
			Language:       "Spanish",
			Status:         "completed",
			IsOneshot:      true,
			ContainsLemons: true,
		},
		Genres:     []string{"Angst"},
		Tags:       []string{"newtag"},
		Characters: []dto.FanficCharacter{{Series: "Higurashi", CharacterID: "rena", CharacterName: "Rena"}},
	})

	// then
	require.NoError(t, err)

	row := fanficRow(t, repos, id, user.ID)
	assert.Equal(t, "New", row.Title)
	assert.Equal(t, "Higurashi", row.Series)
	assert.Equal(t, "completed", row.Status)
	assert.True(t, row.ContainsLemons)

	genres, err := repos.Fanfic.GetGenres(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, []string{"Angst"}, genres)

	tags, err := repos.Fanfic.GetTags(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, []string{"newtag"}, tags)

	chars, err := repos.Fanfic.GetCharacters(ctx, id)
	require.NoError(t, err)
	require.Len(t, chars, 1)
	assert.Equal(t, "Rena", chars[0].CharacterName)
}

func TestFanficDAO_UpdateWithDetails_NonOwner(t *testing.T) {
	cases := []struct {
		name      string
		asAdmin   bool
		wantErr   bool
		wantTitle string
	}{
		{name: "another user is refused and the fanfic is left alone", asAdmin: false, wantErr: true, wantTitle: "Title"},
		{name: "an admin may edit someone else's fanfic", asAdmin: true, wantErr: false, wantTitle: "Edited"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			editor := daotest.CreateUser(t, repos)
			id := createFanfic(t, repos, owner.ID, "Title")

			// when
			err := repos.Fanfic.UpdateWithDetails(context.Background(), spec.FanficUpdateWithDetails{
				FanficUpdate: spec.FanficUpdate{
					ID:       id,
					UserID:   editor.ID,
					Title:    "Edited",
					Series:   "Umineko",
					Rating:   "K",
					Language: "English",
					Status:   "in_progress",
					AsAdmin:  tc.asAdmin,
				},
			})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantTitle, fanficRow(t, repos, id, owner.ID).Title)
		})
	}
}

func TestFanficDAO_GetByID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author Name"))
	id := createFanfic(t, repos, author.ID, "Title")
	require.NoError(t, repos.Fanfic.UpdateCoverImage(ctx, spec.FanficCoverUpdate{ID: id, ImageURL: "https://img/x.png", ThumbnailURL: "https://img/x_t.png"}))

	// when
	row, err := repos.Fanfic.GetByID(ctx, spec.FanficLookup{ID: id, ViewerID: author.ID})
	authorID, authorIDErr := repos.Fanfic.GetAuthorID(ctx, id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Author Name", row.AuthorDisplayName)
	assert.Equal(t, author.Username, row.AuthorUsername)
	assert.Equal(t, "https://img/x.png", row.CoverImageURL)
	assert.Equal(t, "https://img/x_t.png", row.CoverThumbnailURL)

	require.NoError(t, authorIDErr)
	assert.Equal(t, author.ID, authorID)

	// when the fanfic does not exist
	missing, missingErr := repos.Fanfic.GetByID(ctx, spec.FanficLookup{ID: uuid.New(), ViewerID: author.ID})
	_, authorErr := repos.Fanfic.GetAuthorID(ctx, uuid.New())

	// then
	require.NoError(t, missingErr)
	assert.Nil(t, missing)

	require.ErrorIs(t, authorErr, dao.ErrNotFound)
}

func TestFanficDAO_DeleteFanfic(t *testing.T) {
	commentPaths := []string{
		"/uploads/images/one.png",
		"/uploads/images/one_thumb.png",
		"/uploads/images/one_no_thumb.gif",
		"/uploads/images/two.png",
		"/uploads/images/two_thumb.png",
	}

	cases := []struct {
		name       string
		byOwner    bool
		asAdmin    bool
		coverThumb string
		wantErr    bool
		wantPaths  []string
	}{
		{name: "the owner gets back the cover, its thumbnail and every comment's media, blank thumbnails skipped", byOwner: true, asAdmin: false, coverThumb: "/uploads/images/cover_thumb.png", wantErr: false, wantPaths: append([]string{"/uploads/images/cover.png", "/uploads/images/cover_thumb.png"}, commentPaths...)},
		{name: "an admin deleting someone else's fanfic collects every comment's media and a blank cover thumbnail adds no path", byOwner: false, asAdmin: true, coverThumb: "", wantErr: false, wantPaths: append([]string{"/uploads/images/cover.png"}, commentPaths...)},
		{name: "another user is refused, gets no paths back and the fanfic survives", byOwner: false, asAdmin: false, coverThumb: "/uploads/images/cover_thumb.png", wantErr: true, wantPaths: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			commenter := daotest.CreateUser(t, repos)
			id := createFanfic(t, repos, owner.ID, "Title")
			require.NoError(t, repos.Fanfic.UpdateCoverImage(ctx, spec.FanficCoverUpdate{ID: id, ImageURL: "/uploads/images/cover.png", ThumbnailURL: tc.coverThumb}))
			ownersComment := createFanficComment(t, repos, id, nil, owner.ID, "one")
			commentersComment := createFanficComment(t, repos, id, nil, commenter.ID, "two")
			fanficAddCommentMedia(t, repos, ownersComment, "/uploads/images/one.png", "/uploads/images/one_thumb.png")
			fanficAddCommentMedia(t, repos, ownersComment, "/uploads/images/one_no_thumb.gif", "")
			fanficAddCommentMedia(t, repos, commentersComment, "/uploads/images/two.png", "/uploads/images/two_thumb.png")

			deleter := other.ID
			if tc.byOwner {
				deleter = owner.ID
			}

			// when
			paths, err := repos.Fanfic.DeleteFanfic(ctx, spec.FanficDelete{ID: id, UserID: deleter, AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.ElementsMatch(t, tc.wantPaths, paths)

			row, err := repos.Fanfic.GetByID(ctx, spec.FanficLookup{ID: id, ViewerID: owner.ID})
			require.NoError(t, err)
			assert.Equal(t, tc.wantErr, row != nil, "only a refused delete leaves the fanfic in place")
		})
	}
}

func TestFanficDAO_List(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	reader := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	fanficInsert(t, repos, spec.NewFanficWithDetails{
		NewFanfic:  spec.NewFanfic{UserID: author.ID, Title: "Golden Witch", Summary: "summary", Status: "completed"},
		Genres:     []string{"Drama", "Mystery"},
		Tags:       []string{"fluff"},
		Characters: []dto.FanficCharacter{{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"}},
	})
	fanficInsert(t, repos, spec.NewFanficWithDetails{
		NewFanfic:  spec.NewFanfic{UserID: author.ID, Title: "Other", Summary: "golden text here", Rating: "T", IsPairing: true},
		Genres:     []string{"Drama"},
		Tags:       []string{"angst"},
		Characters: makeFanficChars(),
	})
	fanficInsert(t, repos, spec.NewFanficWithDetails{
		NewFanfic:  spec.NewFanfic{UserID: author.ID, Title: "Higu", Series: "Higurashi"},
		Characters: []dto.FanficCharacter{{Series: "Higurashi", CharacterID: "rena", CharacterName: "Rena"}},
	})
	fanficInsert(t, repos, spec.NewFanficWithDetails{NewFanfic: spec.NewFanfic{UserID: author.ID, Title: "Spicy", Rating: "M", ContainsLemons: true}})
	fanficInsert(t, repos, spec.NewFanficWithDetails{NewFanfic: spec.NewFanfic{UserID: author.ID, Title: "Draft", Status: "draft"}})
	fanficInsert(t, repos, spec.NewFanficWithDetails{NewFanfic: spec.NewFanfic{UserID: blocked.ID, Title: "Blocked", Language: "Japanese"}})

	cases := []struct {
		name    string
		viewer  uuid.UUID
		params  fanficparams.ListParams
		exclude []uuid.UUID
		want    []string
	}{
		{name: "with no filters every published fanfic without lemons is listed", want: []string{"Golden Witch", "Other", "Higu", "Blocked"}},
		{name: "the author also sees their own draft", viewer: author.ID, want: []string{"Golden Witch", "Other", "Higu", "Draft", "Blocked"}},
		{name: "another user never sees the author's draft", viewer: reader.ID, want: []string{"Golden Witch", "Other", "Higu", "Blocked"}},
		{name: "show lemons adds the fanfics with lemons", params: fanficparams.ListParams{ShowLemons: true}, want: []string{"Golden Witch", "Other", "Higu", "Spicy", "Blocked"}},
		{name: "series", params: fanficparams.ListParams{Series: "Higurashi"}, want: []string{"Higu"}},
		{name: "rating", params: fanficparams.ListParams{Rating: "T"}, want: []string{"Other"}},
		{name: "language", params: fanficparams.ListParams{Language: "Japanese"}, want: []string{"Blocked"}},
		{name: "status", params: fanficparams.ListParams{Status: "completed"}, want: []string{"Golden Witch"}},
		{name: "both genres must match", params: fanficparams.ListParams{GenreA: "Drama", GenreB: "Mystery"}, want: []string{"Golden Witch"}},
		{name: "tag", params: fanficparams.ListParams{Tag: "angst"}, want: []string{"Other"}},
		{name: "character", params: fanficparams.ListParams{CharacterA: "Battler"}, want: []string{"Golden Witch", "Other"}},
		{name: "pairing keeps only fanfics where the character is in a pairing", params: fanficparams.ListParams{CharacterA: "Battler", IsPairing: true}, want: []string{"Other"}},
		{name: "search matches the title or the summary, ignoring case", params: fanficparams.ListParams{Search: "golden"}, want: []string{"Golden Witch", "Other"}},
		{name: "excluded users' fanfics are left out", exclude: []uuid.UUID{blocked.ID}, want: []string{"Golden Witch", "Other", "Higu"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			params := tc.params
			params.Limit = 10
			rows, total, err := repos.Fanfic.List(ctx, spec.FanficListFilter{ViewerID: tc.viewer, Params: params, ExcludeUserIDs: tc.exclude})

			// then
			require.NoError(t, err)
			assert.Equal(t, len(tc.want), total)
			assert.ElementsMatch(t, tc.want, fanficTitles(rows))

			if tc.params.Series != "" {
				for _, row := range rows {
					assert.Equal(t, tc.params.Series, row.Series)
				}
			}
		})
	}
}

func TestFanficDAO_List_SortFavourites(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	favourited := createFanfic(t, repos, author.ID, "A")
	unfavourited := createFanfic(t, repos, author.ID, "B")
	require.NoError(t, repos.Fanfic.Favourite(ctx, spec.FanficUserRef{UserID: voter.ID, FanficID: favourited}))

	// when
	rows, _, err := repos.Fanfic.List(ctx, spec.FanficListFilter{ViewerID: author.ID, Params: fanficparams.ListParams{Sort: "favourites", Limit: 10}})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, favourited, rows[0].ID, "the favourited fanfic outranks the more recently updated one")
	assert.Equal(t, unfavourited, rows[1].ID)
}

func TestFanficDAO_List_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createFanfic(t, repos, user.ID, "X")
	}

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), spec.FanficListFilter{ViewerID: user.ID, Params: fanficparams.ListParams{Limit: 2, Offset: 1}})

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestFanficDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	reader := daotest.CreateUser(t, repos)
	createFanfic(t, repos, author.ID, "A")
	createFanfic(t, repos, author.ID, "B")
	fanficInsert(t, repos, spec.NewFanficWithDetails{NewFanfic: spec.NewFanfic{UserID: author.ID, Title: "Draft", Status: "draft"}})
	createFanfic(t, repos, reader.ID, "Reader's own")

	cases := []struct {
		name   string
		viewer uuid.UUID
		want   []string
	}{
		{name: "the author sees every fanfic of theirs, drafts included, and nobody else's", viewer: author.ID, want: []string{"A", "B", "Draft"}},
		{name: "another user sees only the published ones", viewer: reader.ID, want: []string{"A", "B"}},
		{name: "an anonymous viewer sees only the published ones", viewer: uuid.Nil, want: []string{"A", "B"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Fanfic.ListByUser(ctx, spec.FanficUserListFilter{UserID: author.ID, ViewerID: tc.viewer, Limit: 10, Offset: 0})

			// then
			require.NoError(t, err)
			assert.Equal(t, len(tc.want), total)
			assert.ElementsMatch(t, tc.want, fanficTitles(rows))
		})
	}
}

func TestFanficDAO_ListByUser_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createFanfic(t, repos, user.ID, "X")
	}

	// when
	rows, total, err := repos.Fanfic.ListByUser(context.Background(), spec.FanficUserListFilter{UserID: user.ID, ViewerID: user.ID, Limit: 1, Offset: 1})

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 1)
}

func TestFanficDAO_Chapters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	created, err := repos.Fanfic.CreateChapter(ctx, spec.NewChapter{FanficID: fid, Number: 1, Title: "Ch 1", Body: "body", WordCount: 10})

	// then
	require.NoError(t, err)

	chapter := fanficChapter(t, repos, fid, 1)
	require.NotNil(t, chapter)
	assert.Equal(t, created.ID, chapter.ID)
	assert.Equal(t, "Ch 1", chapter.Title)
	assert.Equal(t, 10, chapter.WordCount)
	assert.Nil(t, fanficChapter(t, repos, fid, 99))

	// when the chapter is rewritten
	err = repos.Fanfic.UpdateChapter(ctx, spec.ChapterUpdate{ID: created.ID, Title: "New", Body: "new body", WordCount: 50})

	// then
	require.NoError(t, err)

	chapter = fanficChapter(t, repos, fid, 1)
	require.NotNil(t, chapter)
	assert.Equal(t, "New", chapter.Title)
	assert.Equal(t, "new body", chapter.Body)
	assert.Equal(t, 50, chapter.WordCount)

	// when the chapter is deleted
	err = repos.Fanfic.DeleteChapter(ctx, created.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, fanficChapter(t, repos, fid, 1))
}

func TestFanficDAO_ChapterLookups(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	firstNumber, err := repos.Fanfic.GetNextChapterNumber(ctx, fid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, firstNumber)

	// when chapters are added out of order and with a gap, then looked up
	createFanficChapter(t, repos, fid, 2)
	chapterID := createFanficChapter(t, repos, fid, 4)
	createFanficChapter(t, repos, fid, 1)

	chapters, listErr := repos.Fanfic.ListChapters(ctx, fid)
	count, countErr := repos.Fanfic.GetChapterCount(ctx, fid)
	next, nextErr := repos.Fanfic.GetNextChapterNumber(ctx, fid)
	fanficID, fanficIDErr := repos.Fanfic.GetChapterFanficID(ctx, chapterID)
	authorID, authorIDErr := repos.Fanfic.GetChapterAuthorID(ctx, chapterID)

	// then
	require.NoError(t, listErr)

	var numbers []int
	for _, chapter := range chapters {
		numbers = append(numbers, chapter.ChapterNum)
	}

	assert.Equal(t, []int{1, 2, 4}, numbers)

	require.NoError(t, countErr)
	assert.Equal(t, 3, count)

	require.NoError(t, nextErr)
	assert.Equal(t, 5, next)

	require.NoError(t, fanficIDErr)
	assert.Equal(t, fid, fanficID)

	require.NoError(t, authorIDErr)
	assert.Equal(t, user.ID, authorID)
}

func TestFanficDAO_ChaptersWithCount_KeepTheWordCountInStep(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	first, err := repos.Fanfic.CreateChapterWithCount(ctx, spec.NewChapter{FanficID: fid, Number: 1, Title: "c1", Body: "body", WordCount: 500})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, first.ID)
	assert.Equal(t, 500, fanficRow(t, repos, fid, user.ID).WordCount)

	// when a second chapter is added
	second, err := repos.Fanfic.CreateChapterWithCount(ctx, spec.NewChapter{FanficID: fid, Number: 2, Title: "c2", Body: "body", WordCount: 750})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1250, fanficRow(t, repos, fid, user.ID).WordCount)

	// when the first chapter is rewritten
	err = repos.Fanfic.UpdateChapterWithCount(ctx, spec.ChapterUpdate{ID: first.ID, Title: "New", Body: "new body", WordCount: 40})

	// then
	require.NoError(t, err)
	assert.Equal(t, 790, fanficRow(t, repos, fid, user.ID).WordCount)

	// when every chapter is deleted
	require.NoError(t, repos.Fanfic.DeleteChapterWithCount(ctx, first.ID))
	err = repos.Fanfic.DeleteChapterWithCount(ctx, second.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, fanficChapter(t, repos, fid, 1))
	assert.Nil(t, fanficChapter(t, repos, fid, 2))
	assert.Equal(t, 0, fanficRow(t, repos, fid, user.ID).WordCount)
}

func TestFanficDAO_GenresTagsAndCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	first := createFanfic(t, repos, user.ID, "A")
	second := fanficInsert(t, repos, spec.NewFanficWithDetails{
		NewFanfic:  spec.NewFanfic{UserID: user.ID, Title: "B"},
		Genres:     []string{"Horror"},
		Tags:       []string{"gore"},
		Characters: []dto.FanficCharacter{{Series: "Higurashi", CharacterID: "rena", CharacterName: "Rena"}},
	})
	ids := []uuid.UUID{first, second}

	// when the genres are fetched for one fanfic and for a batch
	genres, err := repos.Fanfic.GetGenres(ctx, first)
	genresBatch, batchErr := repos.Fanfic.GetGenresBatch(ctx, ids)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"Drama", "Mystery"}, genres)

	require.NoError(t, batchErr)
	assert.Equal(t, []string{"Drama", "Mystery"}, genresBatch[first])
	assert.Equal(t, []string{"Horror"}, genresBatch[second])

	// when the tags are fetched for one fanfic and for a batch
	tags, err := repos.Fanfic.GetTags(ctx, first)
	tagsBatch, batchErr := repos.Fanfic.GetTagsBatch(ctx, ids)

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"angst", "fluff"}, tags)

	require.NoError(t, batchErr)
	assert.Equal(t, []string{"angst", "fluff"}, tagsBatch[first])
	assert.Equal(t, []string{"gore"}, tagsBatch[second])

	// when the characters are fetched for one fanfic and for a batch
	chars, err := repos.Fanfic.GetCharacters(ctx, first)
	charsBatch, batchErr := repos.Fanfic.GetCharactersBatch(ctx, ids)

	// then
	require.NoError(t, err)
	require.Len(t, chars, 2)
	assert.Equal(t, "Battler", chars[0].CharacterName)
	assert.Equal(t, "Beatrice", chars[1].CharacterName)

	require.NoError(t, batchErr)
	assert.Len(t, charsBatch[first], 2)
	require.Len(t, charsBatch[second], 1)
	assert.Equal(t, "Rena", charsBatch[second][0].CharacterName)

	// when a batch is asked for no ids
	noGenres, genresErr := repos.Fanfic.GetGenresBatch(ctx, nil)
	noTags, tagsErr := repos.Fanfic.GetTagsBatch(ctx, nil)
	noChars, charsErr := repos.Fanfic.GetCharactersBatch(ctx, nil)

	// then
	require.NoError(t, genresErr)
	assert.Nil(t, noGenres)

	require.NoError(t, tagsErr)
	assert.Nil(t, noTags)

	require.NoError(t, charsErr)
	assert.Nil(t, noChars)
}

func TestFanficDAO_OCCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	for _, name := range []string{"Alice", "Bob", "Alicia", "Alice"} {
		require.NoError(t, repos.Fanfic.RegisterOCCharacter(ctx, spec.NewFanficOCCharacter{Name: name, CreatorID: user.ID}))
	}

	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{name: "a query matches every name containing it", query: "Ali", want: []string{"Alice", "Alicia"}},
		{name: "an empty query returns every name, a re-registered one only once", query: "", want: []string{"Alice", "Alicia", "Bob"}},
		{name: "a query nothing matches returns nil", query: "Zed", want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := repos.Fanfic.SearchOCCharacters(ctx, tc.query)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFanficDAO_RegisterLanguageAndSeries(t *testing.T) {
	cases := []struct {
		name     string
		register func(repository.FanficRepository, context.Context, string, ...*sql.Tx) error
		list     func(repository.FanficRepository, context.Context, ...*sql.Tx) ([]string, error)
		seeded   []string
		added    string
	}{
		{name: "languages are seeded and a new one is listed once however often it is registered", register: repository.FanficRepository.RegisterLanguage, list: repository.FanficRepository.GetLanguages, seeded: []string{"English", "Japanese"}, added: "Klingon"},
		{name: "series are seeded and a new one is listed once however often it is registered", register: repository.FanficRepository.RegisterSeries, list: repository.FanficRepository.GetSeries, seeded: []string{"Umineko", "Higurashi"}, added: "Rose Guns Days"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			before, err := tc.list(repos.Fanfic, ctx)
			require.NoError(t, err)

			// when
			require.NoError(t, tc.register(repos.Fanfic, ctx, tc.added))
			err = tc.register(repos.Fanfic, ctx, tc.added)

			// then
			require.NoError(t, err)
			assert.Subset(t, before, tc.seeded)
			assert.NotContains(t, before, tc.added)

			after, err := tc.list(repos.Fanfic, ctx)
			require.NoError(t, err)
			assert.Contains(t, after, tc.added)
			assert.Len(t, after, len(before)+1)
		})
	}
}

func TestFanficDAO_Favourite(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, author.ID, "T")
	ref := spec.FanficUserRef{UserID: voter.ID, FanficID: fid}

	// when
	err := repos.Fanfic.Favourite(ctx, ref)

	// then
	require.NoError(t, err)

	row := fanficRow(t, repos, fid, voter.ID)
	assert.Equal(t, 1, row.FavouriteCount)
	assert.True(t, row.UserFavourited)

	// when the same user favourites it again
	err = repos.Fanfic.Favourite(ctx, ref)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, fanficRow(t, repos, fid, voter.ID).FavouriteCount)

	// when the favourite is taken back
	err = repos.Fanfic.Unfavourite(ctx, ref)

	// then
	require.NoError(t, err)

	row = fanficRow(t, repos, fid, voter.ID)
	assert.Equal(t, 0, row.FavouriteCount)
	assert.False(t, row.UserFavourited)
}

func TestFanficDAO_ListFavourites(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	for _, title := range []string{"A", "B"} {
		id := createFanfic(t, repos, author.ID, title)
		require.NoError(t, repos.Fanfic.Favourite(ctx, spec.FanficUserRef{UserID: voter.ID, FanficID: id}))
	}

	listing := spec.FanficUserListFilter{UserID: voter.ID, ViewerID: voter.ID, Limit: 10, Offset: 0}

	// when
	rows, total, err := repos.Fanfic.ListFavourites(ctx, listing)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.ElementsMatch(t, []string{"A", "B"}, fanficTitles(rows))

	// when the user also favourites someone else's draft
	draft := fanficInsert(t, repos, spec.NewFanficWithDetails{NewFanfic: spec.NewFanfic{UserID: author.ID, Title: "Draft", Status: "draft"}})
	require.NoError(t, repos.Fanfic.Favourite(ctx, spec.FanficUserRef{UserID: voter.ID, FanficID: draft}))
	rows, total, err = repos.Fanfic.ListFavourites(ctx, listing)

	// then the hidden draft is left out of the total as well as the rows
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"A", "B"}, fanficTitles(rows))
	assert.Equal(t, 2, total, "the total must count only the favourites the viewer can see")

	// when the draft's own author views that list
	rows, total, err = repos.Fanfic.ListFavourites(ctx, spec.FanficUserListFilter{UserID: voter.ID, ViewerID: author.ID, Limit: 10, Offset: 0})

	// then the author sees their draft counted too
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"A", "B", "Draft"}, fanficTitles(rows))
	assert.Equal(t, 3, total)
}

func TestFanficDAO_RecordView(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	view := spec.ViewRecord{TargetID: fid, ViewerHash: "hash1"}

	// when
	inserted, err := repos.Fanfic.RecordView(ctx, view)

	// then
	require.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, 1, fanficRow(t, repos, fid, user.ID).ViewCount)

	// when the same viewer is recorded again
	inserted, err = repos.Fanfic.RecordView(ctx, view)

	// then
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Equal(t, 1, fanficRow(t, repos, fid, user.ID).ViewCount)
}

func TestFanficDAO_ReadingProgress(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	ref := spec.FanficUserRef{UserID: user.ID, FanficID: fid}

	// when
	got, err := repos.Fanfic.GetReadingProgress(ctx, ref)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, got)

	// when progress is saved
	require.NoError(t, repos.Fanfic.SetReadingProgress(ctx, spec.FanficReadingProgress{UserID: user.ID, FanficID: fid, ChapterNumber: 3}))
	got, err = repos.Fanfic.GetReadingProgress(ctx, ref)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, got)

	// when progress is saved again
	require.NoError(t, repos.Fanfic.SetReadingProgress(ctx, spec.FanficReadingProgress{UserID: user.ID, FanficID: fid, ChapterNumber: 5}))
	got, err = repos.Fanfic.GetReadingProgress(ctx, ref)

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, got)

	// when the progress cannot be read at all
	_, err = repos.DB().ExecContext(ctx, `ALTER TABLE fanfic_reading_progress RENAME COLUMN chapter_number TO chapter_number_gone`)
	require.NoError(t, err)
	_, err = repos.Fanfic.GetReadingProgress(ctx, ref)

	// then the failure surfaces instead of reading as "not started"
	require.Error(t, err)
}

func TestFanficDAO_CreateComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	author := daotest.CreateUser(t, repos)
	replier := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, author.ID, "T")

	// when
	parentID := createFanficComment(t, repos, fid, nil, author.ID, "Nice!")
	childID := createFanficComment(t, repos, fid, &parentID, replier.ID, "child")

	// then
	comments := fanficComments(t, repos, fid, author.ID)
	require.Len(t, comments, 2)

	parent := fanficCommentByID(t, comments, parentID)
	assert.Equal(t, "Nice!", parent.Body)
	assert.Nil(t, parent.ParentID)

	child := fanficCommentByID(t, comments, childID)
	assert.Equal(t, "child", child.Body)
	require.NotNil(t, child.ParentID)
	assert.Equal(t, parentID, *child.ParentID)

	// when the reply is traced back to its fanfic and its author
	entityID, entityIDErr := repos.Fanfic.GetCommentEntityID(ctx, childID)
	authorID, authorIDErr := repos.Fanfic.GetCommentAuthorID(ctx, childID)

	// then
	require.NoError(t, entityIDErr)
	assert.Equal(t, fid, entityID)

	require.NoError(t, authorIDErr)
	assert.Equal(t, replier.ID, authorID)

	// when the replier is excluded
	visible, total, err := repos.Fanfic.GetComments(ctx, spec.CommentQuery[uuid.UUID]{TargetID: fid, ViewerID: author.ID, Limit: 500, Offset: 0, ExcludeUserIDs: []uuid.UUID{replier.ID}})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, visible, 1)
	assert.Equal(t, "Nice!", visible[0].Body)
}

func TestFanficDAO_UpdateComment(t *testing.T) {
	cases := []struct {
		name     string
		byOwner  bool
		asAdmin  bool
		wantErr  bool
		wantBody string
	}{
		{name: "the author edits their own comment", byOwner: true, asAdmin: false, wantErr: false, wantBody: "edited"},
		{name: "an admin edits someone else's comment", byOwner: false, asAdmin: true, wantErr: false, wantBody: "edited"},
		{name: "another user is refused and the comment is left alone", byOwner: false, asAdmin: false, wantErr: true, wantBody: "old"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			fid := createFanfic(t, repos, owner.ID, "T")
			cid := createFanficComment(t, repos, fid, nil, owner.ID, "old")

			editor := other.ID
			if tc.byOwner {
				editor = owner.ID
			}

			// when
			err := repos.Fanfic.UpdateComment(context.Background(), spec.CommentUpdate{CommentID: cid, UserID: editor, Body: "edited", AsAdmin: tc.asAdmin})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			comments := fanficComments(t, repos, fid, owner.ID)
			require.Len(t, comments, 1)
			assert.Equal(t, tc.wantBody, comments[0].Body)
		})
	}
}

func TestFanficDAO_DeleteCommentWithAudit(t *testing.T) {
	targetPaths := []string{"/uploads/images/target.png", "/uploads/images/target_thumb.png", "/uploads/images/target_no_thumb.gif"}

	cases := []struct {
		name          string
		byOwner       bool
		asAdmin       bool
		action        audit.Action
		wantErr       bool
		wantPaths     []string
		wantRemaining []string
		wantAudits    int
	}{
		{name: "the author deletes their comment, gets back only its media and an audit entry is written", byOwner: true, asAdmin: false, action: audit.ActionFanficCommentDelete, wantErr: false, wantPaths: targetPaths, wantRemaining: []string{"sibling"}, wantAudits: 1},
		{name: "an admin deletes someone else's comment and the audit entry names the admin", byOwner: false, asAdmin: true, action: audit.ActionFanficCommentDeleteAdmin, wantErr: false, wantPaths: targetPaths, wantRemaining: []string{"sibling"}, wantAudits: 1},
		{name: "another user is refused, gets no paths back and no audit entry is written", byOwner: false, asAdmin: false, action: audit.ActionFanficCommentDelete, wantErr: true, wantPaths: nil, wantRemaining: []string{"target", "sibling"}, wantAudits: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			fid := createFanfic(t, repos, owner.ID, "T")
			target := createFanficComment(t, repos, fid, nil, owner.ID, "target")
			sibling := createFanficComment(t, repos, fid, nil, owner.ID, "sibling")
			fanficAddCommentMedia(t, repos, target, "/uploads/images/target.png", "/uploads/images/target_thumb.png")
			fanficAddCommentMedia(t, repos, target, "/uploads/images/target_no_thumb.gif", "")
			fanficAddCommentMedia(t, repos, sibling, "/uploads/images/sibling.png", "")

			deleter := other.ID
			if tc.byOwner {
				deleter = owner.ID
			}

			// when
			paths, err := repos.Fanfic.DeleteCommentWithAudit(ctx, spec.FanficCommentDelete{
				CommentDeletion: spec.CommentDeletion{CommentID: target, UserID: deleter, AsAdmin: tc.asAdmin},
				Audit:           audit.NewEntry{ActorID: deleter, Action: tc.action, TargetType: audit.TargetFanficComment, TargetID: target.String()},
			})

			// then
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.ElementsMatch(t, tc.wantPaths, paths)
			assert.ElementsMatch(t, tc.wantRemaining, fanficCommentBodies(fanficComments(t, repos, fid, owner.ID)))

			siblingMedia, err := repos.Fanfic.GetCommentMedia(ctx, sibling)
			require.NoError(t, err)
			assert.Len(t, siblingMedia, 1)

			entries, auditTotal, err := repos.AuditLog.List(ctx, spec.AuditLogListing{Action: tc.action, Page: bounds.NewPage(10, 0)})
			require.NoError(t, err)
			assert.Equal(t, tc.wantAudits, auditTotal)
			require.Len(t, entries, tc.wantAudits)
			for _, entry := range entries {
				assert.Equal(t, target.String(), entry.TargetID)
				assert.Equal(t, deleter, entry.ActorID)
			}
		})
	}
}

func TestFanficDAO_LikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	like := spec.CommentLike{UserID: liker.ID, CommentID: createFanficComment(t, repos, fid, nil, owner.ID, "body")}

	// when
	err := repos.Fanfic.LikeComment(ctx, like)

	// then
	require.NoError(t, err)

	comments := fanficComments(t, repos, fid, liker.ID)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)

	// when the same user likes it again
	err = repos.Fanfic.LikeComment(ctx, like)

	// then
	require.NoError(t, err)

	comments = fanficComments(t, repos, fid, liker.ID)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)

	// when the like is taken back
	err = repos.Fanfic.UnlikeComment(ctx, like)

	// then
	require.NoError(t, err)

	comments = fanficComments(t, repos, fid, liker.ID)
	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].LikeCount)
	assert.False(t, comments[0].UserLiked)
}

func TestFanficDAO_CommentMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	commentID := createFanficComment(t, repos, fid, nil, user.ID, "a")
	otherID := createFanficComment(t, repos, fid, nil, user.ID, "b")

	// when
	firstID := fanficAddCommentMedia(t, repos, commentID, "http://x/0.png", "http://x/0_t.png")
	secondID := fanficAddCommentMedia(t, repos, commentID, "http://x/1.png", "")
	fanficAddCommentMedia(t, repos, commentID, "http://x/2.png", "")
	fanficAddCommentMedia(t, repos, otherID, "http://x/other.png", "")

	// then
	assert.NotZero(t, firstID)

	media, err := repos.Fanfic.GetCommentMedia(ctx, commentID)
	require.NoError(t, err)
	require.Len(t, media, 3)
	assert.Equal(t, "http://x/0.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
	assert.Equal(t, "http://x/1.png", media[1].MediaURL)
	assert.Equal(t, "http://x/2.png", media[2].MediaURL)

	// when one item's url and thumbnail are replaced
	require.NoError(t, repos.Fanfic.UpdateCommentMediaURL(ctx, spec.MediaURLUpdate{ID: secondID, URL: "http://new/1.png"}))
	err = repos.Fanfic.UpdateCommentMediaThumbnail(ctx, spec.MediaURLUpdate{ID: secondID, URL: "http://x/1_t.png"})

	// then
	require.NoError(t, err)

	media, err = repos.Fanfic.GetCommentMedia(ctx, commentID)
	require.NoError(t, err)
	require.Len(t, media, 3)
	assert.Equal(t, "http://new/1.png", media[1].MediaURL)
	assert.Equal(t, "http://x/1_t.png", media[1].ThumbnailURL)

	// when the media is fetched in a batch
	batch, batchErr := repos.Fanfic.GetCommentMediaBatch(ctx, []uuid.UUID{commentID, otherID})
	none, noneErr := repos.Fanfic.GetCommentMediaBatch(ctx, nil)

	// then
	require.NoError(t, batchErr)
	assert.Len(t, batch[commentID], 3)
	assert.Len(t, batch[otherID], 1)

	require.NoError(t, noneErr)
	assert.Nil(t, none)
}
