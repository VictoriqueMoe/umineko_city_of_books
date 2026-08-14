package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

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
	created, err := repos.Ship.CreateWithCharacters(context.Background(), userID, title, "desc", chars)
	require.NoError(t, err)
	return created.ID
}

func TestShipDAO_CreateWithCharacters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	chars := makeChars()

	// when
	created, err := repos.Ship.CreateWithCharacters(context.Background(), user.ID, "Ship A", "About them", chars)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), created.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Ship A", row.Title)
	assert.Equal(t, "About them", row.Description)
	assert.Equal(t, user.ID, row.UserID)
	got, err := repos.Ship.GetCharacters(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestShipDAO_CreateWithCharacters_TrimsName(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	chars := []dto.ShipCharacter{{Series: "u", CharacterID: "x", CharacterName: "  Padded  "}}

	// when
	created, err := repos.Ship.CreateWithCharacters(context.Background(), user.ID, "T", "", chars)

	// then
	require.NoError(t, err)
	got, err := repos.Ship.GetCharacters(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Padded", got[0].CharacterName)
}

func TestShipDAO_UpdateWithCharacters_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "Old", makeChars())
	newChars := []dto.ShipCharacter{{Series: "u", CharacterID: "c", CharacterName: "Solo"}}

	// when
	err := repos.Ship.UpdateWithCharacters(context.Background(), repository.ShipUpdate{
		ID:          id,
		UserID:      user.ID,
		Title:       "New",
		Description: "ND",
		Characters:  newChars,
	})

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

func TestShipDAO_UpdateWithCharacters_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.UpdateWithCharacters(context.Background(), repository.ShipUpdate{
		ID:         id,
		UserID:     stranger.ID,
		Title:      "Hijacked",
		Characters: makeChars(),
	})

	// then
	require.Error(t, err)
}

func TestShipDAO_UpdateWithCharacters_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	admin := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.UpdateWithCharacters(context.Background(), repository.ShipUpdate{
		ID:         id,
		UserID:     admin.ID,
		Title:      "Modded",
		AsAdmin:    true,
		Characters: makeChars(),
	})

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "Modded", row.Title)
}

func TestShipDAO_UpdateImage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestShipDAO_Delete_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	err := repos.Ship.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipDAO_Delete_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.Delete(context.Background(), id, stranger.ID)

	// then
	require.Error(t, err)
}

func TestShipDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	err := repos.Ship.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipDAO_DeleteShip_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	_, err := repos.Ship.DeleteShip(context.Background(), repository.ShipDeletion{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipDAO_DeleteShip_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	_, err := repos.Ship.DeleteShip(context.Background(), repository.ShipDeletion{ID: id, UserID: stranger.ID})

	// then
	require.Error(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.NotNil(t, row)
}

func TestShipDAO_DeleteShip_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())

	// when
	_, err := repos.Ship.DeleteShip(context.Background(), repository.ShipDeletion{ID: id, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipDAO_DeleteShip_ReturnsImageAndCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())
	require.NoError(t, repos.Ship.UpdateImage(context.Background(), id, "/uploads/ships/cover.png", "/uploads/ships/cover_thumb.png"))
	comment, err := repos.Ship.CreateComment(context.Background(), id, nil, user.ID, "c")
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{
		CommentID:    comment.ID,
		MediaURL:     "/uploads/ships/comment.png",
		MediaType:    "image",
		ThumbnailURL: "/uploads/ships/comment_thumb.png",
	})
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{
		CommentID: comment.ID,
		MediaURL:  "/uploads/ships/comment_two.gif",
		MediaType: "image",
		SortOrder: 1,
	})
	require.NoError(t, err)

	// when
	paths, err := repos.Ship.DeleteShip(context.Background(), repository.ShipDeletion{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/ships/cover.png",
		"/uploads/ships/cover_thumb.png",
		"/uploads/ships/comment.png",
		"/uploads/ships/comment_thumb.png",
		"/uploads/ships/comment_two.gif",
	}, paths)
	row, err := repos.Ship.GetByID(context.Background(), id, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipDAO_DeleteShip_AsAdmin_ReturnsCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), id, nil, owner.ID, "c")
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{
		CommentID:    comment.ID,
		MediaURL:     "/uploads/ships/mod.png",
		MediaType:    "image",
		ThumbnailURL: "/uploads/ships/mod_thumb.png",
	})
	require.NoError(t, err)

	// when
	paths, err := repos.Ship.DeleteShip(context.Background(), repository.ShipDeletion{ID: id, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/ships/mod.png", "/uploads/ships/mod_thumb.png"}, paths)
}

func TestShipDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	viewer := daotest.CreateUser(t, repos)

	// when
	row, err := repos.Ship.GetByID(context.Background(), uuid.New(), viewer.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestShipDAO_GetByID_PopulatesAuthor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	author := daotest.CreateUser(t, repos, daotest.WithUsername("captain_ship"), daotest.WithDisplayName("Captain"))
	viewer := daotest.CreateUser(t, repos)
	id := createShip(t, repos, author.ID, "T", makeChars())

	// when
	row, err := repos.Ship.GetByID(context.Background(), id, viewer.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "captain_ship", row.AuthorUsername)
	assert.Equal(t, "Captain", row.AuthorDisplayName)
}

func TestShipDAO_GetAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createShip(t, repos, user.ID, "T", makeChars())

	// when
	got, err := repos.Ship.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestShipDAO_GetAuthorID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Ship.GetAuthorID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestShipDAO_List_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	viewer := daotest.CreateUser(t, repos)

	// when
	rows, total, err := repos.Ship.List(context.Background(), viewer.ID, "", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestShipDAO_List_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 5 {
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

func TestShipDAO_List_FilterBySeries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestShipDAO_List_FilterByCharacterID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestShipDAO_List_CrackshipsOnlyFilters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	crack := createShip(t, repos, owner.ID, "Crack", makeChars())
	popular := createShip(t, repos, owner.ID, "Popular", makeChars())
	for range 4 {
		voter := daotest.CreateUser(t, repos)
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

func TestShipDAO_List_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	viewer := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
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

func TestShipDAO_List_SortTop(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	low := createShip(t, repos, owner.ID, "Low", makeChars())
	high := createShip(t, repos, owner.ID, "High", makeChars())
	voter := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, high, 1))

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "top", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, high, rows[0].ID)
	assert.Equal(t, low, rows[1].ID)
}

func TestShipDAO_List_SortCrackship(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	a := createShip(t, repos, owner.ID, "A", makeChars())
	b := createShip(t, repos, owner.ID, "B", makeChars())
	voter := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, a, -1))

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "crackship", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, a, rows[0].ID)
	assert.Equal(t, b, rows[1].ID)
}

func TestShipDAO_List_SortControversial(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	plain := createShip(t, repos, owner.ID, "Plain", makeChars())
	controversial := createShip(t, repos, owner.ID, "Controversial", makeChars())
	up := daotest.CreateUser(t, repos)
	down := daotest.CreateUser(t, repos)
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

func TestShipDAO_List_SortComments(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	silent := createShip(t, repos, owner.ID, "Silent", makeChars())
	chatty := createShip(t, repos, owner.ID, "Chatty", makeChars())
	_, err := repos.Ship.CreateComment(context.Background(), chatty, nil, owner.ID, "hi")
	require.NoError(t, err)

	// when
	rows, _, err := repos.Ship.List(context.Background(), owner.ID, "comments", false, "", "", 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, chatty, rows[0].ID)
	assert.Equal(t, silent, rows[1].ID)
}

func TestShipDAO_List_SortOld(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	first := createShip(t, repos, owner.ID, "First", makeChars())
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE ships SET created_at = '2020-01-01 00:00:00' WHERE id = $1`, first)
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

func TestShipDAO_List_PopulatesViewerVote(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
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

func TestShipDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
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

func TestShipDAO_ListByUser_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createShip(t, repos, user.ID, "T", makeChars())
	}

	// when
	rows, total, err := repos.Ship.ListByUser(context.Background(), user.ID, user.ID, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestShipDAO_GetCharacters_Ordered(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
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

func TestShipDAO_GetCharactersBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	a := createShip(t, repos, user.ID, "A", []dto.ShipCharacter{{Series: "u", CharacterID: "x", CharacterName: "X"}})
	b := createShip(t, repos, user.ID, "B", []dto.ShipCharacter{{Series: "u", CharacterID: "y", CharacterName: "Y"}, {Series: "u", CharacterID: "z", CharacterName: "Z"}})

	// when
	got, err := repos.Ship.GetCharactersBatch(context.Background(), []uuid.UUID{a, b})

	// then
	require.NoError(t, err)
	require.Len(t, got[a], 1)
	require.Len(t, got[b], 2)
}

func TestShipDAO_GetCharactersBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Ship.GetCharactersBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestShipDAO_Vote_Insert(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
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

func TestShipDAO_Vote_Update(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
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

func TestShipDAO_Vote_Remove(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
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

func TestShipDAO_Vote_Aggregates(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	id := createShip(t, repos, owner.ID, "T", makeChars())
	for range 3 {
		voter := daotest.CreateUser(t, repos)
		require.NoError(t, repos.Ship.Vote(context.Background(), voter.ID, id, 1))
	}
	downVoter := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Ship.Vote(context.Background(), downVoter.ID, id, -1))

	// when
	row, err := repos.Ship.GetByID(context.Background(), id, owner.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, row.VoteScore)
}

func TestShipDAO_CreateComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())

	// when
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "hello")

	// then
	require.NoError(t, err)
	got, err := repos.Ship.GetCommentEntityID(context.Background(), comment.ID)
	require.NoError(t, err)
	assert.Equal(t, shipID, got)
}

func TestShipDAO_CreateComment_WithParent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	parent, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "parent")
	require.NoError(t, err)
	parentID := parent.ID

	// when
	child, err := repos.Ship.CreateComment(context.Background(), shipID, &parentID, user.ID, "child")

	// then
	require.NoError(t, err)
	comments, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, comments, 2)
	var foundChild bool
	for _, c := range comments {
		if c.ID == child.ID {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parentID, *c.ParentID)
			foundChild = true
		}
	}
	assert.True(t, foundChild)
}

func TestShipDAO_UpdateComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "old")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.UpdateComment(context.Background(), commentID, user.ID, "new")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new", comments[0].Body)
}

func TestShipDAO_UpdateComment_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.UpdateComment(context.Background(), commentID, stranger.ID, "hijack")

	// then
	require.Error(t, err)
}

func TestShipDAO_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "old")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.UpdateCommentAsAdmin(context.Background(), commentID, "moderated")

	// then
	require.NoError(t, err)
	comments, _, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "moderated", comments[0].Body)
}

func TestShipDAO_DeleteComment_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.DeleteComment(context.Background(), commentID, user.ID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestShipDAO_DeleteComment_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.DeleteComment(context.Background(), commentID, stranger.ID)

	// then
	require.Error(t, err)
}

func TestShipDAO_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.DeleteCommentAsAdmin(context.Background(), commentID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestShipDAO_UpdateCommentBody_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "old")
	require.NoError(t, err)

	// when
	err = repos.Ship.UpdateCommentBody(context.Background(), repository.ShipCommentUpdate{CommentID: comment.ID, UserID: moderator.ID, Body: "moderated", AsAdmin: true})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "moderated", comments[0].Body)
}

func TestShipDAO_UpdateCommentBody_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "old")
	require.NoError(t, err)

	// when
	err = repos.Ship.UpdateCommentBody(context.Background(), repository.ShipCommentUpdate{CommentID: comment.ID, UserID: stranger.ID, Body: "hijack"})

	// then
	require.Error(t, err)
}

func TestShipDAO_DeleteCommentWithAudit_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)

	// when
	_, err = repos.Ship.DeleteCommentWithAudit(context.Background(), repository.ShipCommentDeletion{CommentID: comment.ID, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	_, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	entries, _, err := repos.AuditLog.List(context.Background(), "ship_comment_delete_admin", 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, moderator.ID, entries[0].ActorID)
	assert.Equal(t, "ship_comment", entries[0].TargetType)
	assert.Equal(t, comment.ID.String(), entries[0].TargetID)
}

func TestShipDAO_DeleteCommentWithAudit_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)

	// when
	_, err = repos.Ship.DeleteCommentWithAudit(context.Background(), repository.ShipCommentDeletion{CommentID: comment.ID, UserID: user.ID})

	// then
	require.NoError(t, err)
	entries, _, err := repos.AuditLog.List(context.Background(), "ship_comment_delete", 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, user.ID, entries[0].ActorID)
}

func TestShipDAO_DeleteCommentWithAudit_NotOwnedWritesNoAudit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)

	// when
	_, err = repos.Ship.DeleteCommentWithAudit(context.Background(), repository.ShipCommentDeletion{CommentID: comment.ID, UserID: stranger.ID})

	// then
	require.Error(t, err)
	_, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	entries, _, err := repos.AuditLog.List(context.Background(), "ship_comment_delete", 10, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestShipDAO_DeleteCommentWithAudit_ReturnsOnlyThatCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	target, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "target")
	require.NoError(t, err)
	other, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "other")
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{
		CommentID:    target.ID,
		MediaURL:     "/uploads/ships/target.png",
		MediaType:    "image",
		ThumbnailURL: "/uploads/ships/target_thumb.png",
	})
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{
		CommentID: target.ID,
		MediaURL:  "/uploads/ships/target_two.gif",
		MediaType: "image",
		SortOrder: 1,
	})
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{
		CommentID: other.ID,
		MediaURL:  "/uploads/ships/other.png",
		MediaType: "image",
	})
	require.NoError(t, err)

	// when
	paths, err := repos.Ship.DeleteCommentWithAudit(context.Background(), repository.ShipCommentDeletion{CommentID: target.ID, UserID: user.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/ships/target.png",
		"/uploads/ships/target_thumb.png",
		"/uploads/ships/target_two.gif",
	}, paths)
	remaining, err := repos.Ship.GetCommentMedia(context.Background(), other.ID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, "/uploads/ships/other.png", remaining[0].MediaURL)
}

func TestShipDAO_GetComments_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	for range 3 {
		_, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "c")
		require.NoError(t, err)
	}

	// when
	rows, total, err := repos.Ship.GetComments(context.Background(), shipID, user.ID, 2, 0, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 2)
}

func TestShipDAO_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	_, err := repos.Ship.CreateComment(context.Background(), shipID, nil, owner.ID, "ok")
	require.NoError(t, err)
	_, err = repos.Ship.CreateComment(context.Background(), shipID, nil, blocked.ID, "hidden")
	require.NoError(t, err)

	// when
	rows, total, err := repos.Ship.GetComments(context.Background(), shipID, owner.ID, 10, 0, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "ok", rows[0].Body)
}

func TestShipDAO_GetCommentEntityID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	_, err := repos.Ship.GetCommentEntityID(context.Background(), uuid.New())

	// then
	require.Error(t, err)
}

func TestShipDAO_GetCommentAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	got, err := repos.Ship.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, got)
}

func TestShipDAO_LikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, owner.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	err = repos.Ship.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	rows, _, err := repos.Ship.GetComments(context.Background(), shipID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].LikeCount)
	assert.True(t, rows[0].UserLiked)
}

func TestShipDAO_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, owner.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID
	require.NoError(t, repos.Ship.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err = repos.Ship.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	rows, _, err := repos.Ship.GetComments(context.Background(), shipID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, rows[0].LikeCount)
}

func TestShipDAO_UnlikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, owner.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, owner.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID
	require.NoError(t, repos.Ship.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err = repos.Ship.UnlikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	rows, _, err := repos.Ship.GetComments(context.Background(), shipID, liker.ID, 10, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, rows[0].LikeCount)
	assert.False(t, rows[0].UserLiked)
}

func TestShipDAO_AddCommentMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID

	// when
	id, err := repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: commentID, MediaURL: "/m.png", MediaType: "image", ThumbnailURL: "/t.png"})

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

func TestShipDAO_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID
	id, err := repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: commentID, MediaURL: "/old.png", MediaType: "image"})
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

func TestShipDAO_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID
	id, err := repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: commentID, MediaURL: "/m.png", MediaType: "image", ThumbnailURL: "/old.png"})
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

func TestShipDAO_GetCommentMedia_Ordered(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	comment, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "x")
	require.NoError(t, err)
	commentID := comment.ID
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: commentID, MediaURL: "/a.png", MediaType: "image"})
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: commentID, MediaURL: "/b.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	media, err := repos.Ship.GetCommentMedia(context.Background(), commentID)

	// then
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "/b.png", media[1].MediaURL)
}

func TestShipDAO_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	shipID := createShip(t, repos, user.ID, "T", makeChars())
	c1Row, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "a")
	require.NoError(t, err)
	c1 := c1Row.ID
	c2Row, err := repos.Ship.CreateComment(context.Background(), shipID, nil, user.ID, "b")
	require.NoError(t, err)
	c2 := c2Row.ID
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: c1, MediaURL: "/a.png", MediaType: "image"})
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: c2, MediaURL: "/b1.png", MediaType: "image"})
	require.NoError(t, err)
	_, err = repos.Ship.AddCommentMedia(context.Background(), repository.NewShipCommentMedia{CommentID: c2, MediaURL: "/b2.png", MediaType: "image", SortOrder: 1})
	require.NoError(t, err)

	// when
	got, err := repos.Ship.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	assert.Len(t, got[c1], 1)
	require.Len(t, got[c2], 2)
	assert.Equal(t, "/b1.png", got[c2][0].MediaURL)
}

func TestShipDAO_GetCommentMediaBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.Ship.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}
