package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

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
	id := uuid.New()
	err := repos.Fanfic.CreateWithDetails(
		context.Background(), id, userID, title, "summary", "Umineko", "K", "English", "in_progress",
		false, false,
		[]string{"Drama", "Mystery"},
		[]string{"angst", "fluff"},
		makeFanficChars(),
		false,
	)
	require.NoError(t, err)
	return id
}

func createFanficChapter(t *testing.T, repos *repository.Repositories, fanficID uuid.UUID, chapterNumber int, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := repos.Fanfic.CreateChapter(context.Background(), id, fanficID, chapterNumber, title, "body text", 100)
	require.NoError(t, err)
	return id
}

func TestFanficRepository_CreateWithDetails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.Fanfic.CreateWithDetails(
		context.Background(), id, user.ID, "Title", "Summary", "Umineko", "T", "English", "in_progress",
		true, false,
		[]string{"Drama"},
		[]string{"sadtag"},
		makeFanficChars(),
		true,
	)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Title", row.Title)
	assert.Equal(t, "Summary", row.Summary)
	assert.True(t, row.IsOneshot)
	assert.False(t, row.ContainsLemons)
	assert.True(t, row.IsPairing)
}

func TestFanficRepository_CreateWithDetails_TrimsCharacterName(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()
	chars := []dto.FanficCharacter{{Series: "Umineko", CharacterID: "x", CharacterName: "  Padded  "}}

	// when
	err := repos.Fanfic.CreateWithDetails(
		context.Background(), id, user.ID, "T", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil, chars, false,
	)

	// then
	require.NoError(t, err)
	got, err := repos.Fanfic.GetCharacters(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Padded", got[0].CharacterName)
}

func TestFanficRepository_CreateWithDetails_SkipsEmptyTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.Fanfic.CreateWithDetails(
		context.Background(), id, user.ID, "T", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, []string{"  ", "", "keep"}, nil, false,
	)

	// then
	require.NoError(t, err)
	tags, err := repos.Fanfic.GetTags(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, []string{"keep"}, tags)
}

func TestFanficRepository_UpdateWithDetails_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Old")

	// when
	err := repos.Fanfic.UpdateWithDetails(
		context.Background(), id, user.ID, "New", "newsum", "Higurashi", "M", "Spanish", "completed",
		true, true, []string{"Angst"}, []string{"newtag"},
		[]dto.FanficCharacter{{Series: "Higurashi", CharacterID: "rena", CharacterName: "Rena"}},
		false, false,
	)

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

func TestFanficRepository_UpdateWithDetails_NonOwnerFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWithDetails(
		context.Background(), id, other.ID, "New", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil, nil, false, false,
	)

	// then
	require.Error(t, err)
}

func TestFanficRepository_UpdateWithDetails_AsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	admin := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWithDetails(
		context.Background(), id, admin.ID, "AdminEdit", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil, nil, false, true,
	)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "AdminEdit", row.Title)
}

func TestFanficRepository_UpdateWithDetails_ReplacesGenresTagsCharacters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWithDetails(
		context.Background(), id, user.ID, "Title", "", "Umineko", "K", "English", "in_progress",
		false, false,
		[]string{"Horror"},
		[]string{"replaced"},
		[]dto.FanficCharacter{{Series: "Umineko", CharacterID: "ange", CharacterName: "Ange"}},
		false, false,
	)

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

func TestFanficRepository_UpdateCoverImage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_UpdateWordCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")
	c1 := uuid.New()
	c2 := uuid.New()
	require.NoError(t, repos.Fanfic.CreateChapter(context.Background(), c1, id, 1, "c1", "body", 500))
	require.NoError(t, repos.Fanfic.CreateChapter(context.Background(), c2, id, 2, "c2", "body", 750))

	// when
	err := repos.Fanfic.UpdateWordCount(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1250, row.WordCount)
}

func TestFanficRepository_UpdateWordCount_NoChapters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.UpdateWordCount(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.WordCount)
}

func TestFanficRepository_Delete_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	err := repos.Fanfic.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficRepository_Delete_NonOwnerFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.Delete(context.Background(), id, other.ID)

	// then
	require.Error(t, err)
}

func TestFanficRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, owner.ID, "Title")

	// when
	err := repos.Fanfic.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Fanfic.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	row, err := repos.Fanfic.GetByID(context.Background(), uuid.New(), user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestFanficRepository_GetByID_IncludesAuthorDetails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author Name"))
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	row, err := repos.Fanfic.GetByID(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Author Name", row.AuthorDisplayName)
	assert.Equal(t, user.Username, row.AuthorUsername)
}

func TestFanficRepository_GetAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createFanfic(t, repos, user.ID, "Title")

	// when
	got, err := repos.Fanfic.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestFanficRepository_GetAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Fanfic.GetAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestFanficRepository_List_Defaults(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "A")
	createFanfic(t, repos, user.ID, "B")

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestFanficRepository_List_HidesDraftsFromOthers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), id, owner.ID, "Draft", "", "Umineko", "K", "English", "draft",
		false, false, nil, nil, nil, false,
	))

	// when
	_, totalOther, err := repos.Fanfic.List(context.Background(), other.ID, repository.FanficListParams{Limit: 10}, nil)
	require.NoError(t, err)
	_, totalOwner, err := repos.Fanfic.List(context.Background(), owner.ID, repository.FanficListParams{Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, totalOther)
	assert.Equal(t, 1, totalOwner)
}

func TestFanficRepository_List_FiltersLemons(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "Clean")
	spicy := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), spicy, user.ID, "Spicy", "", "Umineko", "M", "English", "in_progress",
		false, true, nil, nil, nil, false,
	))

	// when
	_, totalNoLemons, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Limit: 10}, nil)
	require.NoError(t, err)
	_, totalWithLemons, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Limit: 10, ShowLemons: true}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, totalNoLemons)
	assert.Equal(t, 2, totalWithLemons)
}

func TestFanficRepository_List_FilterSeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "Umi")
	id := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), id, user.ID, "Higu", "", "Higurashi", "K", "English", "in_progress",
		false, false, nil, nil, nil, false,
	))

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Series: "Higurashi", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Higurashi", rows[0].Series)
}

func TestFanficRepository_List_FilterRating(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "K one")
	id := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), id, user.ID, "M one", "", "Umineko", "M", "English", "in_progress",
		false, false, nil, nil, nil, false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Rating: "M", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_FilterLanguage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "English")
	id := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), id, user.ID, "Jap", "", "Umineko", "K", "Japanese", "in_progress",
		false, false, nil, nil, nil, false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Language: "Japanese", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_FilterStatus(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "WIP")
	done := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), done, user.ID, "Done", "", "Umineko", "K", "English", "completed",
		false, false, nil, nil, nil, false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Status: "completed", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_FilterGenres(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := uuid.New()
	b := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), a, user.ID, "A", "", "Umineko", "K", "English", "in_progress",
		false, false, []string{"Drama", "Mystery"}, nil, nil, false,
	))
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), b, user.ID, "B", "", "Umineko", "K", "English", "in_progress",
		false, false, []string{"Drama"}, nil, nil, false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{GenreA: "Drama", GenreB: "Mystery", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_FilterTag(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := uuid.New()
	b := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), a, user.ID, "A", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, []string{"fluff"}, nil, false,
	))
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), b, user.ID, "B", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, []string{"angst"}, nil, false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Tag: "angst", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_FilterCharacter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := uuid.New()
	b := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), a, user.ID, "A", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil,
		[]dto.FanficCharacter{{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"}},
		false,
	))
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), b, user.ID, "B", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil,
		[]dto.FanficCharacter{{Series: "Umineko", CharacterID: "rena", CharacterName: "Rena"}},
		false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{CharacterA: "Battler", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_FilterPairing(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	single := uuid.New()
	pair := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), single, user.ID, "Single", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil,
		[]dto.FanficCharacter{{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"}},
		false,
	))
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), pair, user.ID, "Pair", "", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil,
		[]dto.FanficCharacter{
			{Series: "Umineko", CharacterID: "battler", CharacterName: "Battler"},
			{Series: "Umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
		},
		true,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{CharacterA: "Battler", IsPairing: true, Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_List_Search(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := uuid.New()
	b := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), a, user.ID, "Golden Witch", "summary", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil, nil, false,
	))
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), b, user.ID, "Other", "golden text here", "Umineko", "K", "English", "in_progress",
		false, false, nil, nil, nil, false,
	))

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Search: "golden", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
}

func TestFanficRepository_List_SortFavourites(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	a := createFanfic(t, repos, user.ID, "A")
	b := createFanfic(t, repos, user.ID, "B")
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, b))

	// when
	rows, _, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Sort: "favourites", Limit: 10}, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, b, rows[0].ID)
	assert.Equal(t, a, rows[1].ID)
}

func TestFanficRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createFanfic(t, repos, user.ID, "X")
	}

	// when
	rows, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Limit: 2, Offset: 1}, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestFanficRepository_List_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	createFanfic(t, repos, user.ID, "Mine")
	createFanfic(t, repos, blocked.ID, "Blocked")

	// when
	_, total, err := repos.Fanfic.List(context.Background(), user.ID, repository.FanficListParams{Limit: 10}, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestFanficRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
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

func TestFanficRepository_ListByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createFanfic(t, repos, user.ID, "X")
	}

	// when
	rows, total, err := repos.Fanfic.ListByUser(context.Background(), user.ID, user.ID, 1, 1)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 1)
}

func TestFanficRepository_CreateChapter_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()

	// when
	err := repos.Fanfic.CreateChapter(context.Background(), cid, fid, 1, "Ch 1", "body", 10)

	// then
	require.NoError(t, err)
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 1)
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, cid, ch.ID)
	assert.Equal(t, "Ch 1", ch.Title)
	assert.Equal(t, 10, ch.WordCount)
}

func TestFanficRepository_GetChapter_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	ch, err := repos.Fanfic.GetChapter(context.Background(), fid, 99)

	// then
	require.NoError(t, err)
	assert.Nil(t, ch)
}

func TestFanficRepository_UpdateChapter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_DeleteChapter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_ListChapters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_GetChapterCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	createFanficChapter(t, repos, fid, 1, "A")
	createFanficChapter(t, repos, fid, 2, "B")

	// when
	n, err := repos.Fanfic.GetChapterCount(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestFanficRepository_GetNextChapterNumber(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_GetChapterFanficID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "A")

	// when
	got, err := repos.Fanfic.GetChapterFanficID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, fid, got)
}

func TestFanficRepository_GetChapterAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := createFanficChapter(t, repos, fid, 1, "A")

	// when
	got, err := repos.Fanfic.GetChapterAuthorID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestFanficRepository_GetGenres(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetGenres(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Drama", "Mystery"}, got)
}

func TestFanficRepository_GetGenresBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	f1 := createFanfic(t, repos, user.ID, "A")
	f2 := createFanfic(t, repos, user.ID, "B")

	// when
	got, err := repos.Fanfic.GetGenresBatch(context.Background(), []uuid.UUID{f1, f2})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Drama", "Mystery"}, got[f1])
	assert.ElementsMatch(t, []string{"Drama", "Mystery"}, got[f2])
}

func TestFanficRepository_GetGenresBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetGenresBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFanficRepository_GetTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetTags(context.Background(), fid)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"angst", "fluff"}, got)
}

func TestFanficRepository_GetTagsBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	f1 := createFanfic(t, repos, user.ID, "A")
	f2 := createFanfic(t, repos, user.ID, "B")

	// when
	got, err := repos.Fanfic.GetTagsBatch(context.Background(), []uuid.UUID{f1, f2})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"angst", "fluff"}, got[f1])
	assert.ElementsMatch(t, []string{"angst", "fluff"}, got[f2])
}

func TestFanficRepository_GetTagsBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetTagsBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFanficRepository_GetCharacters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetCharacters(context.Background(), fid)

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Battler", got[0].CharacterName)
	assert.Equal(t, "Beatrice", got[1].CharacterName)
}

func TestFanficRepository_GetCharactersBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	f1 := createFanfic(t, repos, user.ID, "A")
	f2 := createFanfic(t, repos, user.ID, "B")

	// when
	got, err := repos.Fanfic.GetCharactersBatch(context.Background(), []uuid.UUID{f1, f2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[f1], 2)
	assert.Len(t, got[f2], 2)
}

func TestFanficRepository_GetCharactersBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetCharactersBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFanficRepository_RegisterOCCharacter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Fanfic.RegisterOCCharacter(context.Background(), "My OC", user.ID)

	// then
	require.NoError(t, err)
	names, err := repos.Fanfic.SearchOCCharacters(context.Background(), "My")
	require.NoError(t, err)
	assert.Contains(t, names, "My OC")
}

func TestFanficRepository_RegisterOCCharacter_Duplicate(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Dup", user.ID))

	// when
	err := repos.Fanfic.RegisterOCCharacter(context.Background(), "Dup", user.ID)

	// then
	require.NoError(t, err)
	names, err := repos.Fanfic.SearchOCCharacters(context.Background(), "Dup")
	require.NoError(t, err)
	assert.Len(t, names, 1)
}

func TestFanficRepository_SearchOCCharacters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Alice", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Bob", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "Alicia", user.ID))

	// when
	got, err := repos.Fanfic.SearchOCCharacters(context.Background(), "Ali")

	// then
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestFanficRepository_SearchOCCharacters_EmptyQueryReturnsAll(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "A1", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "A2", user.ID))
	require.NoError(t, repos.Fanfic.RegisterOCCharacter(context.Background(), "A3", user.ID))

	// when
	got, err := repos.Fanfic.SearchOCCharacters(context.Background(), "")

	// then
	require.NoError(t, err)
	assert.Len(t, got, 3)
}

func TestFanficRepository_GetLanguages(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	langs, err := repos.Fanfic.GetLanguages(context.Background())

	// then
	require.NoError(t, err)
	assert.Contains(t, langs, "English")
	assert.Contains(t, langs, "Japanese")
}

func TestFanficRepository_RegisterLanguage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Fanfic.RegisterLanguage(context.Background(), "Klingon")

	// then
	require.NoError(t, err)
	langs, err := repos.Fanfic.GetLanguages(context.Background())
	require.NoError(t, err)
	assert.Contains(t, langs, "Klingon")
}

func TestFanficRepository_RegisterLanguage_Duplicate(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	require.NoError(t, repos.Fanfic.RegisterLanguage(context.Background(), "Welsh"))

	// when
	err := repos.Fanfic.RegisterLanguage(context.Background(), "Welsh")

	// then
	require.NoError(t, err)
}

func TestFanficRepository_GetSeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	series, err := repos.Fanfic.GetSeries(context.Background())

	// then
	require.NoError(t, err)
	assert.Contains(t, series, "Umineko")
	assert.Contains(t, series, "Higurashi")
}

func TestFanficRepository_RegisterSeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.Fanfic.RegisterSeries(context.Background(), "Rose Guns Days")

	// then
	require.NoError(t, err)
	series, err := repos.Fanfic.GetSeries(context.Background())
	require.NoError(t, err)
	assert.Contains(t, series, "Rose Guns Days")
}

func TestFanficRepository_Favourite(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
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

func TestFanficRepository_Favourite_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
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

func TestFanficRepository_Unfavourite(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
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

func TestFanficRepository_RecordView_New(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_RecordView_Duplicate(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_ReadingProgress_DefaultZero(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	got, err := repos.Fanfic.GetReadingProgress(context.Background(), user.ID, fid)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, got)
}

func TestFanficRepository_SetAndGetReadingProgress(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")

	// when
	err := repos.Fanfic.SetReadingProgress(context.Background(), user.ID, fid, 3)

	// then
	require.NoError(t, err)
	got, err := repos.Fanfic.GetReadingProgress(context.Background(), user.ID, fid)
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestFanficRepository_SetReadingProgress_Upsert(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
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

func TestFanficRepository_ListFavourites(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
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

func TestFanficRepository_ListFavourites_HidesDrafts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	draftID := uuid.New()
	require.NoError(t, repos.Fanfic.CreateWithDetails(
		context.Background(), draftID, owner.ID, "Draft", "", "Umineko", "K", "English", "draft",
		false, false, nil, nil, nil, false,
	))
	require.NoError(t, repos.Fanfic.Favourite(context.Background(), voter.ID, draftID))

	// when
	rows, _, err := repos.Fanfic.ListFavourites(context.Background(), voter.ID, voter.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Len(t, rows, 0)
}

func TestFanficRepository_CreateComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()

	// when
	err := repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "Nice!")

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "Nice!", cs[0].Body)
}

func TestFanficRepository_CreateComment_Threaded(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	parentID := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), parentID, fid, nil, user.ID, "parent"))
	childID := uuid.New()

	// when
	err := repos.Fanfic.CreateComment(context.Background(), childID, fid, &parentID, user.ID, "child")

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 2)
	var foundChild bool
	for _, c := range cs {
		if c.ID == childID {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parentID, *c.ParentID)
			foundChild = true
		}
	}
	assert.True(t, foundChild)
}

func TestFanficRepository_UpdateComment_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "old"))

	// when
	err := repos.Fanfic.UpdateComment(context.Background(), cid, user.ID, "new")

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "new", cs[0].Body)
}

func TestFanficRepository_UpdateComment_NonOwnerFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "old"))

	// when
	err := repos.Fanfic.UpdateComment(context.Background(), cid, other.ID, "hack")

	// then
	require.Error(t, err)
}

func TestFanficRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "old"))

	// when
	err := repos.Fanfic.UpdateCommentAsAdmin(context.Background(), cid, "admin edit")

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "admin edit", cs[0].Body)
}

func TestFanficRepository_DeleteComment_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))

	// when
	err := repos.Fanfic.DeleteComment(context.Background(), cid, user.ID)

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, user.ID, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 0)
}

func TestFanficRepository_DeleteComment_NonOwnerFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "body"))

	// when
	err := repos.Fanfic.DeleteComment(context.Background(), cid, other.ID)

	// then
	require.Error(t, err)
}

func TestFanficRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "body"))

	// when
	err := repos.Fanfic.DeleteCommentAsAdmin(context.Background(), cid)

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, nil)
	require.NoError(t, err)
	assert.Len(t, cs, 0)
}

func TestFanficRepository_GetComments_ExcludesUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), uuid.New(), fid, nil, owner.ID, "ok"))
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), uuid.New(), fid, nil, blocked.ID, "blocked"))

	// when
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, owner.ID, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, "ok", cs[0].Body)
}

func TestFanficRepository_GetCommentFanficID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))

	// when
	got, err := repos.Fanfic.GetCommentFanficID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, fid, got)
}

func TestFanficRepository_GetCommentAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))

	// when
	got, err := repos.Fanfic.GetCommentAuthorID(context.Background(), cid)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestFanficRepository_LikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "body"))

	// when
	err := repos.Fanfic.LikeComment(context.Background(), liker.ID, cid)

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 1, cs[0].LikeCount)
	assert.True(t, cs[0].UserLiked)
}

func TestFanficRepository_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "body"))
	require.NoError(t, repos.Fanfic.LikeComment(context.Background(), liker.ID, cid))

	// when
	err := repos.Fanfic.LikeComment(context.Background(), liker.ID, cid)

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 1, cs[0].LikeCount)
}

func TestFanficRepository_UnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, owner.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, owner.ID, "body"))
	require.NoError(t, repos.Fanfic.LikeComment(context.Background(), liker.ID, cid))

	// when
	err := repos.Fanfic.UnlikeComment(context.Background(), liker.ID, cid)

	// then
	require.NoError(t, err)
	cs, err := repos.Fanfic.GetComments(context.Background(), fid, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 0, cs[0].LikeCount)
	assert.False(t, cs[0].UserLiked)
}

func TestFanficRepository_AddCommentMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))

	// when
	id, err := repos.Fanfic.AddCommentMedia(context.Background(), cid, "http://x/img.png", "image", "http://x/t.png", 0)

	// then
	require.NoError(t, err)
	assert.NotZero(t, id)
	media, err := repos.Fanfic.GetCommentMedia(context.Background(), cid)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "http://x/img.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
}

func TestFanficRepository_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))
	id, err := repos.Fanfic.AddCommentMedia(context.Background(), cid, "http://old/img.png", "image", "", 0)
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

func TestFanficRepository_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))
	id, err := repos.Fanfic.AddCommentMedia(context.Background(), cid, "http://x/img.png", "image", "", 0)
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

func TestFanficRepository_GetCommentMedia_OrderedBySort(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	cid := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), cid, fid, nil, user.ID, "body"))
	_, err := repos.Fanfic.AddCommentMedia(context.Background(), cid, "http://x/2.png", "image", "", 2)
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), cid, "http://x/0.png", "image", "", 0)
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), cid, "http://x/1.png", "image", "", 1)
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

func TestFanficRepository_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	fid := createFanfic(t, repos, user.ID, "T")
	c1 := uuid.New()
	c2 := uuid.New()
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), c1, fid, nil, user.ID, "a"))
	require.NoError(t, repos.Fanfic.CreateComment(context.Background(), c2, fid, nil, user.ID, "b"))
	_, err := repos.Fanfic.AddCommentMedia(context.Background(), c1, "http://x/1.png", "image", "", 0)
	require.NoError(t, err)
	_, err = repos.Fanfic.AddCommentMedia(context.Background(), c2, "http://x/2.png", "image", "", 0)
	require.NoError(t, err)

	// when
	got, err := repos.Fanfic.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[c1], 1)
	assert.Len(t, got[c2], 1)
}

func TestFanficRepository_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Fanfic.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}
