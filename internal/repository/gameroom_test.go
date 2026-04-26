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

func createFinishedRoom(t *testing.T, repos *repository.Repositories, gameType string, p1, p2 uuid.UUID, winner *uuid.UUID, status string) uuid.UUID {
	t.Helper()
	roomID := uuid.New()
	ctx := context.Background()
	require.NoError(t, repos.GameRoom.CreateRoom(ctx, roomID, gameType, "{}", p1))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, roomID, p1, 0, true))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, roomID, p2, 1, true))
	require.NoError(t, repos.GameRoom.FinishRoom(ctx, roomID, status, winner, "checkmate", "{}"))
	return roomID
}

func TestGameRoomRepository_Scoreboard_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, err := repos.GameRoom.Scoreboard(context.Background(), "chess")

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestGameRoomRepository_Scoreboard_CountsWinsLossesDraws(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos, repotest.WithDisplayName("Alice"))
	bob := repotest.CreateUser(t, repos, repotest.WithDisplayName("Bob"))

	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")
	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")
	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &bob.ID, "finished")
	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, nil, "finished")

	// when
	rows, err := repos.GameRoom.Scoreboard(context.Background(), "chess")

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byUser := map[uuid.UUID]repository.ScoreboardRow{}
	for i := 0; i < len(rows); i++ {
		byUser[rows[i].UserID] = rows[i]
	}

	assert.Equal(t, 2, byUser[alice.ID].Wins)
	assert.Equal(t, 1, byUser[alice.ID].Losses)
	assert.Equal(t, 1, byUser[alice.ID].Draws)

	assert.Equal(t, 1, byUser[bob.ID].Wins)
	assert.Equal(t, 2, byUser[bob.ID].Losses)
	assert.Equal(t, 1, byUser[bob.ID].Draws)
}

func TestGameRoomRepository_Scoreboard_OrdersByWinsThenWinDifferential(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos)
	bob := repotest.CreateUser(t, repos)
	carol := repotest.CreateUser(t, repos)

	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")
	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")
	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")

	createFinishedRoom(t, repos, "chess", carol.ID, bob.ID, &carol.ID, "finished")
	createFinishedRoom(t, repos, "chess", carol.ID, bob.ID, &carol.ID, "finished")
	createFinishedRoom(t, repos, "chess", carol.ID, bob.ID, &carol.ID, "finished")
	createFinishedRoom(t, repos, "chess", carol.ID, bob.ID, &bob.ID, "finished")
	createFinishedRoom(t, repos, "chess", carol.ID, bob.ID, &bob.ID, "finished")

	// when
	rows, err := repos.GameRoom.Scoreboard(context.Background(), "chess")

	// then
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, alice.ID, rows[0].UserID, "alice has 3 wins, 0 losses (diff=+3) — should rank first over carol's 3 wins, 0 losses (diff=+3) only by tiebreak; instead carol has 3 wins, 2 losses (diff=+1), so alice wins outright")
	assert.Equal(t, carol.ID, rows[1].UserID)
	assert.Equal(t, bob.ID, rows[2].UserID)
}

func TestGameRoomRepository_Scoreboard_OnlyFinishedAndAbandoned(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos)
	bob := repotest.CreateUser(t, repos)
	ctx := context.Background()

	pendingID := uuid.New()
	require.NoError(t, repos.GameRoom.CreateRoom(ctx, pendingID, "chess", "{}", alice.ID))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, pendingID, alice.ID, 0, true))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, pendingID, bob.ID, 1, true))

	activeID := uuid.New()
	require.NoError(t, repos.GameRoom.CreateRoom(ctx, activeID, "chess", "{}", alice.ID))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, activeID, alice.ID, 0, true))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, activeID, bob.ID, 1, true))
	require.NoError(t, repos.GameRoom.SetStatus(ctx, activeID, "active"))

	declinedID := uuid.New()
	require.NoError(t, repos.GameRoom.CreateRoom(ctx, declinedID, "chess", "{}", alice.ID))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, declinedID, alice.ID, 0, true))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, declinedID, bob.ID, 1, false))
	require.NoError(t, repos.GameRoom.SetStatus(ctx, declinedID, "declined"))

	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")
	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &bob.ID, "abandoned")

	// when
	rows, err := repos.GameRoom.Scoreboard(ctx, "chess")

	// then
	require.NoError(t, err)
	require.Len(t, rows, 2)
	byUser := map[uuid.UUID]repository.ScoreboardRow{}
	for i := 0; i < len(rows); i++ {
		byUser[rows[i].UserID] = rows[i]
	}
	assert.Equal(t, 1, byUser[alice.ID].Wins)
	assert.Equal(t, 1, byUser[alice.ID].Losses)
	assert.Equal(t, 1, byUser[bob.ID].Wins)
	assert.Equal(t, 1, byUser[bob.ID].Losses)
}

func TestGameRoomRepository_Scoreboard_FiltersByGameType(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos)
	bob := repotest.CreateUser(t, repos)

	createFinishedRoom(t, repos, "chess", alice.ID, bob.ID, &alice.ID, "finished")
	createFinishedRoom(t, repos, "checkers", alice.ID, bob.ID, &bob.ID, "finished")

	// when
	chessRows, err := repos.GameRoom.Scoreboard(context.Background(), "chess")
	require.NoError(t, err)
	checkersRows, err := repos.GameRoom.Scoreboard(context.Background(), "checkers")
	require.NoError(t, err)

	// then
	require.Len(t, chessRows, 2)
	require.Len(t, checkersRows, 2)
	chessByUser := map[uuid.UUID]repository.ScoreboardRow{}
	for i := 0; i < len(chessRows); i++ {
		chessByUser[chessRows[i].UserID] = chessRows[i]
	}
	assert.Equal(t, 1, chessByUser[alice.ID].Wins)
	assert.Equal(t, 0, chessByUser[bob.ID].Wins)
}

func TestGameRoomRepository_Scoreboard_ExcludesUnjoinedPlayers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos)
	bob := repotest.CreateUser(t, repos)
	ctx := context.Background()

	roomID := uuid.New()
	require.NoError(t, repos.GameRoom.CreateRoom(ctx, roomID, "chess", "{}", alice.ID))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, roomID, alice.ID, 0, true))
	require.NoError(t, repos.GameRoom.AddPlayer(ctx, roomID, bob.ID, 1, false))
	require.NoError(t, repos.GameRoom.FinishRoom(ctx, roomID, "abandoned", nil, "abandoned", "{}"))

	// when
	rows, err := repos.GameRoom.Scoreboard(ctx, "chess")

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, alice.ID, rows[0].UserID)
	assert.Equal(t, 0, rows[0].Wins)
	assert.Equal(t, 0, rows[0].Losses)
	assert.Equal(t, 1, rows[0].Draws)
}
