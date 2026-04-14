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

func makeChars() []dto.ShipCharacter {
	return []dto.ShipCharacter{
		{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
		{Series: "umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
	}
}

func createShip(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string, chars []dto.ShipCharacter) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Ship.CreateWithCharacters(context.Background(), id, userID, title, "desc", chars))
	return id
}

func TestShipRepository_CreateWithCharacters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()
	chars := makeChars()

	// when
	err := repos.Ship.CreateWithCharacters(context.Background(), id, user.ID, "Ship A", "About them", chars)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Ship A", row.Title)
	assert.Equal(t, "About them", row.Description)
	assert.Equal(t, user.ID, row.UserID)
	got, err := repos.Ship.GetCharacters(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestShipRepository_CreateWithCharacters_TrimsName(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()
	chars := []dto.ShipCharacter{{Series: "u", CharacterID: "x", CharacterName: "  Padded  "}}

	// when
	err := repos.Ship.CreateWithCharacters(context.Background(), id, user.ID, "T", "", chars)

	// then
	require.NoError(t, err)
	got, err := repos.Ship.GetCharacters(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Padded", got[0].CharacterName)
}

func TestShipRepository_UpdateWithCharacters_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "Old", makeChars())
	newChars := []dto.ShipCharacter{{Series: "u", CharacterID: "c", CharacterName: "Solo"}}

	// when
	err := repos.Ship.UpdateWithCharacters(context.Background(), id, user.ID, "New", "ND", newChars, false)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "New", row.Title)
	assert.Equal(t, "ND", row.Description)
	chars, err := repos.Ship.GetCharacters(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, chars, 1)
	assert.Equal(t, "Solo", chars[0].CharacterName)
}

func TestShipRepository_UpdateWithCharacters_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.UpdateWithCharacters(context.Background(), id, stranger.ID, "Hijacked", "", makeChars(), false)

	// then
	require.Error(t, err)
}

func TestShipRepository_UpdateWithCharacters_AsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	admin := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.UpdateWithCharacters(context.Background(), id, admin.ID, "Modded", "", makeChars(), true)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "Modded", row.Title)
}

func TestShipRepository_UpdateImage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	err := repos.Ship.UpdateImage(context.Background(), id, "/img.png", "/thumb.png")

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "/img.png", row.ImageURL)
	assert.Equal(t, "/thumb.png", row.ThumbnailURL)
}

func TestShipRepository_Delete_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	err := repos.Ship.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipRepository_Delete_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.Delete(context.Background(), id, stranger.ID)

	// then
	require.Error(t, err)
}

func TestShipRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)

	// when
	row, err := repos.Ship.GetByID(context.Background(), uuid.New(), viewer.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipRepository_GetByID_PopulatesAuthor(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos, repotest.WithUsername("captain_ship"), repotest.WithDisplayName("Captain"))
	viewer := repotest.CreateUser(t, repos)
	id := createShip(t, repos, author.ID, "T", makeChars())

	// when
	row, err := repos.Ship.GetByID(context.Background(), id, viewer.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "captain_ship", row.AuthorUsername)
	assert.Equal(t, "Captain", row.AuthorDisplayName)
}

func TestShipRepository_GetAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	got, err := repos.Ship.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestShipRepository_GetAuthorID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Ship.GetAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestShipRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)

	// when
	rows, total, err := repos.Ship.List(context.Background(), viewer.ID, "", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestShipRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 5; i++ {
		createShip(t, repos, user.ID, "T", makeChars())
	}

	// when
	page1, total, err := repos.Ship.List(context.Background(), user.ID, "", false, "", "", 2, 0, nil)
	page2, _, err2 := repos.Ship.List(context.Background(), user.ID, "", false, "", "", 2, 2, nil)

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestShipRepository_List_FilterBySeries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createShip(t, repos, user.ID, "Umi", []dto.ShipCharacter{{Series: "umineko", CharacterID: "a", CharacterName: "A"}})
	createShip(t, repos, user.ID, "Hig", []dto.ShipCharacter{{Series: "higurashi", CharacterID: "b", CharacterName: "B"}})

	// when
	rows, total, err := repos.Ship.List(context.Background(), user.ID, "", false, "umineko", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Umi", rows[0].Title)
}

func TestShipRepository_List_FilterByCharacterID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createShip(t, repos, user.ID, "WithBattler", []dto.ShipCharacter{{Series: "u", CharacterID: "battler", CharacterName: "Battler"}})
	createShip(t, repos, user.ID, "Other", []dto.ShipCharacter{{Series: "u", CharacterID: "other", CharacterName: "Other"}})

	// when
	rows, total, err := repos.Ship.List(context.Background(), user.ID, "", false, "", "battler", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "WithBattler", rows[0].Title)
}

func TestShipRepository_List_CrackshipsOnlyFilters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	crack := createShip(t, repos, owner.ID, "Crack", makeChars())
	popular := createShip(t, repos, owner.ID, "Popular", makeChars())
	for i := 0; i < 4; i++ {
		voter := repotest.CreateUser(t, repos)
		require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, crack, -1))
	}

	// when
	rows, total, err := repos.Ship.List(context.Background(), owner.ID, "", true, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, crack, rows[0].ID)
	_ = popular
}

func TestShipRepository_List_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createShip(t, repos, blocked.ID, "Hidden", makeChars())
	createShip(t, repos, other.ID, "Visible", makeChars())

	// when
	rows, total, err := repos.Ship.List(context.Background(), viewer.ID, "", false, "", "", 10, 0, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Visible", rows[0].Title)
}

func TestShipRepository_List_SortTop(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	low := createShip(t, repos, owner.ID, "Low", makeChars())
	high := createShip(t, repos, owner.ID, "High", makeChars())
	voter := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, high, 1))

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "top", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, high, rows[0].ID)
	assert.Equal(t, low, rows[1].ID)
}

func TestShipRepository_List_SortCrackship(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	a := createShip(t, repos, owner.ID, "A", makeChars())
	b := createShip(t, repos, owner.ID, "B", makeChars())
	voter := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, a, -1))

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "crackship", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, a, rows[0].ID)
	assert.Equal(t, b, rows[1].ID)
}

func TestShipRepository_List_SortControversial(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	plain := createShip(t, repos, owner.ID, "Plain", makeChars())
	controversial := createShip(t, repos, owner.ID, "Controversial", makeChars())
	up := repotest.CreateUser(t, repos)
	down := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), up.ID, controversial, 1))
	require.NoError(t, repos.Ship.Vote(context.Background(), down.ID, controversial, -1))

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "controversial", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, controversial, rows[0].ID)
	assert.Equal(t, plain, rows[1].ID)
}

func TestShipRepository_List_SortComments(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	silent := createShip(t, repos, owner.ID, "Silent", makeChars())
	chatty := createShip(t, repos, owner.ID, "Chatty", makeChars())
	require.NoError(t, repos.Ship.CreateComment(context.Background(), uuid.New(), chatty, nil, owner.ID, "hi"))

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "comments", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, chatty, rows[0].ID)
	assert.Equal(t, silent, rows[1].ID)
}

func TestShipRepository_List_SortOld(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	first := createShip(t, repos, owner.ID, "First", makeChars())
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE ships SET created_at = '2020-01-01 00:00:00' WHERE id = ?`, first)
	require.NoError(t, err)
	second := createShip(t, repos, owner.ID, "Second", makeChars())

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "old", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, first, rows[0].ID)
	assert.Equal(t, second, rows[1].ID)
}

func TestShipRepository_List_PopulatesViewerVote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())
	require.NoError(t, repos.Ship.Vote(context.Background(), viewer.ID, id, 1))

	// when
	rows, _, err := repos.Ship.List(context.Background(), viewer.ID, "", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].UserVote)
	assert.Equal(t, 1, rows[0].VoteScore)
}

func TestShipRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createShip(t, repos, user.ID, "Mine1", makeChars())
	createShip(t, repos, user.ID, "Mine2", makeChars())
	createShip(t, repos, other.ID, "Theirs", makeChars())

	// when
	rows, total, err := repos.Ship.ListByUser(context.Background(), user.ID, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
	for _, r := range rows {
		assert.Equal(t, user.ID, r.UserID)
	}
}

func TestShipRepository_ListByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createShip(t, repos, user.ID, "T", makeChars())
	}

	// when
	rows, total, err := repos.Ship.ListByUser(context.Background(), user.ID, user.ID, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestShipRepository_GetCharacters_Ordered(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", []dto.ShipCharacter{
		{Series: "u", CharacterID: "a", CharacterName: "A"},
		{Series: "u", CharacterID: "b", CharacterName: "B"},
		{Series: "u", CharacterID: "c", CharacterName: "C"},
	})

	// when
	got, err := repos.Ship.GetCharacters(context.Background(), id)

	// then
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "A", got[0].CharacterName)
	assert.Equal(t, 0, got[0].SortOrder)
	assert.Equal(t, "C", got[2].CharacterName)
	assert.Equal(t, 2, got[2].SortOrder)
}

func TestShipRepository_GetCharactersBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := createShip(t, repos, user.ID, "A", []dto.ShipCharacter{{Series: "u", CharacterID: "x", CharacterName: "X"}})
	b := createShip(t, repos, user.ID, "B", []dto.ShipCharacter{{Series: "u", CharacterID: "y", CharacterName: "Y"}, {Series: "u", CharacterID: "z", CharacterName: "Z"}})

	// when
	got, err := repos.Ship.GetCharactersBatch(context.Background(), []uuid.UUID{a, b})

	// then
	require.NoError(t, err)
	require.Len(t, got[a], 1)
	require.Len(t, got[b], 2)
}

func TestShipRepository_GetCharactersBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Ship.GetCharactersBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestShipRepository_Vote_Insert(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.Vote(context.Background(), voter.ID, id, 1)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.VoteScore)
	assert.Equal(t, 1, row.UserVote)
}

func TestShipRepository_Vote_Update(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())
	require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, id, 1))

	// when
	err := repos.Ship.Vote(context.Background(), voter.ID, id, -1)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, row.VoteScore)
	assert.Equal(t, -1, row.UserVote)
}

func TestShipRepository_Vote_Remove(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())
	require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, id, 1))

	// when
	err := repos.Ship.Vote(context.Background(), voter.ID, id, 0)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, voter.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, row.VoteScore)
	assert.Equal(t, 0, row.UserVote)
}

func TestShipRepository_Vote_Aggregates(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())
	for i := 0; i < 3; i++ {
		voter := repotest.CreateUser(t, repos)
		require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, id, 1))
	}
	downVoter := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), downVoter.ID, id, -1))

	// when
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, row.VoteScore)
}

func TestShipRepository_CreateComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()

	// when
	err := repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "hello")

	// then
	require.NoError(t, err)
	got, err := repos.Ship.GetCommentShipID(context.Background(), commentID)
	require.NoError(t, err)
	assert.Equal(t, shipID, got)
}

func TestShipRepository_CreateComment_WithParent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	parentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), parentID, shipID, nil, user.ID, "parent"))
	childID := uuid.New()

	// when
	err := repos.Ship.CreateComment(context.Background(), childID, shipID, &parentID, user.ID, "child")

	// then
	require.NoError(t, err)
	comments, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, comments, 2)
	var foundChild bool
	for _, c := range comments {
		if c.ID == childID {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parentID, *c.ParentID)
			foundChild = true
		}
	}
	assert.True(t, foundChild)
}

func TestShipRepository_UpdateComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "old"))

	// when
	err := repos.Ship.UpdateComment(context.Background(), commentID, user.ID, "new")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new", comments[0].Body)
}

func TestShipRepository_UpdateComment_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))

	// when
	err := repos.Ship.UpdateComment(context.Background(), commentID, stranger.ID, "hijack")

	// then
	require.Error(t, err)
}

func TestShipRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "old"))

	// when
	err := repos.Ship.UpdateCommentAsAdmin(context.Background(), commentID, "moderated")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "moderated", comments[0].Body)
}

func TestShipRepository_DeleteComment_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))

	// when
	err := repos.Ship.DeleteComment(context.Background(), commentID, user.ID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestShipRepository_DeleteComment_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))

	// when
	err := repos.Ship.DeleteComment(context.Background(), commentID, stranger.ID)

	// then
	require.Error(t, err)
}

func TestShipRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))

	// when
	err := repos.Ship.DeleteCommentAsAdmin(context.Background(), commentID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestShipRepository_GetComments_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	for i := 0; i < 3; i++ {
		require.NoError(t, repos.Ship.CreateComment(context.Background(), uuid.New(), shipID, nil, user.ID, "c"))
	}

	// when
	rows, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 2, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestShipRepository_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	require.NoError(t, repos.Ship.CreateComment(context.Background(), uuid.New(), shipID, nil, owner.ID, "ok"))
	require.NoError(t, repos.Ship.CreateComment(context.Background(), uuid.New(), shipID, nil, blocked.ID, "hidden"))

	// when
	rows, total, err := repos.Ship.GetComments(context.Background(), shipID, owner.ID, 10, 0, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "ok", rows[0].Body)
}

func TestShipRepository_GetCommentShipID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, err := repos.Ship.GetCommentShipID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestShipRepository_GetCommentAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))

	// when
	got, err := repos.Ship.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestShipRepository_LikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, owner.ID, "x"))

	// when
	err := repos.Ship.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	rows, _, err := repos.Ship.GetComments(context.Background(), shipID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].LikeCount)
	assert.True(t, rows[0].UserLiked)
}

func TestShipRepository_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, owner.ID, "x"))
	require.NoError(t, repos.Ship.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err := repos.Ship.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	rows, _, err := repos.Ship.GetComments(context.Background(), shipID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, rows[0].LikeCount)
}

func TestShipRepository_UnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, owner.ID, "x"))
	require.NoError(t, repos.Ship.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err := repos.Ship.UnlikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	rows, _, err := repos.Ship.GetComments(context.Background(), shipID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, rows[0].LikeCount)
	assert.False(t, rows[0].UserLiked)
}

func TestShipRepository_AddCommentMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))

	// when
	id, err := repos.Ship.AddCommentMedia(context.Background(), commentID, "/m.png", "image", "/t.png", 0)

	// then
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
	media, err := repos.Ship.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/m.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
	assert.Equal(t, "/t.png", media[0].ThumbnailURL)
}

func TestShipRepository_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))
	id, err := repos.Ship.AddCommentMedia(context.Background(), commentID, "/old.png", "image", "", 0)
	require.NoError(t, err)

	// when
	err = repos.Ship.UpdateCommentMediaURL(context.Background(), id, "/new.png")

	// then
	require.NoError(t, err)
	media, err := repos.Ship.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].MediaURL)
}

func TestShipRepository_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))
	id, err := repos.Ship.AddCommentMedia(context.Background(), commentID, "/m.png", "image", "/old.png", 0)
	require.NoError(t, err)

	// when
	err = repos.Ship.UpdateCommentMediaThumbnail(context.Background(), id, "/new.png")

	// then
	require.NoError(t, err)
	media, err := repos.Ship.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].ThumbnailURL)
}

func TestShipRepository_GetCommentMedia_Ordered(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	commentID := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), commentID, shipID, nil, user.ID, "x"))
	_, err := repos.Ship.AddCommentMedia(context.Background(), commentID, "/b.png", "image", "", 1)
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), commentID, "/a.png", "image", "", 0)
	require.NoError(t, err)

	// when
	media, err := repos.Ship.GetCommentMedia(context.Background(), commentID)

	// then
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "/b.png", media[1].MediaURL)
}

func TestShipRepository_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	c1 := uuid.New()
	c2 := uuid.New()
	require.NoError(t, repos.Ship.CreateComment(context.Background(), c1, shipID, nil, user.ID, "a"))
	require.NoError(t, repos.Ship.CreateComment(context.Background(), c2, shipID, nil, user.ID, "b"))
	_, err := repos.Ship.AddCommentMedia(context.Background(), c1, "/a.png", "image", "", 0)
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), c2, "/b1.png", "image", "", 0)
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), c2, "/b2.png", "image", "", 1)
	require.NoError(t, err)

	// when
	got, err := repos.Ship.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[c1], 1)
	require.Len(t, got[c2], 2)
	assert.Equal(t, "/b1.png", got[c2][0].MediaURL)
}

func TestShipRepository_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	got, err := repos.Ship.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}
