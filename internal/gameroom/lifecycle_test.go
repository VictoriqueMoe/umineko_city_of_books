package gameroom

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func pendingRow(id, creator uuid.UUID) *repository.GameRoomRow {
	return &repository.GameRoomRow{
		ID:        id,
		GameType:  string(dto.GameTypeChess),
		Status:    string(dto.GameStatusPending),
		StateJSON: `{"fen":"8/8/8/8/8/8/8/8 w - - 0 1","pgn":""}`,
		CreatedBy: creator,
		CreatedAt: "2026-04-22T10:00:00Z",
		UpdatedAt: "2026-04-22T10:00:00Z",
	}
}

func TestInvite_RejectsSelfInvite(t *testing.T) {
	// given
	m := newTestService(t)
	userID := uuid.New()

	// when
	_, err := m.svc.Invite(context.Background(), userID, userID, dto.GameTypeChess)

	// then
	require.ErrorIs(t, err, ErrSelfInvite)
}

func TestInvite_RejectsUnknownGameType(t *testing.T) {
	// given
	m := newTestService(t)

	// when
	_, err := m.svc.Invite(context.Background(), uuid.New(), uuid.New(), dto.GameType("hopscotch"))

	// then
	require.ErrorIs(t, err, ErrUnknownGameType)
}

func TestInvite_RejectsMissingOpponent(t *testing.T) {
	// given
	m := newTestService(t)
	inviter := uuid.New()
	opponent := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, opponent).Return(nil, nil)

	// when
	_, err := m.svc.Invite(context.Background(), inviter, opponent, dto.GameTypeChess)

	// then
	require.ErrorIs(t, err, ErrOpponentInactive)
}

func TestInvite_RejectsBlockedOpponent(t *testing.T) {
	// given
	m := newTestService(t)
	inviter := uuid.New()
	opponent := uuid.New()
	seedUser(t, m, opponent, "beato")
	m.blockRepo.EXPECT().IsBlockedEither(mock.Anything, inviter, opponent).Return(true, nil)

	// when
	_, err := m.svc.Invite(context.Background(), inviter, opponent, dto.GameTypeChess)

	// then
	require.ErrorIs(t, err, ErrOpponentBlocked)
}

func TestCancel_RejectsAnyoneButTheInviter(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	invitee := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRow(roomID, uuid.New()), nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, invitee).Return(1, nil)

	// when
	err := m.svc.Cancel(context.Background(), roomID, invitee)

	// then
	require.ErrorIs(t, err, ErrNotInviter)
}

func TestDecline_RejectsAnyoneButTheInvitee(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	inviter := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRow(roomID, inviter), nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, inviter).Return(0, nil)

	// when
	err := m.svc.Decline(context.Background(), roomID, inviter)

	// then
	require.ErrorIs(t, err, ErrNotInvitee)
}

func TestDecline_RejectsARoomThatIsNoLongerPending(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	invitee := uuid.New()
	row := pendingRow(roomID, uuid.New())
	row.Status = string(dto.GameStatusActive)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)

	// when
	err := m.svc.Decline(context.Background(), roomID, invitee)

	// then
	require.Error(t, err)
}

func TestGet_ReportsAMissingRoom(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(nil, nil)

	// when
	_, err := m.svc.Get(context.Background(), roomID, uuid.New())

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGet_HidesAPendingRoomFromAnAnonymousViewer(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRow(roomID, uuid.New()), nil)

	// when
	_, err := m.svc.Get(context.Background(), roomID, uuid.Nil)

	// then
	require.ErrorIs(t, err, ErrNotParticipant)
}

func TestGet_HidesAPendingRoomFromANonParticipant(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	viewer := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRow(roomID, uuid.New()), nil)
	m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, viewer).Return(false, nil)

	// when
	_, err := m.svc.Get(context.Background(), roomID, viewer)

	// then
	require.ErrorIs(t, err, ErrNotParticipant)
}

func TestGet_PropagatesARepositoryFailure(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(nil, errors.New("boom"))

	// when
	_, err := m.svc.Get(context.Background(), roomID, uuid.New())

	// then
	require.Error(t, err)
}

func TestResign_RejectsARoomThatIsNotActive(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	userID := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRow(roomID, userID), nil)

	// when
	_, err := m.svc.Resign(context.Background(), roomID, userID)

	// then
	require.ErrorIs(t, err, ErrRoomNotActive)
}

func TestResign_RejectsANonParticipant(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	outsider := uuid.New()
	row := pendingRow(roomID, uuid.New())
	row.Status = string(dto.GameStatusActive)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().GetPlayerSlot(mock.Anything, roomID, outsider).Return(0, errors.New("not a player"))

	// when
	_, err := m.svc.Resign(context.Background(), roomID, outsider)

	// then
	require.ErrorIs(t, err, ErrNotParticipant)
}

func TestCountLive_ReportsTheRepositoryCount(t *testing.T) {
	// given
	m := newTestService(t)
	m.roomRepo.EXPECT().CountLive(mock.Anything).Return(7, nil)

	// when
	got, err := m.svc.CountLive(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 7, got)
}
