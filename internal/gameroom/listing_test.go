package gameroom

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestList_ClampsThePageAndPassesTheFilterThrough(t *testing.T) {
	// given
	m := newTestService(t)
	userID := uuid.New()
	statuses := []dto.GameStatus{dto.GameStatusActive}
	m.roomRepo.EXPECT().
		ListForUser(mock.Anything, userID, string(dto.GameTypeChess), statuses, bounds.MaxLimit, 0).
		Return(nil, 0, nil)

	// when
	got, err := m.svc.List(context.Background(), userID, ListFilter{
		GameType: dto.GameTypeChess,
		Statuses: statuses,
		Page:     bounds.NewPage(9999, 0),
	})

	// then
	require.NoError(t, err)
	assert.Empty(t, got.Rooms)
}

func TestList_PropagatesARepositoryFailure(t *testing.T) {
	// given
	m := newTestService(t)
	userID := uuid.New()
	m.roomRepo.EXPECT().
		ListForUser(mock.Anything, userID, "", []dto.GameStatus(nil), bounds.DefaultLimit, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	_, err := m.svc.List(context.Background(), userID, ListFilter{Page: bounds.NewPage(0, 0)})

	// then
	require.Error(t, err)
}

func TestListLive_PropagatesARepositoryFailure(t *testing.T) {
	// given
	m := newTestService(t)
	m.roomRepo.EXPECT().
		ListLive(mock.Anything, string(dto.GameTypeChess), 20, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	_, err := m.svc.ListLive(context.Background(), dto.GameTypeChess, bounds.NewPage(20, 0))

	// then
	require.Error(t, err)
}

func TestScoreboard_RejectsAnUnknownGameType(t *testing.T) {
	// given
	m := newTestService(t)

	// when
	_, err := m.svc.Scoreboard(context.Background(), dto.GameType("hopscotch"))

	// then
	require.ErrorIs(t, err, ErrUnknownGameType)
}

func TestScoreboard_ComputesTheWinRateAndSkipsMissingUsers(t *testing.T) {
	// given
	m := newTestService(t)
	known := uuid.New()
	missing := uuid.New()
	seedUser(t, m, known, "battler")
	m.roomRepo.EXPECT().Scoreboard(mock.Anything, string(dto.GameTypeChess)).Return([]repository.ScoreboardRow{
		{UserID: known, Wins: 3, Losses: 1, Draws: 0},
		{UserID: missing, Wins: 1, Losses: 0, Draws: 0},
	}, nil)

	// when
	got, err := m.svc.Scoreboard(context.Background(), dto.GameTypeChess)

	// then
	require.NoError(t, err)
	require.Len(t, got.Rows, 1, "a row whose user cannot be loaded is dropped")
	assert.Equal(t, 4, got.Rows[0].GamesPlayed)
	assert.InDelta(t, 0.75, got.Rows[0].WinRate, 0.0001)
	assert.Equal(t, dto.GameTypeChess, got.GameType)
}

func TestScoreboard_LeavesTheWinRateAtZeroWhenNothingWasPlayed(t *testing.T) {
	// given
	m := newTestService(t)
	userID := uuid.New()
	seedUser(t, m, userID, "beato")
	m.roomRepo.EXPECT().Scoreboard(mock.Anything, string(dto.GameTypeChess)).Return([]repository.ScoreboardRow{
		{UserID: userID},
	}, nil)

	// when
	got, err := m.svc.Scoreboard(context.Background(), dto.GameTypeChess)

	// then
	require.NoError(t, err)
	require.Len(t, got.Rows, 1)
	assert.Zero(t, got.Rows[0].GamesPlayed)
	assert.Zero(t, got.Rows[0].WinRate)
}

func TestScoreboard_PropagatesARepositoryFailure(t *testing.T) {
	// given
	m := newTestService(t)
	m.roomRepo.EXPECT().Scoreboard(mock.Anything, string(dto.GameTypeChess)).Return(nil, errors.New("boom"))

	// when
	_, err := m.svc.Scoreboard(context.Background(), dto.GameTypeChess)

	// then
	require.Error(t, err)
}

func TestCancelIdleGames_ReportsARepositoryFailure(t *testing.T) {
	// given
	m := newTestService(t)
	m.roomRepo.EXPECT().ListIdleActive(mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	// when
	got, err := m.svc.CancelIdleGames(context.Background())

	// then
	require.Error(t, err)
	assert.Zero(t, got)
}

func TestCancelIdleGames_CountsNothingWhenNoRoomIsIdle(t *testing.T) {
	// given
	m := newTestService(t)
	m.roomRepo.EXPECT().ListIdleActive(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := m.svc.CancelIdleGames(context.Background())

	// then
	require.NoError(t, err)
	assert.Zero(t, got)
}

func TestCancelIdleGames_SkipsARoomAnotherWorkerAlreadyClaimed(t *testing.T) {
	// given a room that loses the compare and swap
	m := newTestService(t)
	roomID := uuid.New()
	m.roomRepo.EXPECT().ListIdleActive(mock.Anything, mock.Anything).
		Return([]repository.GameRoomRow{{ID: roomID}}, nil)
	m.roomRepo.EXPECT().CancelIdleRoom(mock.Anything, roomID, mock.Anything).Return(false, nil)

	// when
	got, err := m.svc.CancelIdleGames(context.Background())

	// then
	require.NoError(t, err)
	assert.Zero(t, got, "a room claimed elsewhere must not be counted twice")
}
