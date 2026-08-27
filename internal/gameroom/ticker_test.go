package gameroom

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	tickingGameHandler struct {
		*MockGameHandler
		*MockTickingHandler
	}

	tickerMocks struct {
		*testMocks
		ticking   *MockTickingHandler
		sim       *MockGameSim
		ticks     atomic.Int64
		snapshots atomic.Int64
		newSims   atomic.Int64
	}
)

const (
	testSnapshotType  = "game_test_frame"
	testSnapshotEvery = 3
)

func newTickerTest(t *testing.T, step time.Duration) *tickerMocks {
	t.Helper()

	tm := &tickerMocks{
		testMocks: newTestService(t),
		ticking:   NewMockTickingHandler(t),
		sim:       NewMockGameSim(t),
	}

	tm.ticking.EXPECT().TickInterval().Return(step).Maybe()
	tm.ticking.EXPECT().SnapshotEvery().Return(testSnapshotEvery).Maybe()
	tm.ticking.EXPECT().SnapshotType().Return(testSnapshotType).Maybe()
	tm.ticking.EXPECT().NewSim(mock.Anything, mock.Anything).RunAndReturn(func(uuid.UUID, string) (GameSim, error) {
		tm.newSims.Add(1)
		return tm.sim, nil
	}).Maybe()

	tm.sim.EXPECT().Snapshot().RunAndReturn(func() any {
		tm.snapshots.Add(1)
		return map[string]any{"t": tm.snapshots.Load()}
	}).Maybe()

	return tm
}

func (tm *tickerMocks) start(roomID uuid.UUID, players []dto.GameRoomPlayer) {
	tm.sim.EXPECT().SetConnected(0, true).Once()
	tm.sim.EXPECT().SetConnected(1, true).Once()

	tm.svc.startTicker(roomID, tm.ticking, `{}`, players, [2]bool{true, true})
}

func (tm *tickerMocks) tickAlwaysScores() {
	tm.sim.EXPECT().Tick(mock.Anything).RunAndReturn(func(time.Duration) TickResult {
		tm.ticks.Add(1)
		return TickResult{Scored: true}
	}).Maybe()
}

func (tm *tickerMocks) tickReturns(results ...TickResult) {
	tm.sim.EXPECT().Tick(mock.Anything).RunAndReturn(func(time.Duration) TickResult {
		n := int(tm.ticks.Add(1))
		if n <= len(results) {
			return results[n-1]
		}
		return TickResult{}
	}).Maybe()
}

func activeRow(roomID, creator uuid.UUID) *repository.GameRoomRow {
	return &repository.GameRoomRow{
		ID:        roomID,
		GameType:  string(dto.GameTypeChess),
		Status:    string(dto.GameStatusActive),
		StateJSON: `{"phase":"rally"}`,
		CreatedBy: creator,
		CreatedAt: "2026-04-22T10:00:00Z",
		UpdatedAt: "2026-04-22T10:05:00Z",
	}
}

func twoPlayers(p0, p1 uuid.UUID) []dto.GameRoomPlayer {
	return []dto.GameRoomPlayer{
		{UserID: p0, Slot: 0},
		{UserID: p1, Slot: 1},
	}
}

func TestRunTicker_BroadcastsEveryNthTickAndTouchesNoRepo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := 10 * time.Millisecond
		tm := newTickerTest(t, step)
		tm.tickReturns()
		roomID := uuid.New()

		// when
		tm.start(roomID, twoPlayers(uuid.New(), uuid.New()))
		synctest.Sleep(step*30 + step/2)
		tm.svc.stopTicker(roomID, false)
		synctest.Wait()

		// then
		assert.Equal(t, int64(30), tm.ticks.Load())
		assert.Equal(t, int64(10), tm.snapshots.Load())
		assert.Nil(t, tm.svc.tickerFor(roomID))
	})
}

func TestRunTicker_ScoredWritesStateOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := 10 * time.Millisecond
		tm := newTickerTest(t, step)
		tm.tickReturns(TickResult{Scored: true, WinnerSlot: new(1)})
		roomID := uuid.New()
		player0, player1 := uuid.New(), uuid.New()
		seedUser(t, tm.testMocks, player0, "Alice")
		seedUser(t, tm.testMocks, player1, "Bob")

		tm.sim.EXPECT().StateJSON().Return(`{"scores":[0,1]}`, nil).Once()
		tm.roomRepo.EXPECT().SetState(mock.Anything, roomID, `{"scores":[0,1]}`, (*uuid.UUID)(nil)).Return(nil).Once()
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRow(roomID, player0), nil).Once()
		tm.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
			{UserID: player0, Slot: 0, Joined: true},
			{UserID: player1, Slot: 1, Joined: true},
		}, nil).Once()
		tm.handler.EXPECT().ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()

		// when
		tm.start(roomID, twoPlayers(player0, player1))
		synctest.Sleep(step*5 + step/2)
		tm.svc.stopTicker(roomID, false)
		synctest.Wait()

		// then
		tm.roomRepo.AssertNumberOfCalls(t, "SetState", 1)
		tm.roomRepo.AssertNumberOfCalls(t, "GetRoom", 1)
	})
}

func TestRunTicker_FinishedFinishesRoomAndExits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := 10 * time.Millisecond
		tm := newTickerTest(t, step)
		tm.tickReturns(TickResult{Finished: true, WinnerSlot: new(0), Result: "win"})
		roomID := uuid.New()
		player0, player1 := uuid.New(), uuid.New()
		seedUser(t, tm.testMocks, player0, "Alice")
		seedUser(t, tm.testMocks, player1, "Bob")

		tm.sim.EXPECT().StateJSON().Return(`{"phase":"finished"}`, nil).Once()
		tm.roomRepo.EXPECT().
			FinishRoom(mock.Anything, roomID, string(dto.GameStatusFinished), &player0, "win", `{"phase":"finished"}`).
			Return(nil).
			Once()
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRow(roomID, player0), nil).Once()
		tm.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
			{UserID: player0, Slot: 0, Joined: true},
			{UserID: player1, Slot: 1, Joined: true},
		}, nil).Once()
		tm.roomRepo.EXPECT().CountLive(mock.Anything).Return(0, nil).Maybe()
		tm.handler.EXPECT().ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()
		tm.notifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

		// when
		tm.start(roomID, twoPlayers(player0, player1))
		synctest.Sleep(step * 10)
		synctest.Wait()

		// then: the goroutine cancelled itself from inside finishAndBroadcast without deadlocking
		tm.roomRepo.AssertNumberOfCalls(t, "FinishRoom", 1)
		assert.Equal(t, int64(1), tm.ticks.Load(), "ticking must stop at the finishing tick")
		assert.Nil(t, tm.svc.tickerFor(roomID))
	})
}

func TestStopTicker_TwiceIsANoOpAndHaltsTicking(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := 10 * time.Millisecond
		tm := newTickerTest(t, step)
		tm.tickReturns()
		roomID := uuid.New()
		tm.start(roomID, twoPlayers(uuid.New(), uuid.New()))
		synctest.Sleep(step*3 + step/2)

		// when
		tm.svc.stopTicker(roomID, false)
		tm.svc.stopTicker(roomID, false)
		synctest.Wait()
		frozen := tm.ticks.Load()
		synctest.Sleep(step * 10)

		// then
		assert.Equal(t, int64(3), frozen)
		assert.Equal(t, frozen, tm.ticks.Load(), "no tick may run after the ticker is stopped")
		assert.Nil(t, tm.svc.tickerFor(roomID))
	})
}

func TestRunTicker_HeartbeatReconcilesAfterPersistInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := time.Second
		tm := newTickerTest(t, step)
		tm.tickReturns()
		roomID := uuid.New()
		creator := uuid.New()

		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRow(roomID, creator), nil).Once()
		tm.sim.EXPECT().StateJSON().Return(`{"ticks":60}`, nil).Once()
		tm.roomRepo.EXPECT().SetState(mock.Anything, roomID, `{"ticks":60}`, (*uuid.UUID)(nil)).Return(nil).Once()

		// when
		tm.start(roomID, twoPlayers(uuid.New(), uuid.New()))
		synctest.Sleep(tickerPersistInterval + step)
		tm.svc.stopTicker(roomID, false)
		synctest.Wait()

		// then
		tm.roomRepo.AssertNumberOfCalls(t, "GetRoom", 1)
		tm.roomRepo.AssertNumberOfCalls(t, "SetState", 1)
	})
}

func TestRunTicker_HeartbeatStopsOnNonActiveRoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := time.Second
		tm := newTickerTest(t, step)
		tm.tickReturns()
		roomID := uuid.New()
		creator := uuid.New()
		row := activeRow(roomID, creator)
		row.Status = string(dto.GameStatusFinished)
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil).Once()

		// when
		tm.start(roomID, twoPlayers(uuid.New(), uuid.New()))
		synctest.Sleep(tickerPersistInterval + step)
		synctest.Wait()
		frozen := tm.ticks.Load()
		synctest.Sleep(step * 10)

		// then
		assert.Nil(t, tm.svc.tickerFor(roomID))
		assert.Equal(t, frozen, tm.ticks.Load(), "a non-active row must stop the goroutine")
	})
}

func TestRunTicker_ReconcilesEvenWhenEveryTickScores(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := time.Second
		tm := newTickerTest(t, step)
		tm.tickAlwaysScores()
		roomID := uuid.New()
		row := activeRow(roomID, uuid.New())
		row.Status = string(dto.GameStatusFinished)

		tm.sim.EXPECT().StateJSON().Return("", assert.AnError).Maybe()
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil).Once()

		// when
		tm.start(roomID, twoPlayers(uuid.New(), uuid.New()))
		synctest.Sleep(tickerPersistInterval + step)
		synctest.Wait()
		frozen := tm.ticks.Load()
		synctest.Sleep(step * 10)

		// then
		assert.Nil(t, tm.svc.tickerFor(roomID))
		assert.Equal(t, frozen, tm.ticks.Load(), "a scoring match must still reconcile its status")
	})
}

func TestHandleClientJoin_SpectatorNeverStartsATicker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		tm := newTickerTest(t, 10*time.Millisecond)
		roomID := uuid.New()
		spectator := uuid.New()
		tm.svc.handlers[dto.GameTypeChess] = tickingGameHandler{
			MockGameHandler:    tm.handler,
			MockTickingHandler: tm.ticking,
		}
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRow(roomID, uuid.New()), nil).Once()
		tm.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, spectator).Return(false, nil).Once()

		// when
		tm.svc.HandleClientJoin(t.Context(), spectator, roomID)
		synctest.Wait()

		// then
		assert.Nil(t, tm.svc.tickerFor(roomID))
		assert.Equal(t, int64(0), tm.newSims.Load(), "a spectator must never build a sim")
	})
}

func TestHandleClientInput_RoutesOnlyKnownPlayers(t *testing.T) {
	roomID := uuid.New()
	player0 := uuid.New()
	spectator := uuid.New()
	otherRoom := uuid.New()

	cases := []struct {
		name      string
		userID    uuid.UUID
		roomID    uuid.UUID
		wantSlot  int
		wantApply bool
	}{
		{name: "known player is routed to their slot", userID: player0, roomID: roomID, wantSlot: 0, wantApply: true},
		{name: "spectator is ignored", userID: spectator, roomID: roomID},
		{name: "unknown room is ignored", userID: player0, roomID: otherRoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// given
				step := 10 * time.Millisecond
				tm := newTickerTest(t, step)
				tm.tickReturns()
				if tc.wantApply {
					tm.sim.EXPECT().ApplyInput(tc.wantSlot, mock.Anything).Once()
				}
				tm.start(roomID, twoPlayers(player0, uuid.New()))

				// when
				tm.svc.HandleClientInput(tc.userID, tc.roomID, json.RawMessage(`{"y":400,"seq":7}`))
				synctest.Sleep(step)

				// then
				tm.svc.stopTicker(roomID, false)
				synctest.Wait()
				tm.sim.AssertExpectations(t)
			})
		})
	}
}

func TestHandleClientJoin_ResumesTickerOnceForConcurrentJoins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := 10 * time.Millisecond
		tm := newTickerTest(t, step)
		tm.tickReturns()
		roomID := uuid.New()
		player0, player1 := uuid.New(), uuid.New()
		seedUser(t, tm.testMocks, player0, "Alice")
		seedUser(t, tm.testMocks, player1, "Bob")

		tm.svc.handlers[dto.GameTypeChess] = tickingGameHandler{
			MockGameHandler:    tm.handler,
			MockTickingHandler: tm.ticking,
		}
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRow(roomID, player0), nil)
		tm.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, mock.Anything).Return(true, nil)
		tm.roomRepo.EXPECT().TouchPlayerSeen(mock.Anything, roomID, mock.Anything).Return(nil)
		tm.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
			{UserID: player0, Slot: 0, Joined: true},
			{UserID: player1, Slot: 1, Joined: true},
		}, nil)
		tm.sim.EXPECT().SetConnected(0, true).Times(2)
		tm.sim.EXPECT().SetConnected(1, false).Once()
		tm.sim.EXPECT().SetConnected(1, true).Once()

		// when
		tm.svc.HandleClientJoin(t.Context(), player0, roomID)
		tm.svc.HandleClientJoin(t.Context(), player1, roomID)
		synctest.Sleep(step * 2)

		// then
		require.NotNil(t, tm.svc.tickerFor(roomID))
		assert.Equal(t, int64(1), tm.newSims.Load(), "a second join must not build a second sim")

		tm.svc.stopTicker(roomID, false)
		synctest.Wait()
	})
}

func TestShutdown_FlushesEveryActiveRoomAndReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		step := 10 * time.Millisecond
		tm := newTickerTest(t, step)
		tm.tickReturns()
		roomID := uuid.New()
		creator := uuid.New()

		tm.sim.EXPECT().StateJSON().Return(`{"ticks":5}`, nil).Once()
		tm.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRow(roomID, creator), nil).Once()
		tm.roomRepo.EXPECT().SetState(mock.Anything, roomID, `{"ticks":5}`, (*uuid.UUID)(nil)).Return(nil).Once()

		tm.start(roomID, twoPlayers(uuid.New(), uuid.New()))
		synctest.Sleep(step*5 + step/2)

		// when
		tm.svc.Shutdown(t.Context())

		// then
		tm.roomRepo.AssertNumberOfCalls(t, "SetState", 1)
		assert.Nil(t, tm.svc.tickerFor(roomID))
	})
}

func TestShutdown_WithNoTickersIsANoOp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// given
		tm := newTickerTest(t, 10*time.Millisecond)

		// when
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		tm.svc.Shutdown(ctx)

		// then
		assert.Empty(t, tm.svc.ticks)
	})
}
