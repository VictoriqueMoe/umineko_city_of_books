package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	fanficparams "umineko_city_of_books/internal/fanfic/params"
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

func createFanfic(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string) uuid.UUID {
	t.Helper()
	created, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     userID,
		Title:      title,
		Summary:    "summary",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Genres:     []string{"Drama", "Mystery"},
		Tags:       []string{"angst", "fluff"},
		Characters: makeFanficChars(),
	})
	require.NoError(t, err)
	return created.ID
}

func createFanficChapter(t *testing.T, repos *repository.Repositories, fanficID uuid.UUID, chapterNumber int, title string) uuid.UUID {
	t.Helper()
	created, err := repos.Fanfic.CreateChapter(context.Background(), fanficID, repository.NewChapter{
		Number:    chapterNumber,
		Title:     title,
		Body:      "body text",
		WordCount: 100,
	})
	require.NoError(t, err)
	return created.ID
}

func TestFanficDAO_CreateWithDetails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     user.ID,
		Title:      "Title",
		Summary:    "Summary",
		Series:     "Umineko",
		Rating:     "T",
		Language:   "English",
		Status:     "in_progress",
		IsOneshot:  true,
		Genres:     []string{"Drama"},
		Tags:       []string{"sadtag"},
		Characters: makeFanficChars(),
		IsPairing:  true,
	})

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), created.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Title", row.Title)
	assert.Equal(t, "Summary", row.Summary)
	assert.True(t, row.IsOneshot)
	assert.False(t, row.ContainsLemons)
	assert.True(t, row.IsPairing)
}

func TestFanficDAO_CreateWithDetails_TrimsCharacterName(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	chars := []dto.FanficCharacter{{Series: "Umineko", CharacterID: "x", CharacterName: "  Padded  "}}

	// when
	created, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     user.ID,
		Title:      "T",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Characters: chars,
	})

	// then
	require.NoError(t, err)
	got, err := repos.Fanfic.GetCharacters(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Padded", got[0].CharacterName)
}

func TestFanficDAO_CreateWithDetails_SkipsEmptyTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "T",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
		Tags:     []string{"  ", "", "keep"},
	})

	// then
	require.NoError(t, err)
	tags, err := repos.Fanfic.GetTags(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"keep"}, tags)
}

func TestFanficDAO_UpdateWithDetails_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Old")

	// when
	err := repos.Fanfic.UpdateWithDetails(context.Background(), repository.FanficUpdate{
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
		Genres:         []string{"Angst"},
		Tags:           []string{"newtag"},
		Characters:     []dto.FanficCharacter{{Series: "Higurashi", CharacterID: "rena", CharacterName: "Rena"}},
	})

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "New", row.Title)
	assert.Equal(t, "Higurashi", row.Series)
	assert.Equal(t, "completed", row.Status)
	assert.True(t, row.ContainsLemons)
}

func TestFanficDAO_UpdateWithDetails_NonOwnerFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWithDetails(context.Background(), repository.FanficUpdate{
		ID:       id,
		UserID:   other.ID,
		Title:    "New",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
	})

	// then
	require.Error(t, err)
}

func TestFanficDAO_UpdateWithDetails_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWithDetails(context.Background(), repository.FanficUpdate{
		ID:       id,
		UserID:   admin.ID,
		Title:    "AdminEdit",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
		AsAdmin:  true,
	})

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "AdminEdit", row.Title)
}

func TestFanficDAO_UpdateWithDetails_ReplacesGenresTagsCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWithDetails(context.Background(), repository.FanficUpdate{
		ID:         id,
		UserID:     user.ID,
		Title:      "Title",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Genres:     []string{"Horror"},
		Tags:       []string{"replaced"},
		Characters: []dto.FanficCharacter{{Series: "Umineko", CharacterID: "ange", CharacterName: "Ange"}},
	})

	// then
	require.NoError(t, err)
	genres, err := repos.Fanfic.GetGenres(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, []string{"Horror"}, genres)
	tags, err := repos.Fanfic.GetTags(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, []string{"replaced"}, tags)
	chars, err := repos.Fanfic.GetCharacters(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, chars, 1)
	assert.Equal(t, "Ange", chars[0].CharacterName)
}

func TestFanficDAO_UpdateCoverImage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.UpdateCoverImage(context.Background(), id, "https://img/x.png", "https://img/x_t.png")

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://img/x.png", row.CoverImageURL)
	assert.Equal(t, "https://img/x_t.png", row.CoverThumbnailURL)
}

func TestFanficDAO_UpdateWordCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")
	_, err := repos.Fanfic.CreateChapter(context.Background(), id, repository.NewChapter{
		Number:    1,
		Title:     "c1",
		Body:      "body",
		WordCount: 500,
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateChapter(context.Background(), id, repository.NewChapter{
		Number:    2,
		Title:     "c2",
		Body:      "body",
		WordCount: 750,
	})
	require.NoError(t, err)

	// when
	err = repos.Fanfic.UpdateWordCount(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1250, row.WordCount)
}

func TestFanficDAO_UpdateWordCount_NoChapters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWordCount(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.WordCount)
}

func TestFanficDAO_Delete_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficDAO_Delete_NonOwnerFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.Delete(context.Background(), id, other.ID)

	// then
	require.Error(t, err)
}

func TestFanficDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficDAO_DeleteFanfic_ReturnsCoverAndCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")
	require.NoError(t, repos.Fanfic.UpdateCoverImage(context.Background(), id, "/uploads/images/cover.png", "/uploads/images/cover_thumb.png"))
	comment, err := repos.Fanfic.CreateComment(context.Background(), id, nil, user.ID, "body")
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID:    comment.ID,
		MediaURL:     "/uploads/images/comment.png",
		MediaType:    "image",
		ThumbnailURL: "/uploads/images/comment_thumb.png",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: comment.ID,
		MediaURL:  "/uploads/images/comment_no_thumb.gif",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	paths, err := repos.Fanfic.DeleteFanfic(context.Background(), repository.FanficDelete{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/images/cover.png",
		"/uploads/images/cover_thumb.png",
		"/uploads/images/comment.png",
		"/uploads/images/comment_thumb.png",
		"/uploads/images/comment_no_thumb.gif",
	}, paths)
	assert.NotContains(t, paths, "")
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficDAO_DeleteFanfic_AsAdmin_CollectsEveryCommentMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")
	require.NoError(t, repos.Fanfic.UpdateCoverImage(context.Background(), id, "/uploads/images/cover.png", ""))
	first, err := repos.Fanfic.CreateComment(context.Background(), id, nil, owner.ID, "one")
	require.NoError(t, err)
	second, err := repos.Fanfic.CreateComment(context.Background(), id, nil, commenter.ID, "two")
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: first.ID,
		MediaURL:  "/uploads/images/one.png",
		MediaType: "image",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID:    second.ID,
		MediaURL:     "/uploads/images/two.png",
		MediaType:    "image",
		ThumbnailURL: "/uploads/images/two_thumb.png",
	})
	require.NoError(t, err)

	// when
	paths, err := repos.Fanfic.DeleteFanfic(context.Background(), repository.FanficDelete{ID: id, UserID: admin.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/images/cover.png",
		"/uploads/images/one.png",
		"/uploads/images/two.png",
		"/uploads/images/two_thumb.png",
	}, paths)
	assert.NotContains(t, paths, "")
	row, err := repos.Fanfic.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficDAO_DeleteFanfic_NonOwnerReturnsNoPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")
	require.NoError(t, repos.Fanfic.UpdateCoverImage(context.Background(), id, "/uploads/images/cover.png", "/uploads/images/cover_thumb.png"))

	// when
	paths, err := repos.Fanfic.DeleteFanfic(context.Background(), repository.FanficDelete{ID: id, UserID: other.ID})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	row, err := repos.Fanfic.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
}

func TestFanficDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	row, err := repos.Fanfic.GetByID(context.Background(), uuid.New(), user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficDAO_GetByID_IncludesAuthorDetails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author Name"))
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Author Name", row.AuthorDisplayName)
	assert.Equal(t, user.Username, row.AuthorUsername)
}

func TestFanficDAO_GetAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	got, err := repos.Fanfic.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestFanficDAO_GetAuthorID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Fanfic.GetAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestFanficDAO_List_Defaults(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "A")
	createFanfic(t, repos, user.ID, "B")

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestFanficDAO_List_HidesDraftsFromOthers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   owner.ID,
		Title:    "Draft",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "draft",
	})
	require.NoError(t, err)

	// when
	_, totalOther, err := repos.Fanfic.List(context.Background(), other.ID, fanficparams.ListParams{Limit: 10}, nil)
	require.NoError(t, err)
	_, totalOwner, err := repos.Fanfic.List(context.Background(), owner.ID, fanficparams.ListParams{Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, totalOther)
	assert.Equal(t, 1, totalOwner)
}

func TestFanficDAO_List_FiltersLemons(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "Clean")
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:         user.ID,
		Title:          "Spicy",
		Series:         "Umineko",
		Rating:         "M",
		Language:       "English",
		Status:         "in_progress",
		ContainsLemons: true,
	})
	require.NoError(t, err)

	// when
	_, totalNoLemons, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Limit: 10}, nil)
	require.NoError(t, err)
	_, totalWithLemons, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Limit: 10, ShowLemons: true}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, totalNoLemons)
	assert.Equal(t, 2, totalWithLemons)
}

func TestFanficDAO_List_FilterSeries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "Umi")
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Higu",
		Series:   "Higurashi",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
	})
	require.NoError(t, err)

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Series: "Higurashi", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Higurashi", rows[0].Series)
}

func TestFanficDAO_List_FilterRating(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "K one")
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "M one",
		Series:   "Umineko",
		Rating:   "M",
		Language: "English",
		Status:   "in_progress",
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Rating: "M", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_FilterLanguage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "English")
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Jap",
		Series:   "Umineko",
		Rating:   "K",
		Language: "Japanese",
		Status:   "in_progress",
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Language: "Japanese", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_FilterStatus(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "WIP")
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Done",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "completed",
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Status: "completed", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_FilterGenres(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "A",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
		Genres:   []string{"Drama", "Mystery"},
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "B",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
		Genres:   []string{"Drama"},
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{GenreA: "Drama", GenreB: "Mystery", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_FilterTag(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "A",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
		Tags:     []string{"fluff"},
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "B",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
		Tags:     []string{"angst"},
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Tag: "angst", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_FilterCharacter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     user.ID,
		Title:      "A",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Characters: []dto.FanficCharacter{{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"}},
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     user.ID,
		Title:      "B",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Characters: []dto.FanficCharacter{{Series: "Umineko", CharacterID: "rena", CharacterName: "Rena"}},
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{CharacterA: "Battler", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_FilterPairing(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     user.ID,
		Title:      "Single",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Characters: []dto.FanficCharacter{{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"}},
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:     user.ID,
		Title:      "Pair",
		Series:     "Umineko",
		Rating:     "K",
		Language:   "English",
		Status:     "in_progress",
		Characters: []dto.FanficCharacter{{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"}, {Series: "Umineko", CharacterID: "beatrice", CharacterName: "Beatrice"}},
		IsPairing:  true,
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{CharacterA: "Battler", IsPairing: true, Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_List_Search(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	_, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Golden Witch",
		Summary:  "summary",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   user.ID,
		Title:    "Other",
		Summary:  "golden text here",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "in_progress",
	})
	require.NoError(t, err)

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Search: "golden", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
}

func TestFanficDAO_List_SortFavourites(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	a := createFanfic(t, repos, user.ID, "A")
	b := createFanfic(t, repos, user.ID, "B")
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, b))

	// when
	rows, _, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Sort: "favourites", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, b, rows[0].ID)
	assert.Equal(t, a, rows[1].ID)
}

func TestFanficDAO_List_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createFanfic(t, repos, user.ID, "X")
	}

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Limit: 2, Offset: 1}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestFanficDAO_List_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "Mine")
	createFanfic(t, repos, blocked.ID, "Blocked")

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, fanficparams.ListParams{Limit: 10}, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	createFanfic(t, repos, owner.ID, "A")
	createFanfic(t, repos, owner.ID, "B")
	createFanfic(t, repos, other.ID, "C")

	// when
	rows, total, err := repos.Fanfic.ListByUser(context.Background(), owner.ID, owner.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestFanficDAO_ListByUser_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createFanfic(t, repos, user.ID, "X")
	}

	// when
	rows, total, err := repos.Fanfic.ListByUser(context.Background(), user.ID, user.ID, 1, 1)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 1)
}

func TestFanficDAO_CreateChapter_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	cidRow, err := repos.Fanfic.CreateChapter(context.Background(), fid, repository.NewChapter{
		Number:    1,
		Title:     "Ch 1",
		Body:      "body",
		WordCount: 10,
	})

	// then
	require.NoError(t, err)
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 1)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, cidRow.ID, ch.ID)
	assert.Equal(t, "Ch 1", ch.Title)
	assert.Equal(t, 10, ch.WordCount)
}

func TestFanficDAO_GetChapter_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 99)

	// then
	require.NoError(t, err)
	assert.Nil(t, ch)
}

func TestFanficDAO_UpdateChapter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "Old")

	// when
	err := repos.Fanfic.UpdateChapter(context.Background(), cid, "New", "new body", 50)

	// then
	require.NoError(t, err)
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 1)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, "New", ch.Title)
	assert.Equal(t, "new body", ch.Body)
	assert.Equal(t, 50, ch.WordCount)
}

func TestFanficDAO_DeleteChapter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "Ch")

	// when
	err := repos.Fanfic.DeleteChapter(context.Background(), cid)

	// then
	require.NoError(t, err)
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 1)
	require.NoError(t, err)
	assert.Nil(t, ch)
}

func TestFanficDAO_CreateChapterWithCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	created, err := repos.Fanfic.CreateChapterWithCount(context.Background(), fid, repository.NewChapter{
		Number:    1,
		Title:     "Ch 1",
		Body:      "body",
		WordCount: 320,
	})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 320, row.WordCount)
}

func TestFanficDAO_UpdateChapterWithCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "Ch")

	// when
	err := repos.Fanfic.UpdateChapterWithCount(context.Background(), repository.ChapterUpdate{
		ID:        cid,
		Title:     "New",
		Body:      "new body",
		WordCount: 40,
	})

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 40, row.WordCount)
}

func TestFanficDAO_DeleteChapterWithCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "Ch")
	require.NoError(t, repos.Fanfic.UpdateWordCount(context.Background(), fid))

	// when
	err := repos.Fanfic.DeleteChapterWithCount(context.Background(), cid)

	// then
	require.NoError(t, err)
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 1)
	require.NoError(t, err)
	assert.Nil(t, ch)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.WordCount)
}

func TestFanficDAO_ListChapters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	createFanficChapter(t, repos, fid, 2, "B")
	createFanficChapter(t, repos, fid, 1, "A")
	createFanficChapter(t, repos, fid, 3, "C")

	// when
	chs, err := repos.Fanfic.ListChapters(context.Background(), fid)

	// then
	require.NoError(t, err)
	require.Len(t, chs, 3)
	assert.Equal(t, 1, chs[0].ChapterNum)
	assert.Equal(t, 2, chs[1].ChapterNum)
	assert.Equal(t, 3, chs[2].ChapterNum)
}

func TestFanficDAO_GetChapterCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	createFanficChapter(t, repos, fid, 1, "A")
	createFanficChapter(t, repos, fid, 2, "B")

	// when
	n, err := repos.Fanfic.GetChapterCount(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestFanficDAO_GetNextChapterNumber(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	empty, err := repos.Fanfic.GetNextChapterNumber(context.Background(), fid)
	require.NoError(t, err)
	createFanficChapter(t, repos, fid, 1, "A")
	createFanficChapter(t, repos, fid, 4, "D")
	next, err := repos.Fanfic.GetNextChapterNumber(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, empty)
	assert.Equal(t, 5, next)
}

func TestFanficDAO_GetChapterFanficID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "A")

	// when
	got, err := repos.Fanfic.GetChapterFanficID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, fid, got)
}

func TestFanficDAO_GetChapterAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "A")

	// when
	got, err := repos.Fanfic.GetChapterAuthorID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestFanficDAO_GetGenres(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetGenres(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Drama", "Mystery"}, got)
}

func TestFanficDAO_GetGenresBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	f1 := createFanfic(t, repos, user.ID, "A")
	f2 := createFanfic(t, repos, user.ID, "B")

	// when
	got, err := repos.Fanfic.GetGenresBatch(context.Background(), []uuid.UUID{f1, f2})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Drama", "Mystery"}, got[f1])
	assert.ElementsMatch(t, []string{"Drama", "Mystery"}, got[f2])
}

func TestFanficDAO_GetGenresBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetGenresBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFanficDAO_GetTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetTags(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"angst", "fluff"}, got)
}

func TestFanficDAO_GetTagsBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	f1 := createFanfic(t, repos, user.ID, "A")
	f2 := createFanfic(t, repos, user.ID, "B")

	// when
	got, err := repos.Fanfic.GetTagsBatch(context.Background(), []uuid.UUID{f1, f2})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"angst", "fluff"}, got[f1])
	assert.ElementsMatch(t, []string{"angst", "fluff"}, got[f2])
}

func TestFanficDAO_GetTagsBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetTagsBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFanficDAO_GetCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetCharacters(context.Background(), fid)

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Battler", got[0].CharacterName)
	assert.Equal(t, "Beatrice", got[1].CharacterName)
}

func TestFanficDAO_GetCharactersBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	f1 := createFanfic(t, repos, user.ID, "A")
	f2 := createFanfic(t, repos, user.ID, "B")

	// when
	got, err := repos.Fanfic.GetCharactersBatch(context.Background(), []uuid.UUID{f1, f2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[f1], 2)
	assert.Len(t, got[f2], 2)
}

func TestFanficDAO_GetCharactersBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetCharactersBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFanficDAO_RegisterOCCharacter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	err := repos.Fanfic.RegisterOCCharacter(context.Background(), "My OC", user.ID)

	// then
	require.NoError(t, err)
	names, err := repos.Fanfic.SearchOCCharacters(context.Background(), "My")
	require.NoError(t, err)
	assert.Contains(t, names, "My OC")
}

func TestFanficDAO_RegisterOCCharacter_Duplicate(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Dup", user.ID))

	// when
	err := repos.Fanfic.RegisterOCCharacter(context.Background(), "Dup", user.ID)

	// then
	require.NoError(t, err)
	names, err := repos.Fanfic.SearchOCCharacters(context.Background(), "Dup")
	require.NoError(t, err)
	assert.Len(t, names, 1)
}

func TestFanficDAO_SearchOCCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Alice", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Bob", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Alicia", user.ID))

	// when
	got, err := repos.Fanfic.SearchOCCharacters(context.Background(), "Ali")

	// then
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestFanficDAO_SearchOCCharacters_EmptyQueryReturnsAll(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "A1", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "A2", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "A3", user.ID))

	// when
	got, err := repos.Fanfic.SearchOCCharacters(context.Background(), "")

	// then
	require.NoError(t, err)
	assert.Len(t, got, 3)
}

func TestFanficDAO_GetLanguages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	langs, err := repos.Fanfic.GetLanguages(context.Background())

	// then
	require.NoError(t, err)
	assert.Contains(t, langs, "English")
	assert.Contains(t, langs, "Japanese")
}

func TestFanficDAO_RegisterLanguage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	err := repos.Fanfic.RegisterLanguage(context.Background(), "Klingon")

	// then
	require.NoError(t, err)
	langs, err := repos.Fanfic.GetLanguages(context.Background())
	require.NoError(t, err)
	assert.Contains(t, langs, "Klingon")
}

func TestFanficDAO_RegisterLanguage_Duplicate(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	require.NoError(t, repos.Fanfic.RegisterLanguage(context.Background(), "Welsh"))

	// when
	err := repos.Fanfic.RegisterLanguage(context.Background(), "Welsh")

	// then
	require.NoError(t, err)
}

func TestFanficDAO_GetSeries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	series, err := repos.Fanfic.GetSeries(context.Background())

	// then
	require.NoError(t, err)
	assert.Contains(t, series, "Umineko")
	assert.Contains(t, series, "Higurashi")
}

func TestFanficDAO_RegisterSeries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	err := repos.Fanfic.RegisterSeries(context.Background(), "Rose Guns Days")

	// then
	require.NoError(t, err)
	series, err := repos.Fanfic.GetSeries(context.Background())
	require.NoError(t, err)
	assert.Contains(t, series, "Rose Guns Days")
}

func TestFanficDAO_Favourite(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	err := repos.Fanfic.Favourite(context.Background(), voter.ID, fid)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, voter.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, 1, row.FavouriteCount)
	assert.True(t, row.UserFavourited)
}

func TestFanficDAO_Favourite_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, fid))

	// when
	err := repos.Fanfic.Favourite(context.Background(), voter.ID, fid)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.FavouriteCount)
}

func TestFanficDAO_Unfavourite(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, fid))

	// when
	err := repos.Fanfic.Unfavourite(context.Background(), voter.ID, fid)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.FavouriteCount)
	assert.False(t, row.UserFavourited)
}

func TestFanficDAO_RecordView_New(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	inserted, err := repos.Fanfic.RecordView(context.Background(), fid, "hash1")

	// then
	require.NoError(t, err)
	assert.True(t, inserted)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.ViewCount)
}

func TestFanficDAO_RecordView_Duplicate(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	_, err := repos.Fanfic.RecordView(context.Background(), fid, "hash1")
	require.NoError(t, err)

	// when
	inserted, err := repos.Fanfic.RecordView(context.Background(), fid, "hash1")

	// then
	require.NoError(t, err)
	assert.False(t, inserted)
	row, err := repos.Fanfic.GetByID(context.Background(), fid, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.ViewCount)
}

func TestFanficDAO_ReadingProgress_DefaultZero(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetReadingProgress(context.Background(), user.ID, fid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, got)
}

func TestFanficDAO_SetAndGetReadingProgress(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	err := repos.Fanfic.SetReadingProgress(context.Background(), user.ID, fid, 3)

	// then
	require.NoError(t, err)
	got, err := repos.Fanfic.GetReadingProgress(context.Background(), user.ID, fid)
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestFanficDAO_SetReadingProgress_Upsert(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	require.NoError(t, repos.Fanfic.SetReadingProgress(context.Background(), user.ID, fid, 2))

	// when
	err := repos.Fanfic.SetReadingProgress(context.Background(), user.ID, fid, 5)

	// then
	require.NoError(t, err)
	got, err := repos.Fanfic.GetReadingProgress(context.Background(), user.ID, fid)
	require.NoError(t, err)
	assert.Equal(t, 5, got)
}

func TestFanficDAO_ListFavourites(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	a := createFanfic(t, repos, owner.ID, "A")
	b := createFanfic(t, repos, owner.ID, "B")
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, a))
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, b))

	// when
	rows, total, err := repos.Fanfic.ListFavourites(context.Background(), voter.ID, voter.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestFanficDAO_ListFavourites_HidesDrafts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	draft, err := repos.Fanfic.CreateWithDetails(context.Background(), repository.NewFanfic{
		UserID:   owner.ID,
		Title:    "Draft",
		Series:   "Umineko",
		Rating:   "K",
		Language: "English",
		Status:   "draft",
	})
	require.NoError(t, err)
	draftID := draft.ID
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, draftID))

	// when
	rows, _, err := repos.Fanfic.ListFavourites(context.Background(), voter.ID, voter.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Len(t, rows, 0)
}

func TestFanficDAO_CreateComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	_, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "Nice!")

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "Nice!", cs[0].Body)
}

func TestFanficDAO_CreateComment_Threaded(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	parent, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "parent")
	require.NoError(t, err)
	parentID := parent.ID

	// when
	child, err := repos.Fanfic.CreateComment(context.Background(), fid, &parentID, user.ID, "child")

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 2)
	var foundChild bool
	for _, c := range cs {
		if c.ID == child.ID {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parentID, *c.ParentID)
			foundChild = true
		}
	}
	assert.True(t, foundChild)
}

func TestFanficDAO_UpdateComment_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "old")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.UpdateComment(context.Background(), cid, user.ID, "new")

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "new", cs[0].Body)
}

func TestFanficDAO_UpdateComment_NonOwnerFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "old")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.UpdateComment(context.Background(), cid, other.ID, "hack")

	// then
	require.Error(t, err)
}

func TestFanficDAO_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "old")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.UpdateCommentAsAdmin(context.Background(), cid, "admin edit")

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "admin edit", cs[0].Body)
}

func TestFanficDAO_DeleteComment_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.DeleteComment(context.Background(), cid, user.ID)

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, 500, 0, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 0)
}

func TestFanficDAO_DeleteComment_NonOwnerFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.DeleteComment(context.Background(), cid, other.ID)

	// then
	require.Error(t, err)
}

func TestFanficDAO_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.DeleteCommentAsAdmin(context.Background(), cid)

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, 500, 0, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 0)
}

func TestFanficDAO_DeleteCommentWithAudit_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	_, err = repos.Fanfic.DeleteCommentWithAudit(context.Background(), repository.FanficCommentDelete{
		ID:     cid,
		UserID: user.ID,
		Audit: repository.NewAuditEntry{
			ActorID:    user.ID,
			Action:     repository.AuditActionFanficCommentDelete,
			TargetType: repository.AuditTargetFanficComment,
			TargetID:   cid.String(),
		},
	})

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, 500, 0, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 0)
	entries, auditTotal, err := repos.AuditLog.List(context.Background(), repository.AuditActionFanficCommentDelete, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, auditTotal)
	require.Len(t, entries, 1)
	assert.Equal(t, cid.String(), entries[0].TargetID)
}

func TestFanficDAO_DeleteCommentWithAudit_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	_, err = repos.Fanfic.DeleteCommentWithAudit(context.Background(), repository.FanficCommentDelete{
		ID:      cid,
		UserID:  admin.ID,
		AsAdmin: true,
		Audit: repository.NewAuditEntry{
			ActorID:    admin.ID,
			Action:     repository.AuditActionFanficCommentDeleteAdmin,
			TargetType: repository.AuditTargetFanficComment,
			TargetID:   cid.String(),
		},
	})

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, 500, 0, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 0)
	entries, auditTotal, err := repos.AuditLog.List(context.Background(), repository.AuditActionFanficCommentDeleteAdmin, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, auditTotal)
	require.Len(t, entries, 1)
	assert.Equal(t, admin.ID, entries[0].ActorID)
}

func TestFanficDAO_DeleteCommentWithAudit_NonOwnerWritesNoAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	paths, err := repos.Fanfic.DeleteCommentWithAudit(context.Background(), repository.FanficCommentDelete{
		ID:     cid,
		UserID: other.ID,
		Audit: repository.NewAuditEntry{
			ActorID:    other.ID,
			Action:     repository.AuditActionFanficCommentDelete,
			TargetType: repository.AuditTargetFanficComment,
			TargetID:   cid.String(),
		},
	})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, 500, 0, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 1)
	_, auditTotal, err := repos.AuditLog.List(context.Background(), repository.AuditActionFanficCommentDelete, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, auditTotal)
}

func TestFanficDAO_DeleteCommentWithAudit_ReturnsOnlyThatCommentsMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	target, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "target")
	require.NoError(t, err)
	sibling, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "sibling")
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID:    target.ID,
		MediaURL:     "/uploads/images/target.png",
		MediaType:    "image",
		ThumbnailURL: "/uploads/images/target_thumb.png",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: target.ID,
		MediaURL:  "/uploads/images/target_no_thumb.gif",
		MediaType: "image",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: sibling.ID,
		MediaURL:  "/uploads/images/sibling.png",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	paths, err := repos.Fanfic.DeleteCommentWithAudit(context.Background(), repository.FanficCommentDelete{
		ID:     target.ID,
		UserID: user.ID,
		Audit: repository.NewAuditEntry{
			ActorID:    user.ID,
			Action:     repository.AuditActionFanficCommentDelete,
			TargetType: repository.AuditTargetFanficComment,
			TargetID:   target.ID.String(),
		},
	})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/images/target.png",
		"/uploads/images/target_thumb.png",
		"/uploads/images/target_no_thumb.gif",
	}, paths)
	assert.NotContains(t, paths, "/uploads/images/sibling.png")
	assert.NotContains(t, paths, "")
	remaining, err := repos.Fanfic.GetCommentMedia(context.Background(), sibling.ID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
}

func TestFanficDAO_GetComments_ExcludesUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	_, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "ok")
	require.NoError(t, err)
	_, err = repos.Fanfic.CreateComment(context.Background(), fid, nil, blocked.ID, "blocked")
	require.NoError(t, err)

	// when
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, 500, 0, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "ok", cs[0].Body)
}

func TestFanficDAO_GetCommentEntityID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	got, err := repos.Fanfic.GetCommentEntityID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, fid, got)
}

func TestFanficDAO_GetCommentAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	got, err := repos.Fanfic.GetCommentAuthorID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestFanficDAO_LikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	err = repos.Fanfic.LikeComment(context.Background(), liker.ID, cid)

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, liker.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 1, cs[0].LikeCount)
	assert.True(t, cs[0].UserLiked)
}

func TestFanficDAO_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID
	require.NoError(t, repos.Fanfic.LikeComment(context.Background(), liker.ID, cid))

	// when
	err = repos.Fanfic.LikeComment(context.Background(), liker.ID, cid)

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, liker.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 1, cs[0].LikeCount)
}

func TestFanficDAO_UnlikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, owner.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID
	require.NoError(t, repos.Fanfic.LikeComment(context.Background(), liker.ID, cid))

	// when
	err = repos.Fanfic.UnlikeComment(context.Background(), liker.ID, cid)

	// then
	require.NoError(t, err)
	cs, _, err := repos.Fanfic.GetComments(context.Background(), fid, liker.ID, 500, 0, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 0, cs[0].LikeCount)
	assert.False(t, cs[0].UserLiked)
}

func TestFanficDAO_AddCommentMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID

	// when
	id, err := repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID:    cid,
		MediaURL:     "http://x/img.png",
		MediaType:    "image",
		ThumbnailURL: "http://x/t.png",
	})

	// then
	require.NoError(t, err)
	assert.NotZero(t, id)
	media, err := repos.Fanfic.GetCommentMedia(context.Background(), cid)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "http://x/img.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
}

func TestFanficDAO_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID
	id, err := repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: cid,
		MediaURL:  "http://old/img.png",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	err = repos.Fanfic.UpdateCommentMediaURL(context.Background(), id, "http://new/img.png")

	// then
	require.NoError(t, err)
	media, err := repos.Fanfic.GetCommentMedia(context.Background(), cid)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "http://new/img.png", media[0].MediaURL)
}

func TestFanficDAO_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID
	id, err := repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: cid,
		MediaURL:  "http://x/img.png",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	err = repos.Fanfic.UpdateCommentMediaThumbnail(context.Background(), id, "http://x/thumb.png")

	// then
	require.NoError(t, err)
	media, err := repos.Fanfic.GetCommentMedia(context.Background(), cid)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "http://x/thumb.png", media[0].ThumbnailURL)
}

func TestFanficDAO_GetCommentMedia_OrderedBySort(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cidRow, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "body")
	require.NoError(t, err)
	cid := cidRow.ID
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: cid,
		MediaURL:  "http://x/0.png",
		MediaType: "image",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: cid,
		MediaURL:  "http://x/1.png",
		MediaType: "image",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: cid,
		MediaURL:  "http://x/2.png",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	media, err := repos.Fanfic.GetCommentMedia(context.Background(), cid)

	// then
	require.NoError(t, err)
	require.Len(t, media, 3)
	assert.Equal(t, "http://x/0.png", media[0].MediaURL)
	assert.Equal(t, "http://x/1.png", media[1].MediaURL)
	assert.Equal(t, "http://x/2.png", media[2].MediaURL)
}

func TestFanficDAO_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	c1Row, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "a")
	require.NoError(t, err)
	c1 := c1Row.ID
	c2Row, err := repos.Fanfic.CreateComment(context.Background(), fid, nil, user.ID, "b")
	require.NoError(t, err)
	c2 := c2Row.ID
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: c1,
		MediaURL:  "http://x/1.png",
		MediaType: "image",
	})
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), repository.NewFanficCommentMedia{
		CommentID: c2,
		MediaURL:  "http://x/2.png",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	got, err := repos.Fanfic.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[c1], 1)
	assert.Len(t, got[c2], 1)
}

func TestFanficDAO_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}
