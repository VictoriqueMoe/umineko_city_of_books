package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMystery(t *testing.T, repos *repository.Repositories, userID uuid.UUID, title string, difficulty string, freeForAll bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Mystery.Create(context.Background(), id, userID, title, "body", difficulty, freeForAll))
	return id
}

func createAttempt(t *testing.T, repos *repository.Repositories, mysteryID, userID uuid.UUID, parent *uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Mystery.CreateAttempt(context.Background(), id, mysteryID, userID, parent, body))
	return id
}

func createMysteryComment(t *testing.T, repos *repository.Repositories, mysteryID uuid.UUID, parent *uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Mystery.CreateComment(context.Background(), id, mysteryID, parent, userID, body))
	return id
}

func TestMysteryRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.Mystery.Create(context.Background(), id, user.ID, "The Murder", "Who did it?", "hard", false)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "The Murder", row.Title)
	assert.Equal(t, "Who did it?", row.Body)
	assert.Equal(t, "hard", row.Difficulty)
	assert.False(t, row.FreeForAll)
	assert.False(t, row.Solved)
}

func TestMysteryRepository_Create_FreeForAll(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := uuid.New()

	// when
	err := repos.Mystery.Create(context.Background(), id, user.ID, "FFA", "body", "medium", true)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.FreeForAll)
}

func TestMysteryRepository_GetByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	row, err := repos.Mystery.GetByID(context.Background(), uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryRepository_GetByID_PopulatesAuthor(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Author Name"))
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	row, err := repos.Mystery.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, user.Username, row.AuthorUsername)
	assert.Equal(t, "Author Name", row.AuthorDisplayName)
}

func TestMysteryRepository_Update_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "Old", "easy", false)

	// when
	err := repos.Mystery.Update(context.Background(), id, user.ID, "New", "new body", "hard")

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "New", row.Title)
	assert.Equal(t, "new body", row.Body)
	assert.Equal(t, "hard", row.Difficulty)
}

func TestMysteryRepository_Update_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.Update(context.Background(), id, stranger.ID, "X", "X", "easy")

	// then
	require.Error(t, err)
}

func TestMysteryRepository_UpdateAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.UpdateAsAdmin(context.Background(), id, "Admin Title", "Admin Body", "nightmare", true)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Admin Title", row.Title)
	assert.Equal(t, "nightmare", row.Difficulty)
	assert.True(t, row.FreeForAll)
}

func TestMysteryRepository_Delete_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	err := repos.Mystery.Delete(context.Background(), id, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryRepository_Delete_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.Delete(context.Background(), id, stranger.ID)

	// then
	require.Error(t, err)
}

func TestMysteryRepository_DeleteAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, owner.ID, "T", "easy", false)

	// when
	err := repos.Mystery.DeleteAsAdmin(context.Background(), id)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestMysteryRepository_GetAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	author, err := repos.Mystery.GetAuthorID(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, author)
}

func TestMysteryRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, total, err := repos.Mystery.List(context.Background(), "new", nil, 10, 0, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestMysteryRepository_List_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createMystery(t, repos, user.ID, "T", "easy", false)
	}

	// when
	page1, total1, err1 := repos.Mystery.List(context.Background(), "new", nil, 2, 0, nil)
	page2, total2, err2 := repos.Mystery.List(context.Background(), "new", nil, 2, 2, nil)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 1)
	assert.Equal(t, 3, total1)
	assert.Equal(t, 3, total2)
}

func TestMysteryRepository_List_SortOld(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := createMystery(t, repos, user.ID, "first", "easy", false)
	b := createMystery(t, repos, user.ID, "second", "easy", false)

	// when
	rows, _, err := repos.Mystery.List(context.Background(), "old", nil, 10, 0, nil)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	ids := []uuid.UUID{rows[0].ID, rows[1].ID}
	assert.ElementsMatch(t, []uuid.UUID{a, b}, ids)
}

func TestMysteryRepository_List_FilterSolved(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	solver := repotest.CreateUser(t, repos)
	solvedID := createMystery(t, repos, gm.ID, "solved", "easy", false)
	_ = createMystery(t, repos, gm.ID, "unsolved", "easy", false)
	attemptID := createAttempt(t, repos, solvedID, solver.ID, nil, "answer")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), solvedID, attemptID))
	truthy := true
	falsy := false

	// when
	solved, _, errS := repos.Mystery.List(context.Background(), "new", &truthy, 10, 0, nil)
	unsolved, _, errU := repos.Mystery.List(context.Background(), "new", &falsy, 10, 0, nil)

	// then
	require.NoError(t, errS)
	require.NoError(t, errU)
	require.Len(t, solved, 1)
	require.Len(t, unsolved, 1)
	assert.Equal(t, solvedID, solved[0].ID)
}

func TestMysteryRepository_List_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	userA := repotest.CreateUser(t, repos)
	userB := repotest.CreateUser(t, repos)
	createMystery(t, repos, userA.ID, "A", "easy", false)
	idB := createMystery(t, repos, userB.ID, "B", "easy", false)

	// when
	rows, total, err := repos.Mystery.List(context.Background(), "new", nil, 10, 0, []uuid.UUID{userA.ID})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, idB, rows[0].ID)
	assert.Equal(t, 1, total)
}

func TestMysteryRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	createMystery(t, repos, user.ID, "mine1", "easy", false)
	createMystery(t, repos, user.ID, "mine2", "easy", false)
	createMystery(t, repos, other.ID, "theirs", "easy", false)

	// when
	rows, total, err := repos.Mystery.ListByUser(context.Background(), user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, 2, total)
}

func TestMysteryRepository_ListByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		createMystery(t, repos, user.ID, "x", "easy", false)
	}

	// when
	rows, total, err := repos.Mystery.ListByUser(context.Background(), user.ID, 1, 1)

	// then
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, 3, total)
}

func TestMysteryRepository_AddClue_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)

	// when
	err1 := repos.Mystery.AddClue(context.Background(), id, "first clue", "red", 1, nil)
	err2 := repos.Mystery.AddClue(context.Background(), id, "second clue", "blue", 0, nil)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 2)
	assert.Equal(t, "second clue", clues[0].Body)
	assert.Equal(t, "first clue", clues[1].Body)
}

func TestMysteryRepository_AddClue_WithPlayer(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	err := repos.Mystery.AddClue(context.Background(), id, "private", "red", 0, &player.ID)

	// then
	require.NoError(t, err)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)
	require.NotNil(t, clues[0].PlayerID)
	assert.Equal(t, player.ID, *clues[0].PlayerID)
}

func TestMysteryRepository_DeleteClues_SkipsPrivate(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "public", "red", 0, nil))
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "private", "red", 1, &player.ID))

	// when
	err := repos.Mystery.DeleteClues(context.Background(), id)

	// then
	require.NoError(t, err)
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)
	assert.Equal(t, "private", clues[0].Body)
}

func TestMysteryRepository_DeleteClue(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "a", "red", 0, nil))
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

func TestMysteryRepository_UpdateClue(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "old", "red", 0, nil))
	clues, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, clues, 1)

	// when
	err = repos.Mystery.UpdateClue(context.Background(), clues[0].ID, "new")

	// then
	require.NoError(t, err)
	updated, err := repos.Mystery.GetClues(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "new", updated[0].Body)
}

func TestMysteryRepository_CountClues(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, user.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "a", "red", 0, nil))
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "b", "blue", 1, nil))

	// when
	count, err := repos.Mystery.CountClues(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMysteryRepository_CreateAttempt_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	attemptID := createAttempt(t, repos, id, player.ID, nil, "the answer")

	// then
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, player.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, attemptID, attempts[0].ID)
	assert.Equal(t, "the answer", attempts[0].Body)
	assert.False(t, attempts[0].IsWinner)
}

func TestMysteryRepository_CreateAttempt_ThreadedReply(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	parent := createAttempt(t, repos, id, player.ID, nil, "root")

	// when
	reply := createAttempt(t, repos, id, gm.ID, &parent, "reply")

	// then
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, gm.ID)
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

func TestMysteryRepository_DeleteAttempt_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttempt(context.Background(), attemptID, player.ID)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, player.ID)
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

func TestMysteryRepository_DeleteAttempt_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttempt(context.Background(), attemptID, stranger.ID)

	// then
	require.Error(t, err)
}

func TestMysteryRepository_DeleteAttemptAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.DeleteAttemptAsAdmin(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, player.ID)
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

func TestMysteryRepository_GetAttemptAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	author, err := repos.Mystery.GetAttemptAuthorID(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	assert.Equal(t, player.ID, author)
}

func TestMysteryRepository_GetAttemptMysteryID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	got, err := repos.Mystery.GetAttemptMysteryID(context.Background(), attemptID)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestMysteryRepository_VoteAttempt_Upvote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.VoteAttempt(context.Background(), voter.ID, attemptID, 1)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, voter.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, 1, attempts[0].VoteScore)
	assert.Equal(t, 1, attempts[0].UserVote)
}

func TestMysteryRepository_VoteAttempt_AggregateMultipleVoters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	v1 := repotest.CreateUser(t, repos)
	v2 := repotest.CreateUser(t, repos)
	v3 := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), v1.ID, attemptID, 1))
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), v2.ID, attemptID, 1))
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), v3.ID, attemptID, -1))

	// then
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, v1.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, 1, attempts[0].VoteScore)
}

func TestMysteryRepository_VoteAttempt_ChangeVote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), voter.ID, attemptID, 1))

	// when
	err := repos.Mystery.VoteAttempt(context.Background(), voter.ID, attemptID, -1)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, voter.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, -1, attempts[0].VoteScore)
	assert.Equal(t, -1, attempts[0].UserVote)
}

func TestMysteryRepository_VoteAttempt_ZeroRemovesVote(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	voter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.VoteAttempt(context.Background(), voter.ID, attemptID, 1))

	// when
	err := repos.Mystery.VoteAttempt(context.Background(), voter.ID, attemptID, 0)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, voter.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, 0, attempts[0].VoteScore)
	assert.Equal(t, 0, attempts[0].UserVote)
}

func TestMysteryRepository_MarkSolved(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")

	// when
	err := repos.Mystery.MarkSolved(context.Background(), id, attemptID)

	// then
	require.NoError(t, err)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.Solved)
	require.NotNil(t, row.WinnerID)
	assert.Equal(t, player.ID, *row.WinnerID)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, player.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.True(t, attempts[0].IsWinner)
}

func TestMysteryRepository_MarkSolved_ClearsPreviousWinner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	p1 := repotest.CreateUser(t, repos)
	p2 := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	a1 := createAttempt(t, repos, id, p1.ID, nil, "first")
	a2 := createAttempt(t, repos, id, p2.ID, nil, "second")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), id, a1))

	// when
	err := repos.Mystery.MarkSolved(context.Background(), id, a2)

	// then
	require.NoError(t, err)
	attempts, err := repos.Mystery.GetAttempts(context.Background(), id, p2.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 2)
	for _, a := range attempts {
		if a.ID == a1 {
			assert.False(t, a.IsWinner)
		}
		if a.ID == a2 {
			assert.True(t, a.IsWinner)
		}
	}
}

func TestMysteryRepository_MarkSolved_MismatchMysteryFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	m1 := createMystery(t, repos, gm.ID, "T1", "easy", false)
	m2 := createMystery(t, repos, gm.ID, "T2", "easy", false)
	attemptID := createAttempt(t, repos, m1, player.ID, nil, "a")

	// when
	err := repos.Mystery.MarkSolved(context.Background(), m2, attemptID)

	// then
	require.Error(t, err)
}

func TestMysteryRepository_IsSolved(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	before, err1 := repos.Mystery.IsSolved(context.Background(), id)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), id, attemptID))
	after, err2 := repos.Mystery.IsSolved(context.Background(), id)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.False(t, before)
	assert.True(t, after)
}

func TestMysteryRepository_SetPaused_AndIsPaused(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	require.NoError(t, repos.Mystery.SetPaused(context.Background(), id, true))
	paused, err := repos.Mystery.IsPaused(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.True(t, paused)
	row, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.NotNil(t, row.PausedAt)
}

func TestMysteryRepository_SetPaused_Unpause(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.SetPaused(context.Background(), id, true))

	// when
	err := repos.Mystery.SetPaused(context.Background(), id, false)

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

func TestMysteryRepository_SetGmAway(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	require.NoError(t, repos.Mystery.SetGmAway(context.Background(), id, true))
	awayRow, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, awayRow)
	require.NoError(t, repos.Mystery.SetGmAway(context.Background(), id, false))
	backRow, err := repos.Mystery.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, backRow)

	// then
	assert.True(t, awayRow.GmAway)
	assert.False(t, backRow.GmAway)
}

func TestMysteryRepository_CountAttempts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	p1 := repotest.CreateUser(t, repos)
	p2 := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createAttempt(t, repos, id, p1.ID, nil, "a")
	createAttempt(t, repos, id, p2.ID, nil, "b")

	// when
	count, err := repos.Mystery.CountAttempts(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMysteryRepository_GetPlayerIDs_ExcludesAuthor(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	p1 := repotest.CreateUser(t, repos)
	p2 := repotest.CreateUser(t, repos)
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

func TestMysteryRepository_GetLeaderboard_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	entries, err := repos.Mystery.GetLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMysteryRepository_GetLeaderboard_ScoresByDifficulty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	winner := repotest.CreateUser(t, repos, repotest.WithDisplayName("Winner"))
	easyID := createMystery(t, repos, gm.ID, "e", "easy", false)
	hardID := createMystery(t, repos, gm.ID, "h", "hard", false)
	a1 := createAttempt(t, repos, easyID, winner.ID, nil, "a")
	a2 := createAttempt(t, repos, hardID, winner.ID, nil, "b")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), easyID, a1))
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), hardID, a2))

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

func TestMysteryRepository_GetLeaderboard_Ordering(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	low := repotest.CreateUser(t, repos, repotest.WithDisplayName("Low"))
	high := repotest.CreateUser(t, repos, repotest.WithDisplayName("High"))
	lowM := createMystery(t, repos, gm.ID, "l", "easy", false)
	highM := createMystery(t, repos, gm.ID, "h", "nightmare", false)
	la := createAttempt(t, repos, lowM, low.ID, nil, "a")
	ha := createAttempt(t, repos, highM, high.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), lowM, la))
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), highM, ha))

	// when
	entries, err := repos.Mystery.GetLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, high.ID, entries[0].UserID)
	assert.Equal(t, low.ID, entries[1].UserID)
}

func TestMysteryRepository_GetTopDetectiveIDs(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	winner := repotest.CreateUser(t, repos)
	mID := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, mID, winner.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), mID, attemptID))

	// when
	ids, err := repos.Mystery.GetTopDetectiveIDs(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, winner.ID.String(), ids[0])
}

func TestMysteryRepository_GetGMLeaderboard_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	entries, err := repos.Mystery.GetGMLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMysteryRepository_GetGMLeaderboard_ScoresSolvedMysteries(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos, repotest.WithDisplayName("Ruler"))
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "hard", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), id, attemptID))

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

func TestMysteryRepository_GetTopGMIDs(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	player := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attemptID := createAttempt(t, repos, id, player.ID, nil, "a")
	require.NoError(t, repos.Mystery.MarkSolved(context.Background(), id, attemptID))

	// when
	ids, err := repos.Mystery.GetTopGMIDs(context.Background())

	// then
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, gm.ID.String(), ids[0])
}

func TestMysteryRepository_CreateComment_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "nice mystery")

	// then
	comments, err := repos.Mystery.GetComments(context.Background(), id, commenter.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Equal(t, "nice mystery", comments[0].Body)
}

func TestMysteryRepository_CreateComment_Threaded(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	parent := createMysteryComment(t, repos, id, nil, commenter.ID, "parent")

	// when
	reply := createMysteryComment(t, repos, id, &parent, gm.ID, "reply")

	// then
	comments, err := repos.Mystery.GetComments(context.Background(), id, gm.ID, nil)
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

func TestMysteryRepository_UpdateComment_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

	// when
	err := repos.Mystery.UpdateComment(context.Background(), commentID, commenter.ID, "new body")

	// then
	require.NoError(t, err)
	comments, err := repos.Mystery.GetComments(context.Background(), id, commenter.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "new body", comments[0].Body)
	assert.NotNil(t, comments[0].UpdatedAt)
}

func TestMysteryRepository_UpdateComment_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

	// when
	err := repos.Mystery.UpdateComment(context.Background(), commentID, stranger.ID, "hack")

	// then
	require.Error(t, err)
}

func TestMysteryRepository_UpdateCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "old")

	// when
	err := repos.Mystery.UpdateCommentAsAdmin(context.Background(), commentID, "admin edit")

	// then
	require.NoError(t, err)
	comments, err := repos.Mystery.GetComments(context.Background(), id, commenter.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "admin edit", comments[0].Body)
}

func TestMysteryRepository_DeleteComment_AsOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	err := repos.Mystery.DeleteComment(context.Background(), commentID, commenter.ID)

	// then
	require.NoError(t, err)
	comments, err := repos.Mystery.GetComments(context.Background(), id, commenter.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, comments)
}

func TestMysteryRepository_DeleteComment_NotOwnedFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	stranger := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	err := repos.Mystery.DeleteComment(context.Background(), commentID, stranger.ID)

	// then
	require.Error(t, err)
}

func TestMysteryRepository_DeleteCommentAsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "bye")

	// when
	err := repos.Mystery.DeleteCommentAsAdmin(context.Background(), commentID)

	// then
	require.NoError(t, err)
	comments, err := repos.Mystery.GetComments(context.Background(), id, commenter.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, comments)
}

func TestMysteryRepository_GetComments_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	c1 := repotest.CreateUser(t, repos)
	c2 := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createMysteryComment(t, repos, id, nil, c1.ID, "A")
	keepID := createMysteryComment(t, repos, id, nil, c2.ID, "B")

	// when
	comments, err := repos.Mystery.GetComments(context.Background(), id, gm.ID, []uuid.UUID{c1.ID})

	// then
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, keepID, comments[0].ID)
}

func TestMysteryRepository_GetCommentMysteryID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	got, err := repos.Mystery.GetCommentMysteryID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestMysteryRepository_GetCommentAuthorID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	got, err := repos.Mystery.GetCommentAuthorID(context.Background(), commentID)

	// then
	require.NoError(t, err)
	assert.Equal(t, commenter.ID, got)
}

func TestMysteryRepository_LikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	err := repos.Mystery.LikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	comments, err := repos.Mystery.GetComments(context.Background(), id, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)
}

func TestMysteryRepository_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	require.NoError(t, repos.Mystery.LikeComment(context.Background(), liker.ID, commentID))
	require.NoError(t, repos.Mystery.LikeComment(context.Background(), liker.ID, commentID))

	// then
	comments, err := repos.Mystery.GetComments(context.Background(), id, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
}

func TestMysteryRepository_UnlikeComment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	require.NoError(t, repos.Mystery.LikeComment(context.Background(), liker.ID, commentID))

	// when
	err := repos.Mystery.UnlikeComment(context.Background(), liker.ID, commentID)

	// then
	require.NoError(t, err)
	comments, err := repos.Mystery.GetComments(context.Background(), id, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].LikeCount)
	assert.False(t, comments[0].UserLiked)
}

func TestMysteryRepository_AddCommentMedia_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")

	// when
	mediaID, err := repos.Mystery.AddCommentMedia(context.Background(), commentID, "/a.png", "image", "/t.png", 0)

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

func TestMysteryRepository_UpdateCommentMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	mediaID, err := repos.Mystery.AddCommentMedia(context.Background(), commentID, "/a.png", "image", "/t.png", 0)
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateCommentMediaURL(context.Background(), mediaID, "/new.png")

	// then
	require.NoError(t, err)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].MediaURL)
}

func TestMysteryRepository_UpdateCommentMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	mediaID, err := repos.Mystery.AddCommentMedia(context.Background(), commentID, "/a.png", "image", "/old.png", 0)
	require.NoError(t, err)

	// when
	err = repos.Mystery.UpdateCommentMediaThumbnail(context.Background(), mediaID, "/new.png")

	// then
	require.NoError(t, err)
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/new.png", media[0].ThumbnailURL)
}

func TestMysteryRepository_GetCommentMedia_Ordering(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	commentID := createMysteryComment(t, repos, id, nil, commenter.ID, "x")
	_, err := repos.Mystery.AddCommentMedia(context.Background(), commentID, "/b.png", "image", "", 2)
	require.NoError(t, err)
	_, err = repos.Mystery.AddCommentMedia(context.Background(), commentID, "/a.png", "image", "", 1)
	require.NoError(t, err)

	// when
	media, err := repos.Mystery.GetCommentMedia(context.Background(), commentID)

	// then
	require.NoError(t, err)
	require.Len(t, media, 2)
	assert.Equal(t, "/a.png", media[0].MediaURL)
	assert.Equal(t, "/b.png", media[1].MediaURL)
}

func TestMysteryRepository_GetCommentMediaBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	commenter := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	c1 := createMysteryComment(t, repos, id, nil, commenter.ID, "a")
	c2 := createMysteryComment(t, repos, id, nil, commenter.ID, "b")
	_, err := repos.Mystery.AddCommentMedia(context.Background(), c1, "/c1.png", "image", "", 0)
	require.NoError(t, err)
	_, err = repos.Mystery.AddCommentMedia(context.Background(), c2, "/c2.png", "image", "", 0)
	require.NoError(t, err)

	// when
	result, err := repos.Mystery.GetCommentMediaBatch(context.Background(), []uuid.UUID{c1, c2})

	// then
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "/c1.png", result[c1][0].MediaURL)
	assert.Equal(t, "/c2.png", result[c2][0].MediaURL)
}

func TestMysteryRepository_GetCommentMediaBatch_EmptyInput(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	result, err := repos.Mystery.GetCommentMediaBatch(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMysteryRepository_AddAttachment_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	attID, err := repos.Mystery.AddAttachment(context.Background(), id, "/file.pdf", "file.pdf", 1234)

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

func TestMysteryRepository_DeleteAttachment(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	attID, err := repos.Mystery.AddAttachment(context.Background(), id, "/f.pdf", "f.pdf", 1)
	require.NoError(t, err)

	// when
	err = repos.Mystery.DeleteAttachment(context.Background(), attID, id)

	// then
	require.NoError(t, err)
	atts, err := repos.Mystery.GetAttachments(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, atts)
}

func TestMysteryRepository_DeleteAttachment_WrongMysteryFails(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	m1 := createMystery(t, repos, gm.ID, "A", "easy", false)
	m2 := createMystery(t, repos, gm.ID, "B", "easy", false)
	attID, err := repos.Mystery.AddAttachment(context.Background(), m1, "/f.pdf", "f.pdf", 1)
	require.NoError(t, err)

	// when
	err = repos.Mystery.DeleteAttachment(context.Background(), attID, m2)

	// then
	require.Error(t, err)
}

func TestMysteryRepository_GetAttachments_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)

	// when
	atts, err := repos.Mystery.GetAttachments(context.Background(), id)

	// then
	require.NoError(t, err)
	assert.Empty(t, atts)
}

func TestMysteryRepository_GetByID_AttemptCount_ExcludesAuthor(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	p1 := repotest.CreateUser(t, repos)
	p2 := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	createAttempt(t, repos, id, p1.ID, nil, "a")
	createAttempt(t, repos, id, p2.ID, nil, "b")
	createAttempt(t, repos, id, gm.ID, nil, "gm")
	parent := createAttempt(t, repos, id, p1.ID, nil, "parent")
	createAttempt(t, repos, id, p1.ID, &parent, "reply")

	// when
	row, err := repos.Mystery.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, 3, row.AttemptCount)
}

func TestMysteryRepository_GetByID_ClueCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	gm := repotest.CreateUser(t, repos)
	id := createMystery(t, repos, gm.ID, "T", "easy", false)
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "a", "red", 0, nil))
	require.NoError(t, repos.Mystery.AddClue(context.Background(), id, "b", "blue", 1, nil))

	// when
	row, err := repos.Mystery.GetByID(context.Background(), id)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, 2, row.ClueCount)
}
