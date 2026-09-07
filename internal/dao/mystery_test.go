package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMystery(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string, difficulty string, freeForAll bool) uuid.UUID {
	t.Helper()
	created, err := repos.Mystery.Create(context.Background(), spec.NewMystery{
		UserID:             userID,
		Title:              title,
		Body:               "body",
		Difficulty:         difficulty,
		FreeForAll:         freeForAll,
		KeepOpenAfterSolve: false,
		Knox:               dto.DefaultKnoxContract(),
	})
	require.NoError(t, err)
	return created.ID
}

func createAttempt(t *testing.T, repos *repository.Repositories, mysteryID, userID uuid.UUID, parent *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Mystery.CreateAttempt(context.Background(), spec.NewMysteryAttempt{
		MysteryID: mysteryID,
		UserID:    userID,
		ParentID:  parent,
		Body:      body,
	})
	require.NoError(t, err)
	return created.ID
}

func createMysteryComment(t *testing.T, repos *repository.Repositories, mysteryID uuid.UUID, parent *uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()
	created, err := repos.Comments.ByID[string(mention.KindMysteryComment)].CreateComment(context.Background(), spec.NewComment[uuid.UUID]{
		TargetID: mysteryID,
		ParentID: parent,
		UserID:   userID,
		Body:     body,
	})
	require.NoError(t, err)
	return created.ID
}

func TestMysteryDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Mystery.Create(context.Background(), spec.NewMystery{
		UserID:             user.ID,
		Title:              "The Murder",
		Body:               "Who did it?",
		Difficulty:         "hard",
		FreeForAll:         false,
		KeepOpenAfterSolve: false,
		Knox:               dto.DefaultKnoxContract(),
	})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "The Murder", row.Title)
	assert.Equal(t, "Who did it?", row.Body)
	assert.Equal(t, "hard", row.Difficulty)
	assert.False(t, row.FreeForAll)
	assert.False(t, row.Solved)
}

func TestMysteryDAO_Create_FreeForAll(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Mystery.Create(context.Background(), spec.NewMystery{
		UserID:             user.ID,
		Title:              "FFA",
		Body:               "body",
		Difficulty:         "medium",
		FreeForAll:         true,
		KeepOpenAfterSolve: false,
		Knox:               dto.DefaultKnoxContract(),
	})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.FreeForAll)
}

func TestMysteryDAO_GetByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	row, err := repos.Mystery.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryDAO_GetByID_PopulatesAuthor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Author Name"))
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	row, err := repos.Mystery.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, user.Username, row.AuthorUsername)
	assert.Equal(t, "Author Name", row.AuthorDisplayName)
}

func TestMysteryDAO_Update_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "Old", "easy", false)

	// when
	err := repos.Mystery.Update(context.Background(), spec.MysteryOwnerUpdate{
		ID:         id,
		UserID:     user.ID,
		Title:      "New",
		Body:       "new body",
		Difficulty: "hard",
	})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "New", row.Title)
	assert.Equal(t, "new body", row.Body)
	assert.Equal(t, "hard", row.Difficulty)
}

func TestMysteryDAO_Update_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.Update(context.Background(), spec.MysteryOwnerUpdate{
		ID:         id,
		UserID:     stranger.ID,
		Title:      "X",
		Body:       "X",
		Difficulty: "easy",
	})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_UpdateAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.UpdateAsAdmin(context.Background(), spec.MysteryUpdate{
		ID:                 id,
		Title:              "Admin Title",
		Body:               "Admin Body",
		Difficulty:         "nightmare",
		FreeForAll:         true,
		KeepOpenAfterSolve: false,
		Knox:               dto.DefaultKnoxContract(),
	})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Admin Title", row.Title)
	assert.Equal(t, "nightmare", row.Difficulty)
	assert.True(t, row.FreeForAll)
}

func TestMysteryRepo_CreateWithClues(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	created, err := repos.Mystery.CreateWithClues(context.Background(), spec.NewMysteryWithClues{
		NewMystery: spec.NewMystery{
			UserID:     user.ID,
			Title:      "Locked Room",
			Body:       "How?",
			Difficulty: "hard",
			Knox:       dto.DefaultKnoxContract(),
		},
		Clues: []spec.NewClue{
			{Body: "first", TruthType: "red", SortOrder: 0},
			{Body: "second", TruthType: "blue", SortOrder: 1},
		},
	})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Locked Room", row.Title)
	clues, err := repos.Mystery.GetClues(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, clues, 2)
	assert.Equal(t, "first", clues[0].Body)
	assert.Equal(t, "second", clues[1].Body)
}

func TestMysteryRepo_UpdateWithClues_ReplacesPublicCluesOnly(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "old public", TruthType: "red", SortOrder: 0},
	})
	require.NoError(t, err)
	_, err = repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "private", TruthType: "red", SortOrder: 1, PlayerID: &player.ID},
	})
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateWithClues(context.Background(), spec.MysteryUpdateWithClues{
		MysteryUpdate: spec.MysteryUpdate{
			ID:         id,
			Title:      "New Title",
			Body:       "New Body",
			Difficulty: "nightmare",
			FreeForAll: true,
			Knox:       dto.DefaultKnoxContract(),
		},
		Clues: []spec.NewClue{{Body: "new public", TruthType: "blue", SortOrder: 0}},
	})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "New Title", row.Title)
	assert.Equal(t, "nightmare", row.Difficulty)
	assert.True(t, row.FreeForAll)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 2)
	assert.Equal(t, "new public", clues[0].Body)
	assert.Equal(t, "private", clues[1].Body)
}

func TestMysteryRepo_UpdateWithClues_ClueFailureRollsBackTheUpdate(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "Original", "easy", false)

	// when
	err := repos.Mystery.UpdateWithClues(context.Background(), spec.MysteryUpdateWithClues{
		MysteryUpdate: spec.MysteryUpdate{
			ID:         id,
			Title:      "New Title",
			Body:       "New Body",
			Difficulty: "nightmare",
			Knox:       dto.DefaultKnoxContract(),
		},
		Clues: []spec.NewClue{{Body: "doomed", TruthType: "red", SortOrder: -1}},
	})

	// then
	require.Error(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Original", row.Title)
	assert.Equal(t, "easy", row.Difficulty)
}

func TestMysteryDAO_Delete_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	err := repos.Mystery.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryDAO_Delete_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.Delete(context.Background(), spec.OwnedDeletion{ID: id, UserID: stranger.ID})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_DeleteAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	owner := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryDAO_GetAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	author, err := repos.Mystery.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, author)
}

func TestMysteryDAO_List_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	rows, total, err := repos.Mystery.List(context.Background(), spec.MysteryListFilter{Sort: "new", Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestMysteryDAO_List_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createMystery(t, repos, user.ID, "T", "easy", false)
	}

	// when
	page1, total1, err1 := repos.Mystery.List(context.Background(), spec.MysteryListFilter{Sort: "new", Limit: 2, Offset: 0})
	page2, total2, err2 := repos.Mystery.List(context.Background(), spec.MysteryListFilter{Sort: "new", Limit: 2, Offset: 2})

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 1)
	assert.Equal(t, 3, total1)
	assert.Equal(t, 3, total2)
}

func TestMysteryDAO_List_SortOld(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	a := createMystery(t, repos, user.ID, "first", "easy", false)
	b := createMystery(t, repos, user.ID, "second", "easy", false)

	// when
	rows, _, err := repos.Mystery.List(context.Background(), spec.MysteryListFilter{Sort: "old", Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	ids := []uuid.UUID{rows[0].ID, rows[1].ID}
	assert.ElementsMatch(t, []uuid.UUID{a, b}, ids)
}

func TestMysteryDAO_List_FilterSolved(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	solver := daotest.CreateUser(t, repos)
	solvedID := createMystery(t, repos, gm.ID, "solved", "easy", false)
	_ = createMystery(t, repos, gm.ID, "unsolved", "easy", false)
	attemptID := createAttempt(t, repos, solvedID, solver.ID, nil, "answer")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: solvedID, AttemptID: attemptID, LockMystery: true})) // when
	solved, _, errS := repos.Mystery.List(context.Background(), spec.MysteryListFilter{Sort: "new", Solved: new(true), Limit: 10, Offset: 0})
	unsolved, _, errU := repos.Mystery.List(context.Background(), spec.MysteryListFilter{Sort: "new", Solved: new(false), Limit: 10, Offset: 0})

	// then
	require.NoError(t, errS)
	require.NoError(t, errU)
	require.Len(t, solved, 1)
	require.Len(t, unsolved, 1)
	assert.Equal(t, solvedID, solved[0].ID)
}

func TestMysteryDAO_List_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	createMystery(t, repos, userA.ID, "A", "easy", false)
	idB := createMystery(t, repos, userB.ID, "B", "easy", false)

	// when
	rows, total, err := repos.Mystery.List(context.Background(), spec.MysteryListFilter{
		Sort:           "new",
		Limit:          10,
		Offset:         0,
		ExcludeUserIDs: []uuid.UUID{userA.ID},
	})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, idB, rows[0].ID)
	assert.Equal(t, 1, total)
}

func TestMysteryDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	createMystery(t, repos, user.ID, "mine1", "easy", false)
	createMystery(t, repos, user.ID, "mine2", "easy", false)
	createMystery(t, repos, other.ID, "theirs", "easy", false)

	// when
	rows, total, err := repos.Mystery.ListByUser(context.Background(), spec.MysteryUserListFilter{UserID: user.ID, Limit: 10, Offset: 0})

	// then
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, 2, total)
}

func TestMysteryDAO_ListByUser_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	for range 3 {
		createMystery(t, repos, user.ID, "x", "easy", false)
	}

	// when
	rows, total, err := repos.Mystery.ListByUser(context.Background(), spec.MysteryUserListFilter{UserID: user.ID, Limit: 1, Offset: 1})

	// then
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, 3, total)
}

func TestMysteryDAO_AddClue_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	_, err1 := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "first clue", TruthType: "red", SortOrder: 1},
	})
	_, err2 := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "second clue", TruthType: "blue", SortOrder: 0},
	})

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 2)
	assert.Equal(t, "second clue", clues[0].Body)
	assert.Equal(t, "first clue", clues[1].Body)
}

func TestMysteryDAO_AddClue_WithPlayer(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "private", TruthType: "red", SortOrder: 0, PlayerID: &player.ID},
	})

	// then
	require.NoError(t, err)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)
	require.NotNil(t, clues[0].PlayerID)
	assert.Equal(t, player.ID, *clues[0].PlayerID)
}

func TestMysteryDAO_DeleteClues_SkipsPrivate(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "public", TruthType: "red", SortOrder: 0},
	})
	require.NoError(t, err)
	_, err = repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "private", TruthType: "red", SortOrder: 1, PlayerID: &player.ID},
	})
	require.NoError(t, err)

	// when
	err = repos.Mystery.DeleteClues(context.Background(), id)

	// then
	require.NoError(t, err)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)
	assert.Equal(t, "private", clues[0].Body)
}

func TestMysteryDAO_DeleteClue(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "a", TruthType: "red", SortOrder: 0},
	})
	require.NoError(t, err)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)

	// when
	err = repos.Mystery.DeleteClue(context.Background(), clues[0].ID)

	// then
	require.NoError(t, err)
	remaining, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestMysteryDAO_UpdateClue(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "old", TruthType: "red", SortOrder: 0},
	})
	require.NoError(t, err)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)

	// when
	err = repos.Mystery.UpdateClue(context.Background(), spec.MysteryClueUpdate{ClueID: clues[0].ID, Body: "new"})

	// then
	require.NoError(t, err)
	updated, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "new", updated[0].Body)
}

func TestMysteryDAO_CountClues(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "a", TruthType: "red", SortOrder: 0},
	})
	require.NoError(t, err)
	_, err = repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "b", TruthType: "blue", SortOrder: 1},
	})
	require.NoError(t, err)

	// when
	count, err := repos.Mystery.CountClues(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMysteryDAO_CreateAttempt_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	attemptID := createAttempt(t, repos, id, player.ID, nil, "the answer")

	// then
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: player.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, attemptID, attempts[0].ID)
	assert.Equal(t, "the answer", attempts[0].Body)
	assert.False(t, attempts[0].IsWinner)
}

func TestMysteryDAO_CreateAttempt_ThreadedReply(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	parent := createAttempt(t, repos, id, player.ID, nil, "root")

	// when
	reply := createAttempt(t, repos, id, gm.ID, &parent, "reply")

	// then
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: gm.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 2)
	var found bool
	for _, a := range attempts {
		if a.ID == reply {
			require.NotNil(t, a.ParentID)
			assert.Equal(t, parent, *a.ParentID)
			found = true
		}
	}
	assert.True(t, found)
}

func TestMysteryDAO_DeleteAttempt_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttempt(context.Background(), spec.MysteryAttemptDeletion{ID: attemptID, UserID: player.ID})

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: player.ID})
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

func TestMysteryDAO_DeleteAttempt_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttempt(context.Background(), spec.MysteryAttemptDeletion{ID: attemptID, UserID: stranger.ID})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_DeleteAttemptAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttemptAsAdmin(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: player.ID})
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

func TestMysteryDAO_GetAttemptAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	author, err := repos.Mystery.GetAttemptAuthorID(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	assert.Equal(t, player.ID, author)
}

func TestMysteryDAO_GetAttemptMysteryID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	got, err := repos.Mystery.GetAttemptMysteryID(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestMysteryDAO_VoteAttempt_Upvote(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: voter.ID, TargetID: attemptID, Value: 1})

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: voter.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, 1, attempts[0].VoteScore)
	assert.Equal(t, 1, attempts[0].UserVote)
}

func TestMysteryDAO_VoteAttempt_AggregateMultipleVoters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	v1 := daotest.CreateUser(t, repos)
	v2 := daotest.CreateUser(t, repos)
	v3 := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: v1.ID, TargetID: attemptID, Value: 1}))
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: v2.ID, TargetID: attemptID, Value: 1}))
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: v3.ID, TargetID: attemptID, Value: -1}))

	// then
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: v1.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, 1, attempts[0].VoteScore)
}

func TestMysteryDAO_VoteAttempt_ChangeVote(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: voter.ID, TargetID: attemptID, Value: 1}))

	// when
	err := repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: voter.ID, TargetID: attemptID, Value: -1})

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: voter.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, -1, attempts[0].VoteScore)
	assert.Equal(t, -1, attempts[0].UserVote)
}

func TestMysteryDAO_VoteAttempt_ZeroRemovesVote(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	voter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: voter.ID, TargetID: attemptID, Value: 1}))

	// when
	err := repos.Mystery.VoteAttempt(context.Background(), spec.Vote{UserID: voter.ID, TargetID: attemptID, Value: 0})

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: voter.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, 0, attempts[0].VoteScore)
	assert.Equal(t, 0, attempts[0].UserVote)
}

func TestMysteryDAO_MarkSolved(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: attemptID, LockMystery: true})

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.Solved)
	require.NotNil(t, row.WinnerID)
	assert.Equal(t, player.ID, *row.WinnerID)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: player.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.True(t, attempts[0].IsWinner)
}

func TestMysteryDAO_MarkSolved_PreservesPreviousWinner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	p1 := daotest.CreateUser(t, repos)
	p2 := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	a1 := createAttempt(t, repos, id, p1.ID, nil, "first")
	a2 := createAttempt(t, repos, id, p2.ID, nil, "second")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: a1, LockMystery: true}))

	// when
	err := repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: a2, LockMystery: true})

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), spec.MysteryAttemptQuery{MysteryID: id, ViewerID: p2.ID})
	require.NoError(t, err)
	require.Len(t, attempts, 2)
	for _, a := range attempts {
		if a.ID == a1 {
			assert.True(t, a.IsWinner)
		}
		if a.ID == a2 {
			assert.True(t, a.IsWinner)
		}
	}
}

func TestMysteryDAO_MarkSolved_MismatchMysteryFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	m1 := createMystery(t, repos, gm.ID, "T1", "easy", false)
	m2 := createMystery(t, repos, gm.ID, "T2", "easy", false)
	attemptID := createAttempt(t, repos, m1, player.ID, nil, "a")

	// when
	err := repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: m2, AttemptID: attemptID, LockMystery: true})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_IsSolved(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	before, err1 := repos.Mystery.IsSolved(context.Background(), id)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: attemptID, LockMystery: true}))
	after, err2 := repos.Mystery.IsSolved(context.Background(), id)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.False(t, before)
	assert.True(t, after)
}

func TestMysteryDAO_SetPaused_AndIsPaused(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	require.NoError(t, repos.Mystery.SetPaused(context.Background(), spec.MysteryPauseUpdate{MysteryID: id, Paused: true}))
	paused, err := repos.Mystery.IsPaused(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.True(t, paused)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.NotNil(t, row.PausedAt)
}

func TestMysteryDAO_SetPaused_Unpause(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.SetPaused(context.Background(), spec.MysteryPauseUpdate{MysteryID: id, Paused: true}))

	// when
	err := repos.Mystery.SetPaused(context.Background(), spec.MysteryPauseUpdate{MysteryID: id, Paused: false})

	// then
	require.NoError(t, err)
	paused, err := repos.Mystery.IsPaused(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, paused)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Nil(t, row.PausedAt)
}

func TestMysteryDAO_SetGmAway(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	require.NoError(t, repos.Mystery.SetGmAway(context.Background(), spec.MysteryGmAwayUpdate{MysteryID: id, Away: true}))
	awayRow, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, awayRow)
	require.NoError(t, repos.Mystery.SetGmAway(context.Background(), spec.MysteryGmAwayUpdate{MysteryID: id, Away: false}))
	backRow, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, backRow)

	// then
	assert.True(t, awayRow.GmAway)
	assert.False(t, backRow.GmAway)
}

func TestMysteryDAO_CountAttempts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	p1 := daotest.CreateUser(t, repos)
	p2 := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createAttempt(t, repos, id, p1.ID, nil, "a")
	createAttempt(t, repos, id, p2.ID, nil, "b")

	// when
	count, err := repos.Mystery.CountAttempts(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMysteryDAO_GetPlayerIDs_ExcludesAuthor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	p1 := daotest.CreateUser(t, repos)
	p2 := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createAttempt(t, repos, id, p1.ID, nil, "a")
	createAttempt(t, repos, id, p1.ID, nil, "b")
	createAttempt(t, repos, id, p2.ID, nil, "c")
	createAttempt(t, repos, id, gm.ID, nil, "gm reply")

	// when
	ids, err := repos.Mystery.GetPlayerIDs(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.ElementsMatch(t, []uuid.UUID{p1.ID, p2.ID}, ids)
}

func TestMysteryDAO_GetLeaderboard_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	entries, err := repos.Mystery.GetLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMysteryDAO_GetLeaderboard_ScoresByDifficulty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	winner := daotest.CreateUser(t, repos, daotest.WithDisplayName("Winner"))
	easyID := createMystery(t, repos, gm.ID, "e", "easy", false)
	hardID := createMystery(t, repos, gm.ID, "h", "hard", false)
	a1 := createAttempt(t, repos, easyID, winner.ID, nil, "a")
	a2 := createAttempt(t, repos, hardID, winner.ID, nil, "b")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: easyID, AttemptID: a1, LockMystery: true}))
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: hardID, AttemptID: a2, LockMystery: true}))

	// when
	entries, err := repos.Mystery.GetLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, winner.ID, entries[0].UserID)
	assert.Equal(t, 8, entries[0].Score)
	assert.Equal(t, 1, entries[0].EasySolved)
	assert.Equal(t, 1, entries[0].HardSolved)
}

func TestMysteryDAO_GetLeaderboard_Ordering(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	low := daotest.CreateUser(t, repos, daotest.WithDisplayName("Low"))
	high := daotest.CreateUser(t, repos, daotest.WithDisplayName("High"))
	lowM := createMystery(t, repos, gm.ID, "l", "easy", false)
	highM := createMystery(t, repos, gm.ID, "h", "nightmare", false)
	la := createAttempt(t, repos, lowM, low.ID, nil, "a")
	ha := createAttempt(t, repos, highM, high.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: lowM, AttemptID: la, LockMystery: true}))
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: highM, AttemptID: ha, LockMystery: true}))

	// when
	entries, err := repos.Mystery.GetLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, high.ID, entries[0].UserID)
	assert.Equal(t, low.ID, entries[1].UserID)
}

func TestMysteryDAO_GetTopDetectiveIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	winner := daotest.CreateUser(t, repos)
	mID := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, mID, winner.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: mID, AttemptID: attemptID, LockMystery: true}))

	// when
	ids, err := repos.Mystery.GetTopDetectiveIDs(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, winner.ID.String(), ids[0])
}

func TestMysteryDAO_GetGMLeaderboard_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	entries, err := repos.Mystery.GetGMLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMysteryDAO_GetGMLeaderboard_ScoresSolvedMysteries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos, daotest.WithDisplayName("Ruler"))
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "hard", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: attemptID, LockMystery: true}))

	// when
	entries, err := repos.Mystery.GetGMLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, gm.ID, entries[0].UserID)
	assert.Equal(t, 1, entries[0].MysteryCount)
	assert.Equal(t, 1, entries[0].PlayerCount)
	assert.Equal(t, 7, entries[0].Score)
}

func TestMysteryDAO_GetTopGMIDs(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	player := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), spec.MysterySolve{MysteryID: id, AttemptID: attemptID, LockMystery: true}))

	// when
	ids, err := repos.Mystery.GetTopGMIDs(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, gm.ID.String(), ids[0])
}

func TestMysteryDAO_CreateComment_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "nice mystery")

	// then
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Equal(t, "nice mystery", comments[0].Body)
}

func TestMysteryDAO_CreateComment_Threaded(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	parent := createMysteryComment(t, repos, id, nil, commenter.ID, "parent")

	// when
	reply := createMysteryComment(t, repos, id, &parent, gm.ID, "reply")

	// then
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: gm.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 2)
	var found bool
	for _, c := range comments {
		if c.ID == reply {
			require.NotNil(t, c.ParentID)
			assert.Equal(t, parent, *c.ParentID)
			found = true
		}
	}
	assert.True(t, found)
}

func TestMysteryDAO_UpdateComment_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

	// when
	err := repos.Mystery.UpdateComment(context.Background(), spec.CommentUpdate{CommentID: commentID, UserID: commenter.ID, Body: "new body"})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new body", comments[0].Body)
	assert.NotNil(t, comments[0].UpdatedAt)
}

func TestMysteryDAO_UpdateComment_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

	// when
	err := repos.Mystery.UpdateComment(context.Background(), spec.CommentUpdate{CommentID: commentID, UserID: stranger.ID, Body: "hack"})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

	// when
	err := repos.Mystery.UpdateComment(context.Background(), spec.CommentUpdate{CommentID: commentID, Body: "admin edit", AsAdmin: true})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin edit", comments[0].Body)
}

func TestMysteryDAO_DeleteComment_AsOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	err := repos.Mystery.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: commenter.ID})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	assert.Empty(t, comments)
}

func TestMysteryDAO_DeleteComment_NotOwnedFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	err := repos.Mystery.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: stranger.ID})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	err := repos.Mystery.DeleteComment(context.Background(), spec.CommentDeletion{CommentID: commentID, AsAdmin: true})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	assert.Empty(t, comments)
}

func TestMysteryRepo_DeleteCommentWithAudit(t *testing.T) {
	tests := []struct {
		name       string
		asAdmin    bool
		wantAction audit.Action
	}{
		{name: "the owner deleting their own comment", asAdmin: false, wantAction: audit.ActionMysteryCommentDelete},
		{name: "a moderator deleting someone else's comment", asAdmin: true, wantAction: audit.ActionMysteryCommentDeleteAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			gm := daotest.CreateUser(t, repos)
			commenter := daotest.CreateUser(t, repos)
			id := createMystery(t, repos, gm.ID, "T", "easy", false)
			commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

			actor := commenter
			if tt.asAdmin {
				actor = gm
			}

			// when
			_, err := repos.Mystery.DeleteCommentWithAudit(context.Background(), spec.CommentDeletion{
				CommentID: commentID,
				UserID:    actor.ID,
				AsAdmin:   tt.asAdmin,
			})

			// then
			require.NoError(t, err)
			comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
			require.NoError(t, err)
			assert.Empty(t, comments)
			entries, total, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: tt.wantAction, Page: bounds.NewPage(10, 0)})
			require.NoError(t, err)
			assert.Equal(t, 1, total)
			require.Len(t, entries, 1)
			assert.Equal(t, actor.ID, entries[0].ActorID)
			assert.Equal(t, audit.TargetMysteryComment, entries[0].TargetType)
			assert.Equal(t, commentID.String(), entries[0].TargetID)
		})
	}
}

func TestMysteryRepo_DeleteCommentWithAudit_NotOwnedWritesNoAuditRow(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	_, err := repos.Mystery.DeleteCommentWithAudit(context.Background(), spec.CommentDeletion{
		CommentID: commentID,
		UserID:    stranger.ID,
	})

	// then
	require.Error(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: commenter.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, comments, 1)
	_, total, err := repos.AuditLog.List(context.Background(), spec.AuditLogListing{Action: "", Page: bounds.NewPage(10, 0)})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestMysteryDAO_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	c1 := daotest.CreateUser(t, repos)
	c2 := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createMysteryComment(t, repos, id, nil, c1.ID, "A")
	keepID := createMysteryComment(t, repos, id, nil, c2.ID, "B")

	// when
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{
		TargetID:       id,
		ViewerID:       gm.ID,
		Limit:          500,
		Offset:         0,
		ExcludeUserIDs: []uuid.UUID{c1.ID},
	})

	// then
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, keepID, comments[0].ID)
}

func TestMysteryDAO_GetCommentEntityID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	got, err := repos.Mystery.GetCommentEntityID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestMysteryDAO_GetCommentAuthorID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	got, err := repos.Mystery.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, commenter.ID, got)
}

func TestMysteryDAO_LikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	err := repos.Mystery.LikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: liker.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)
}

func TestMysteryDAO_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	require.NoError(t, repos.Mystery.LikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID}))
	require.NoError(t, repos.Mystery.LikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID}))

	// then
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: liker.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
}

func TestMysteryDAO_UnlikeComment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	liker := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	require.NoError(t, repos.Mystery.LikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID}))

	// when
	err := repos.Mystery.UnlikeComment(context.Background(), spec.CommentLike{UserID: liker.ID, CommentID: commentID})

	// then
	require.NoError(t, err)
	comments, _, err := repos.Mystery.GetComments(context.Background(), spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: liker.ID, Limit: 500, Offset: 0})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].LikeCount)
	assert.False(t, comments[0].UserLiked)
}

func TestMysteryDAO_AddCommentMedia_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	mediaID, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/a.png", MediaType: "image", ThumbnailURL: "/t.png"})

	// then
	require.NoError(t, err)
	assert.NotZero(t, mediaID)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
	assert.Equal(t, "/t.png", media[0].ThumbnailURL)
}

func TestMysteryDAO_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	mediaID, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/a.png", MediaType: "image", ThumbnailURL: "/t.png"})
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateCommentMediaURL(context.Background(), spec.MediaURLUpdate{ID: mediaID, URL: "/new.png"})

	// then
	require.NoError(t, err)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].MediaURL)
}

func TestMysteryDAO_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	mediaID, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/a.png", MediaType: "image", ThumbnailURL: "/old.png"})
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateCommentMediaThumbnail(context.Background(), spec.MediaURLUpdate{ID: mediaID, URL: "/new.png"})

	// then
	require.NoError(t, err)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].ThumbnailURL)
}

func TestMysteryDAO_GetCommentMedia_Ordering(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	_, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/a.png", MediaType: "image"})
	require.NoError(t, err)
	_, err = repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/b.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)

	// then
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "/b.png", media[1].MediaURL)
}

func TestMysteryDAO_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	c1 := createMysteryComment(t, repos, id, nil, commenter.ID, "a")
	c2 := createMysteryComment(t, repos, id, nil, commenter.ID, "b")
	_, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: c1, MediaURL: "/c1.png", MediaType: "image"})
	require.NoError(t, err)
	_, err = repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: c2, MediaURL: "/c2.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	result, err := repos.Mystery.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "/c1.png", result[c1][0].MediaURL)
	assert.Equal(t, "/c2.png", result[c2][0].MediaURL)
}

func TestMysteryDAO_GetCommentMediaBatch_EmptyInput(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	result, err := repos.Mystery.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMysteryDAO_AddAttachment_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	attID, err := repos.Mystery.AddAttachment(context.Background(), spec.NewMysteryAttachment{
		MysteryID: id,
		FileURL:   "/file.pdf",
		FileName:  "file.pdf",
		FileSize:  1234,
	})

	// then
	require.NoError(t, err)
	assert.NotZero(t, attID)
	atts, err := repos.Mystery.GetAttachments(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, atts, 1)
	assert.Equal(t, "/file.pdf", atts[0].FileURL)
	assert.Equal(t, "file.pdf", atts[0].FileName)
	assert.Equal(t, 1234, atts[0].FileSize)
}

func TestMysteryDAO_DeleteAttachment(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attID, err := repos.Mystery.AddAttachment(context.Background(), spec.NewMysteryAttachment{
		MysteryID: id,
		FileURL:   "/f.pdf",
		FileName:  "f.pdf",
		FileSize:  1,
	})
	require.NoError(t, err)

	// when
	err = repos.Mystery.DeleteAttachment(context.Background(), spec.MysteryAttachmentDeletion{ID: attID, MysteryID: id})

	// then
	require.NoError(t, err)
	atts, err := repos.Mystery.GetAttachments(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, atts)
}

func TestMysteryDAO_DeleteAttachment_WrongMysteryFails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	m1 := createMystery(t, repos, gm.ID, "A", "easy", false)
	m2 := createMystery(t, repos, gm.ID, "B", "easy", false)
	attID, err := repos.Mystery.AddAttachment(context.Background(), spec.NewMysteryAttachment{
		MysteryID: m1,
		FileURL:   "/f.pdf",
		FileName:  "f.pdf",
		FileSize:  1,
	})
	require.NoError(t, err)

	// when
	err = repos.Mystery.DeleteAttachment(context.Background(), spec.MysteryAttachmentDeletion{ID: attID, MysteryID: m2})

	// then
	require.Error(t, err)
}

func TestMysteryDAO_GetAttachments_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	atts, err := repos.Mystery.GetAttachments(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Empty(t, atts)
}

func TestMysteryDAO_GetByID_AttemptCount_ExcludesAuthor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	p1 := daotest.CreateUser(t, repos)
	p2 := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createAttempt(t, repos, id, p1.ID, nil, "a")
	createAttempt(t, repos, id, p2.ID, nil, "b")
	createAttempt(t, repos, id, gm.ID, nil, "gm")
	createAttempt(t, repos, id, p1.ID, new(createAttempt(t, repos, id, p1.ID, nil, "parent")), "reply")

	// when
	row, err := repos.Mystery.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, 3, row.AttemptCount)
}

func TestMysteryDAO_GetByID_ClueCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "a", TruthType: "red", SortOrder: 0},
	})
	require.NoError(t, err)
	_, err = repos.Mystery.AddClue(context.Background(), spec.NewMysteryClue{
		MysteryID: id,
		NewClue:   spec.NewClue{Body: "b", TruthType: "blue", SortOrder: 1},
	})
	require.NoError(t, err)

	// when
	row, err := repos.Mystery.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, 2, row.ClueCount)
}

func TestMysteryDAO_AddMedia_AndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	mediaID, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/img.png", MediaType: "image", ThumbnailURL: "/t.png"})

	// then
	require.NoError(t, err)
	assert.NotZero(t, mediaID)
	media, err := repos.Mystery.GetMedia(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/img.png", media[0].MediaURL)
	assert.Equal(t, "image", media[0].MediaType)
	assert.Equal(t, "/t.png", media[0].ThumbnailURL)
	assert.Equal(t, 0, media[0].SortOrder)
}

func TestMysteryDAO_UpdateMediaURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	mediaID, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/old.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateMediaURL(context.Background(), spec.MediaURLUpdate{ID: mediaID, URL: "/new.png"})

	// then
	require.NoError(t, err)
	media, err := repos.Mystery.GetMedia(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].MediaURL)
}

func TestMysteryDAO_UpdateMediaThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	mediaID, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/v.mp4", MediaType: "video", ThumbnailURL: "/old.png"})
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateMediaThumbnail(context.Background(), spec.MediaURLUpdate{ID: mediaID, URL: "/new.png"})

	// then
	require.NoError(t, err)
	media, err := repos.Mystery.GetMedia(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].ThumbnailURL)
}

func TestMysteryDAO_GetMedia_Ordering(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/a.png", MediaType: "image"})
	require.NoError(t, err)
	_, err = repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/b.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	media, err := repos.Mystery.GetMedia(context.Background(), id)

	// then
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "/b.png", media[1].MediaURL)
}

func TestMysteryDAO_DeleteMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	mediaID, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/x.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	url, err := repos.Mystery.DeleteMedia(context.Background(), spec.MediaDeletion{ID: mediaID, TargetID: id})

	// then
	require.NoError(t, err)
	assert.Equal(t, "/x.png", url)
	media, err := repos.Mystery.GetMedia(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, media)
}

func TestMysteryDAO_DeleteMedia_WrongMystery(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	otherID := createMystery(t, repos, gm.ID, "Other", "easy", false)
	mediaID, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/x.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	_, err = repos.Mystery.DeleteMedia(context.Background(), spec.MediaDeletion{ID: mediaID, TargetID: otherID})

	// then
	require.Error(t, err)
}

func TestMysteryRepo_DeleteWithFiles_ReturnsEveryUploadedPath(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/uploads/mystery/board.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/board_thumb.png"})
	require.NoError(t, err)
	_, err = repos.Mystery.AddAttachment(context.Background(), spec.NewMysteryAttachment{
		MysteryID: id,
		FileURL:   "/uploads/mystery/case.pdf",
		FileName:  "case.pdf",
		FileSize:  42,
	})
	require.NoError(t, err)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
	_, err = repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/mystery/reply.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/reply_thumb.png"})
	require.NoError(t, err)

	// when
	paths, err := repos.Mystery.DeleteWithFiles(context.Background(), spec.MysteryDelete{ID: id, UserID: gm.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/mystery/board.png",
		"/uploads/mystery/board_thumb.png",
		"/uploads/mystery/case.pdf",
		"/uploads/mystery/reply.png",
		"/uploads/mystery/reply_thumb.png",
	}, paths)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryRepo_DeleteWithFiles_AsAdmin(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	moderator := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/uploads/mystery/board.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/board_thumb.png"})
	require.NoError(t, err)
	_, err = repos.Mystery.AddAttachment(context.Background(), spec.NewMysteryAttachment{
		MysteryID: id,
		FileURL:   "/uploads/mystery/case.pdf",
		FileName:  "case.pdf",
		FileSize:  42,
	})
	require.NoError(t, err)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
	_, err = repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/mystery/reply.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/reply_thumb.png"})
	require.NoError(t, err)

	// when
	paths, err := repos.Mystery.DeleteWithFiles(context.Background(), spec.MysteryDelete{ID: id, UserID: moderator.ID, AsAdmin: true})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/mystery/board.png",
		"/uploads/mystery/board_thumb.png",
		"/uploads/mystery/case.pdf",
		"/uploads/mystery/reply.png",
		"/uploads/mystery/reply_thumb.png",
	}, paths)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryRepo_DeleteWithFiles_SkipsBlankThumbnails(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/uploads/mystery/board.png", MediaType: "image"})
	require.NoError(t, err)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
	_, err = repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/mystery/reply.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	paths, err := repos.Mystery.DeleteWithFiles(context.Background(), spec.MysteryDelete{ID: id, UserID: gm.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/mystery/board.png", "/uploads/mystery/reply.png"}, paths)
}

func TestMysteryRepo_DeleteWithFiles_NotOwnedReturnsNoPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	_, err := repos.Mystery.AddMedia(context.Background(), spec.NewMedia{TargetID: id, MediaURL: "/uploads/mystery/board.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/board_thumb.png"})
	require.NoError(t, err)

	// when
	paths, err := repos.Mystery.DeleteWithFiles(context.Background(), spec.MysteryDelete{ID: id, UserID: stranger.ID})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	media, err := repos.Mystery.GetMedia(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/uploads/mystery/board.png", media[0].MediaURL)
}

func TestMysteryRepo_DeleteWithFiles_NoUploadsReturnsEmpty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	paths, err := repos.Mystery.DeleteWithFiles(context.Background(), spec.MysteryDelete{ID: id, UserID: gm.ID})

	// then
	require.NoError(t, err)
	assert.Empty(t, paths)
}

func TestMysteryRepo_DeleteCommentWithAudit_ReturnsCommentMediaPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
	_, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/mystery/reply.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/reply_thumb.png"})
	require.NoError(t, err)
	otherCommentID := createMysteryComment(t, repos, id, nil, commenter.ID, "another clue")
	_, err = repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: otherCommentID, MediaURL: "/uploads/mystery/keep.png", MediaType: "image"})
	require.NoError(t, err)

	// when
	paths, err := repos.Mystery.DeleteCommentWithAudit(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: commenter.ID})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"/uploads/mystery/reply.png", "/uploads/mystery/reply_thumb.png"}, paths)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), otherCommentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/uploads/mystery/keep.png", media[0].MediaURL)
}

func TestMysteryRepo_DeleteCommentWithAudit_NotOwnedReturnsNoPaths(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	gm := daotest.CreateUser(t, repos)
	commenter := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "a clue")
	_, err := repos.Mystery.AddCommentMedia(context.Background(), spec.NewMedia{TargetID: commentID, MediaURL: "/uploads/mystery/reply.png", MediaType: "image", ThumbnailURL: "/uploads/mystery/reply_thumb.png"})
	require.NoError(t, err)

	// when
	paths, err := repos.Mystery.DeleteCommentWithAudit(context.Background(), spec.CommentDeletion{CommentID: commentID, UserID: stranger.ID})

	// then
	require.Error(t, err)
	assert.Empty(t, paths)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/uploads/mystery/reply.png", media[0].MediaURL)
	assert.Equal(t, "/uploads/mystery/reply_thumb.png", media[0].ThumbnailURL)
}
