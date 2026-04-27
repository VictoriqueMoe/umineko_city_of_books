package gameroom

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	svc        *service
	roomRepo   *repository.MockGameRoomRepository
	userRepo   *repository.MockUserRepository
	blockRepo  *repository.MockBlockRepository
	notifier   *MockNotifier
	handler    *MockGameHandler
	seededByID map[uuid.UUID]model.User
}

func newTestService(t *testing.T) *testMocks {
	t.Helper()
	roomRepo := repository.NewMockGameRoomRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	blockRepo := repository.NewMockBlockRepository(t)
	notifier := NewMockNotifier(t)
	handler := NewMockGameHandler(t)
	handler.EXPECT().GameType().Return(dto.GameTypeChess).Maybe()

	svc := NewService(
		roomRepo,
		userRepo,
		blockRepo,
		notifier,
		ws.NewHub(),
		contentfilter.New(),
		[]GameHandler{handler},
	).(*service)
	m := &testMocks{
		svc:        svc,
		roomRepo:   roomRepo,
		userRepo:   userRepo,
		blockRepo:  blockRepo,
		notifier:   notifier,
		handler:    handler,
		seededByID: map[uuid.UUID]model.User{},
	}
	userRepo.EXPECT().GetByIDs(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, ids []uuid.UUID) ([]model.User, error) {
		out := make([]model.User, 0, len(ids))
		for i := 0; i < len(ids); i++ {
			if u, ok := m.seededByID[ids[i]]; ok {
				out = append(out, u)
			}
		}
		return out, nil
	}).Maybe()
	return m
}

func seedUser(t *testing.T, m *testMocks, id uuid.UUID, name string) {
	t.Helper()
	u := model.User{
		ID:          id,
		Username:    name,
		DisplayName: name,
		Role:        "user",
	}
	m.seededByID[id] = u
	m.userRepo.EXPECT().GetByID(mock.Anything, id).Return(&u, nil).Maybe()
}

func finishedRow(t *testing.T, id, creator uuid.UUID, finishedAt string) repository.GameRoomRow {
	t.Helper()
	return repository.GameRoomRow{
		ID:         id,
		GameType:   string(dto.GameTypeChess),
		Status:     string(dto.GameStatusFinished),
		StateJSON:  `{"fen":"8/8/8/8/8/8/8/8 w - - 0 1","pgn":""}`,
		CreatedBy:  creator,
		CreatedAt:  "2026-04-22T10:00:00Z",
		UpdatedAt:  "2026-04-22T10:30:00Z",
		FinishedAt: &finishedAt,
		Result:     "checkmate",
	}
}

func TestListFinished_Empty(t *testing.T) {
	// given
	m := newTestService(t)
	m.roomRepo.EXPECT().
		ListFinished(mock.Anything, string(dto.GameTypeChess), 20, 0).
		Return(nil, 0, nil)

	// when
	resp, err := m.svc.ListFinished(context.Background(), dto.GameTypeChess, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Rooms)
}

func TestListFinished_DefaultsLimitWhenInvalid(t *testing.T) {
	cases := []struct {
		name     string
		inLimit  int
		expected int
	}{
		{"zero uses default", 0, 20},
		{"negative uses default", -5, 20},
		{"over 50 uses default", 100, 20},
		{"within range is kept", 10, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			m := newTestService(t)
			m.roomRepo.EXPECT().
				ListFinished(mock.Anything, string(dto.GameTypeChess), tc.expected, 0).
				Return(nil, 0, nil)

			// when
			_, err := m.svc.ListFinished(context.Background(), dto.GameTypeChess, tc.inLimit, 0)

			// then
			require.NoError(t, err)
		})
	}
}

func TestListFinished_PropagatesRepoError(t *testing.T) {
	// given
	m := newTestService(t)
	wantErr := errors.New("db down")
	m.roomRepo.EXPECT().
		ListFinished(mock.Anything, "", 20, 0).
		Return(nil, 0, wantErr)

	// when
	_, err := m.svc.ListFinished(context.Background(), "", 20, 0)

	// then
	require.ErrorIs(t, err, wantErr)
}

func TestListFinished_HydratesRoomsWithComputedStats(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	whiteID := uuid.New()
	blackID := uuid.New()
	finishedAt := "2026-04-22T10:30:00Z"
	row := finishedRow(t, roomID, whiteID, finishedAt)

	m.roomRepo.EXPECT().
		ListFinished(mock.Anything, string(dto.GameTypeChess), 20, 0).
		Return([]repository.GameRoomRow{row}, 1, nil)
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: whiteID, Slot: 0, Joined: true},
		{UserID: blackID, Slot: 1, Joined: true},
	}, nil)
	seedUser(t, m, whiteID, "Alice")
	seedUser(t, m, blackID, "Bob")

	statsPayload := map[string]any{
		"total_ply":        42,
		"result_reason":    "checkmate",
		"final_fen":        "8/8/8/8/8/8/8/8 w - - 0 1",
		"duration_seconds": 1800,
	}
	m.handler.EXPECT().
		ComputeStats(row.StateJSON, row.Result, row.CreatedAt, finishedAt).
		Return(statsPayload, nil)

	// when
	resp, err := m.svc.ListFinished(context.Background(), dto.GameTypeChess, 20, 0)

	// then
	require.NoError(t, err)
	require.Len(t, resp.Rooms, 1)
	got := resp.Rooms[0]
	assert.Equal(t, dto.GameStatusFinished, got.Status)
	assert.Equal(t, roomID, got.ID)
	require.NotNil(t, got.FinishedAt)
	assert.Equal(t, finishedAt, *got.FinishedAt)
	require.Len(t, got.Players, 2)

	require.NotNil(t, got.Stats)
	var stats map[string]any
	require.NoError(t, json.Unmarshal(got.Stats, &stats))
	assert.EqualValues(t, 42, stats["total_ply"])
	assert.Equal(t, "checkmate", stats["result_reason"])
}

func TestListFinished_SkipsStatsWhenHandlerFails(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	creator := uuid.New()
	finishedAt := "2026-04-22T10:30:00Z"
	row := finishedRow(t, roomID, creator, finishedAt)

	m.roomRepo.EXPECT().
		ListFinished(mock.Anything, "", 20, 0).
		Return([]repository.GameRoomRow{row}, 1, nil)
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: creator, Slot: 0, Joined: true},
	}, nil)
	seedUser(t, m, creator, "Alice")
	m.handler.EXPECT().
		ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("bad pgn"))

	// when
	resp, err := m.svc.ListFinished(context.Background(), "", 20, 0)

	// then
	require.NoError(t, err)
	require.Len(t, resp.Rooms, 1)
	assert.Nil(t, resp.Rooms[0].Stats)
}

func TestHydrate_ComputesStatsForActiveRoom(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	creator := uuid.New()
	row := &repository.GameRoomRow{
		ID:        roomID,
		GameType:  string(dto.GameTypeChess),
		Status:    string(dto.GameStatusActive),
		StateJSON: `{}`,
		CreatedBy: creator,
		CreatedAt: "2026-04-22T10:00:00Z",
		UpdatedAt: "2026-04-22T10:05:00Z",
	}
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: creator, Slot: 0, Joined: true},
	}, nil)
	seedUser(t, m, creator, "Alice")
	m.handler.EXPECT().
		ComputeStats(row.StateJSON, row.Result, row.CreatedAt, "").
		Return(map[string]any{"total_ply": 4}, nil)

	// when
	got, err := m.svc.hydrateRoom(context.Background(), row)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.Stats, "active rooms should include live stats")
}

func TestHydrate_ExposesDisconnectedAtForOfflinePlayer(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	playerID := uuid.New()
	row := &repository.GameRoomRow{
		ID:        roomID,
		GameType:  string(dto.GameTypeChess),
		Status:    string(dto.GameStatusActive),
		StateJSON: `{}`,
		CreatedBy: playerID,
		CreatedAt: "2026-04-22T10:00:00Z",
		UpdatedAt: "2026-04-22T10:05:00Z",
	}
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: playerID, Slot: 0, Joined: true},
	}, nil)
	seedUser(t, m, playerID, "Alice")
	m.handler.EXPECT().
		ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Maybe()

	disconnectedAt := time.Date(2026, 4, 22, 10, 4, 0, 0, time.UTC)
	m.svc.mu.Lock()
	st := m.svc.stateFor(roomID)
	st.disconnectedAt[playerID] = disconnectedAt
	m.svc.mu.Unlock()

	// when
	got, err := m.svc.hydrateRoom(context.Background(), row)

	// then
	require.NoError(t, err)
	require.Len(t, got.Players, 1)
	p := got.Players[0]
	assert.False(t, p.Connected)
	require.NotNil(t, p.DisconnectedAt)
	assert.Equal(t, disconnectedAt.Format(time.RFC3339), *p.DisconnectedAt)
}

func TestClearForfeit_StopsTimerAndClearsDisconnectedAt(t *testing.T) {
	// given: a room where player is marked disconnected with a running forfeit timer
	m := newTestService(t)
	roomID := uuid.New()
	playerID := uuid.New()
	fired := make(chan struct{}, 1)

	m.svc.mu.Lock()
	st := m.svc.stateFor(roomID)
	st.disconnectedAt[playerID] = time.Now()
	st.timers[playerID] = time.AfterFunc(time.Hour, func() {
		fired <- struct{}{}
	})
	m.svc.mu.Unlock()

	// when
	m.svc.clearForfeit(roomID, playerID)

	// then: disconnect state is cleared, timer was stopped
	m.svc.mu.Lock()
	_, hasDisconnect := st.disconnectedAt[playerID]
	_, hasTimer := st.timers[playerID]
	m.svc.mu.Unlock()
	assert.False(t, hasDisconnect, "disconnectedAt should be cleared")
	assert.False(t, hasTimer, "timer should be removed from map")

	select {
	case <-fired:
		t.Fatal("stopped timer must not fire")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSubmitAction_ClearsDisconnectForfeitOnMove(t *testing.T) {
	// given: an active game where it is player1's turn but player1 was flagged disconnected
	m := newTestService(t)
	roomID := uuid.New()
	player1 := uuid.New()
	player2 := uuid.New()

	row := &repository.GameRoomRow{
		ID:         roomID,
		GameType:   string(dto.GameTypeChess),
		Status:     string(dto.GameStatusActive),
		StateJSON:  `{"fen":"start","pgn":""}`,
		TurnUserID: &player1,
		CreatedBy:  player1,
		CreatedAt:  "2026-04-22T10:00:00Z",
		UpdatedAt:  "2026-04-22T10:05:00Z",
	}
	updatedRow := *row
	updatedRow.StateJSON = `{"fen":"after","pgn":"e4"}`
	updatedRow.TurnUserID = &player2

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil).Once()
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, player1).Return(0, nil)
	next := 1
	m.handler.EXPECT().
		ValidateAction(row.StateJSON, 0, mock.Anything).
		Return(ActionResult{NewStateJSON: updatedRow.StateJSON, NextTurnSlot: &next}, nil)
	m.roomRepo.EXPECT().NextPly(mock.Anything, roomID).Return(1, nil)
	m.roomRepo.EXPECT().AppendMove(mock.Anything, roomID, 1, player1, mock.Anything).Return(nil)
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: player1, Slot: 0, Joined: true},
		{UserID: player2, Slot: 1, Joined: true},
	}, nil).Maybe()
	m.roomRepo.EXPECT().SetState(mock.Anything, roomID, updatedRow.StateJSON, mock.Anything).Return(nil)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(&updatedRow, nil).Maybe()
	seedUser(t, m, player1, "Alice")
	seedUser(t, m, player2, "Bob")
	m.handler.EXPECT().
		ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Maybe()
	m.notifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// seed the disconnect state as if player1's WS had dropped moments ago
	fired := make(chan struct{}, 1)
	m.svc.mu.Lock()
	st := m.svc.stateFor(roomID)
	st.disconnectedAt[player1] = time.Now()
	st.timers[player1] = time.AfterFunc(time.Hour, func() {
		fired <- struct{}{}
	})
	m.svc.mu.Unlock()

	// when: player1 submits a move via HTTP while still flagged offline
	_, err := m.svc.SubmitAction(context.Background(), roomID, player1, json.RawMessage(`{"from":"e2","to":"e4"}`))

	// then
	require.NoError(t, err)
	m.svc.mu.Lock()
	_, hasDisconnect := st.disconnectedAt[player1]
	_, hasTimer := st.timers[player1]
	m.svc.mu.Unlock()
	assert.False(t, hasDisconnect, "submitting a move must clear the disconnect flag")
	assert.False(t, hasTimer, "submitting a move must stop the pending forfeit timer")

	select {
	case <-fired:
		t.Fatal("the forfeit timer must not fire after the player moved")
	case <-time.After(50 * time.Millisecond):
	}
}

func activeRoomRow(roomID, creator uuid.UUID) *repository.GameRoomRow {
	return &repository.GameRoomRow{
		ID:        roomID,
		GameType:  string(dto.GameTypeChess),
		Status:    string(dto.GameStatusActive),
		StateJSON: `{"fen":"start","pgn":""}`,
		CreatedBy: creator,
		CreatedAt: "2026-04-22T10:00:00Z",
		UpdatedAt: "2026-04-22T10:05:00Z",
	}
}

func expectHydrate(t *testing.T, m *testMocks, roomID, p1, p2 uuid.UUID) {
	t.Helper()
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: p1, Slot: 0, Joined: true},
		{UserID: p2, Slot: 1, Joined: true},
	}, nil).Maybe()
	seedUser(t, m, p1, "Alice")
	seedUser(t, m, p2, "Bob")
	m.handler.EXPECT().
		ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Maybe()
}

func pendingRoomRow(roomID, inviter uuid.UUID) *repository.GameRoomRow {
	return &repository.GameRoomRow{
		ID:        roomID,
		GameType:  string(dto.GameTypeChess),
		Status:    string(dto.GameStatusPending),
		StateJSON: `{}`,
		CreatedBy: inviter,
		CreatedAt: "2026-04-22T10:00:00Z",
		UpdatedAt: "2026-04-22T10:00:00Z",
	}
}

func TestAccept_RejectsWhenInviteeNotInRoom(t *testing.T) {
	// given: pending room, invitee has not opened the game page yet (no WS presence)
	m := newTestService(t)
	roomID := uuid.New()
	inviter := uuid.New()
	invitee := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRoomRow(roomID, inviter), nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, invitee).Return(1, nil)
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: inviter, Slot: 0, Joined: true},
		{UserID: invitee, Slot: 1, Joined: false},
	}, nil)
	seedUser(t, m, inviter, "Alice")
	seedUser(t, m, invitee, "Bob")

	// when
	_, err := m.svc.Accept(context.Background(), roomID, invitee)

	// then
	assert.ErrorIs(t, err, ErrAccepterNotInRoom)
}

func TestAccept_RejectsWhenInviterNotInRoom(t *testing.T) {
	// given: pending room, invitee is present but inviter has closed their browser
	m := newTestService(t)
	roomID := uuid.New()
	inviter := uuid.New()
	invitee := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRoomRow(roomID, inviter), nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, invitee).Return(1, nil)
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: inviter, Slot: 0, Joined: true},
		{UserID: invitee, Slot: 1, Joined: false},
	}, nil)
	seedUser(t, m, inviter, "Alice")
	seedUser(t, m, invitee, "Bob")

	// mark only the invitee as WS-present
	m.svc.mu.Lock()
	m.svc.stateFor(roomID).players[invitee] = 1
	m.svc.mu.Unlock()

	// when
	_, err := m.svc.Accept(context.Background(), roomID, invitee)

	// then
	assert.ErrorIs(t, err, ErrInviterNotInRoom)
}

func TestOfferDraw_StoresOfferAndReturnsHydratedRoom(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()
	row := activeRoomRow(roomID, p1)

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p1).Return(0, nil)
	expectHydrate(t, m, roomID, p1, p2)

	room, err := m.svc.OfferDraw(context.Background(), roomID, p1)
	require.NoError(t, err)
	require.NotNil(t, room.DrawOfferFromUser)
	assert.Equal(t, p1, *room.DrawOfferFromUser)

	m.svc.mu.Lock()
	st := m.svc.rooms[roomID]
	m.svc.mu.Unlock()
	require.NotNil(t, st)
	require.NotNil(t, st.draw)
	assert.Equal(t, 0, st.draw.fromSlot)
}

func TestOfferDraw_RejectsWhenOfferAlreadyPending(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p1 := uuid.New()
	row := activeRoomRow(roomID, p1)

	m.svc.mu.Lock()
	m.svc.stateFor(roomID).draw = &drawOffer{fromSlot: 1, offeredAt: time.Now()}
	m.svc.mu.Unlock()

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p1).Return(0, nil)

	_, err := m.svc.OfferDraw(context.Background(), roomID, p1)
	assert.ErrorIs(t, err, ErrDrawOfferPending)
}

func TestOfferDraw_RejectsWhenRoomNotActive(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	row := activeRoomRow(roomID, uuid.New())
	row.Status = string(dto.GameStatusPending)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)

	_, err := m.svc.OfferDraw(context.Background(), roomID, uuid.New())
	assert.ErrorIs(t, err, ErrRoomNotActive)
}

func TestAcceptDraw_FinishesRoomAsDrawAgreed(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()
	row := activeRoomRow(roomID, p1)

	m.svc.mu.Lock()
	m.svc.stateFor(roomID).draw = &drawOffer{fromSlot: 0, offeredAt: time.Now()}
	m.svc.mu.Unlock()

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p2).Return(1, nil)
	m.roomRepo.EXPECT().
		FinishRoom(mock.Anything, roomID, string(dto.GameStatusFinished), (*uuid.UUID)(nil), "draw_agreed", row.StateJSON).
		Return(nil)
	m.roomRepo.EXPECT().CountLive(mock.Anything).Return(0, nil).Maybe()
	expectHydrate(t, m, roomID, p1, p2)
	m.notifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	room, err := m.svc.AcceptDraw(context.Background(), roomID, p2)
	require.NoError(t, err)
	assert.Nil(t, room.DrawOfferFromUser)

	m.svc.mu.Lock()
	st := m.svc.rooms[roomID]
	hasDraw := st != nil && st.draw != nil
	m.svc.mu.Unlock()
	assert.False(t, hasDraw, "accepting must clear the pending offer")
}

func TestAcceptDraw_RejectsOwnOffer(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p1 := uuid.New()
	row := activeRoomRow(roomID, p1)

	m.svc.mu.Lock()
	m.svc.stateFor(roomID).draw = &drawOffer{fromSlot: 0, offeredAt: time.Now()}
	m.svc.mu.Unlock()

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p1).Return(0, nil)

	_, err := m.svc.AcceptDraw(context.Background(), roomID, p1)
	assert.ErrorIs(t, err, ErrCannotAcceptOwnDraw)
}

func TestAcceptDraw_RejectsWhenNoOffer(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p2 := uuid.New()
	row := activeRoomRow(roomID, uuid.New())
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p2).Return(1, nil)

	_, err := m.svc.AcceptDraw(context.Background(), roomID, p2)
	assert.ErrorIs(t, err, ErrNoDrawOffer)
}

func TestDeclineDraw_ClearsOffer(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()
	row := activeRoomRow(roomID, p1)

	m.svc.mu.Lock()
	m.svc.stateFor(roomID).draw = &drawOffer{fromSlot: 0, offeredAt: time.Now()}
	m.svc.mu.Unlock()

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p2).Return(1, nil)
	expectHydrate(t, m, roomID, p1, p2)

	room, err := m.svc.DeclineDraw(context.Background(), roomID, p2)
	require.NoError(t, err)
	assert.Nil(t, room.DrawOfferFromUser)
}

func TestSubmitAction_ClearsPendingDrawOffer(t *testing.T) {
	m := newTestService(t)
	roomID := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()
	row := activeRoomRow(roomID, p1)
	row.TurnUserID = &p1
	updatedRow := *row
	updatedRow.TurnUserID = &p2

	m.svc.mu.Lock()
	m.svc.stateFor(roomID).draw = &drawOffer{fromSlot: 0, offeredAt: time.Now()}
	m.svc.mu.Unlock()

	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil).Once()
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, p1).Return(0, nil)
	nextSlot := 1
	m.handler.EXPECT().ValidateAction(row.StateJSON, 0, mock.Anything).Return(ActionResult{
		NewStateJSON: row.StateJSON,
		NextTurnSlot: &nextSlot,
	}, nil)
	m.roomRepo.EXPECT().NextPly(mock.Anything, roomID).Return(1, nil)
	m.roomRepo.EXPECT().AppendMove(mock.Anything, roomID, 1, p1, mock.Anything).Return(nil)
	m.roomRepo.EXPECT().GetPlayers(mock.Anything, roomID).Return([]repository.GameRoomPlayerRow{
		{UserID: p1, Slot: 0, Joined: true},
		{UserID: p2, Slot: 1, Joined: true},
	}, nil).Maybe()
	m.roomRepo.EXPECT().SetState(mock.Anything, roomID, row.StateJSON, mock.Anything).Return(nil)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(&updatedRow, nil).Maybe()
	seedUser(t, m, p1, "Alice")
	seedUser(t, m, p2, "Bob")
	m.handler.EXPECT().ComputeStats(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	m.notifier.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	_, err := m.svc.SubmitAction(context.Background(), roomID, p1, json.RawMessage(`{"from":"e2","to":"e4"}`))
	require.NoError(t, err)

	m.svc.mu.Lock()
	st := m.svc.rooms[roomID]
	hasDraw := st != nil && st.draw != nil
	m.svc.mu.Unlock()
	assert.False(t, hasDraw, "moving must clear any pending draw offer")
}
