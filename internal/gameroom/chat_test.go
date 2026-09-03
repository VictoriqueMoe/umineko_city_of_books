package gameroom

import (
	"context"
	"strings"
	"testing"
	"umineko_city_of_books/internal/dto"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPostSpectatorChat_ClampsByRunesNotBytes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "an ascii body within the limit is untouched",
			body: "nice move",
			want: "nice move",
		},
		{
			name: "a japanese body the form accepted is no longer chopped",
			body: strings.Repeat("素晴らしい", 40),
			want: strings.Repeat("素晴らしい", 40),
		},
		{
			name: "an ascii body over the limit keeps five hundred characters",
			body: strings.Repeat("a", 600),
			want: strings.Repeat("a", 500),
		},
		{
			name: "a japanese body over the limit keeps five hundred characters",
			body: strings.Repeat("あ", 600),
			want: strings.Repeat("あ", 500),
		},
		{
			name: "an emoji body over the limit is never halved",
			body: strings.Repeat("🎉", 600),
			want: strings.Repeat("🎉", 500),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			m := newTestService(t)
			roomID := uuid.New()
			creator := uuid.New()
			spectator := uuid.New()

			m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(activeRoomRow(roomID, creator), nil)
			m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, spectator).Return(false, nil)
			seedUser(t, m, spectator, "Spectator")

			// when
			msg, err := m.svc.PostSpectatorChat(context.Background(), roomID, spectator, tc.body)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, msg.Body)
			assert.True(t, utf8.ValidString(msg.Body))
			assert.LessOrEqual(t, utf8.RuneCountInString(msg.Body), maxChatBodyLen)
		})
	}
}

func TestPostChat_RejectsAnEmptyBody(t *testing.T) {
	// given
	m := newTestService(t)

	// when
	_, err := m.svc.PostSpectatorChat(context.Background(), uuid.New(), uuid.New(), "   ")

	// then
	require.ErrorIs(t, err, ErrEmptyChat)
}

func TestPostChat_RejectsAMissingRoom(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(nil, nil)

	// when
	_, err := m.svc.PostSpectatorChat(context.Background(), roomID, uuid.New(), "hello")

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPostChat_RejectsARoomThatHasNotStarted(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(pendingRow(roomID, uuid.New()), nil)

	// when
	_, err := m.svc.PostSpectatorChat(context.Background(), roomID, uuid.New(), "hello")

	// then
	require.ErrorIs(t, err, ErrRoomNotActive)
}

func TestPostSpectatorChat_RejectsAPlayer(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	player := uuid.New()
	row := pendingRow(roomID, player)
	row.Status = string(dto.GameStatusActive)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, player).Return(true, nil)

	// when
	_, err := m.svc.PostSpectatorChat(context.Background(), roomID, player, "hello")

	// then
	require.ErrorIs(t, err, ErrPlayersCantChat)
}

func TestPostPlayerChat_RejectsANonParticipant(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	outsider := uuid.New()
	row := pendingRow(roomID, uuid.New())
	row.Status = string(dto.GameStatusActive)
	m.roomRepo.EXPECT().GetRoom(mock.Anything, roomID).Return(row, nil)
	m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, outsider).Return(false, nil)

	// when
	_, err := m.svc.PostPlayerChat(context.Background(), roomID, outsider, "hello")

	// then
	require.ErrorIs(t, err, ErrNotParticipant)
}

func TestGetSpectatorChat_HidesTheSpectatorFeedFromAPlayer(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	player := uuid.New()
	m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, player).Return(true, nil)

	// when
	_, err := m.svc.GetSpectatorChat(context.Background(), roomID, player)

	// then
	require.ErrorIs(t, err, ErrPlayersCantChat)
}

func TestGetSpectatorChat_ReturnsAnEmptyFeedForAnUnknownRoom(t *testing.T) {
	// given
	m := newTestService(t)

	// when
	got, err := m.svc.GetSpectatorChat(context.Background(), uuid.New(), uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, got.Messages)
}

func TestGetPlayerChat_RejectsANonParticipant(t *testing.T) {
	// given
	m := newTestService(t)
	roomID := uuid.New()
	outsider := uuid.New()
	m.roomRepo.EXPECT().IsParticipant(mock.Anything, roomID, outsider).Return(false, nil)

	// when
	_, err := m.svc.GetPlayerChat(context.Background(), roomID, outsider)

	// then
	require.ErrorIs(t, err, ErrNotParticipant)
}
