package gameroom

import (
	"testing"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPresence(t *testing.T) {
	roomID := uuid.New()
	player := uuid.New()
	spectator := uuid.New()
	offlineAt := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name               string
		playerTabs         int
		offline            bool
		joined             bool
		asPlayer           bool
		userID             uuid.UUID
		wantConnected      bool
		wantDisconnectedAt string
	}{
		{name: "a player closing one of two tabs is still connected", playerTabs: 1, joined: false, asPlayer: true, userID: player, wantConnected: true},
		{name: "a player whose last tab closed is offline with a grace clock", offline: true, joined: false, asPlayer: true, userID: player, wantConnected: false, wantDisconnectedAt: offlineAt.Format(time.RFC3339)},
		{name: "a player outside an active game is offline with no grace clock", joined: false, asPlayer: true, userID: player, wantConnected: false},
		{name: "a spectator who joins is connected", joined: true, asPlayer: false, userID: spectator, wantConnected: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			m := newTestService(t)
			m.svc.mu.Lock()
			st := m.svc.stateFor(roomID)
			st.spectators[spectator] = 1
			if tc.playerTabs > 0 {
				st.players[player] = tc.playerTabs
			}
			if tc.offline {
				st.disconnectedAt[player] = offlineAt
			}
			m.svc.mu.Unlock()

			// when
			got := m.svc.presence(roomID, tc.userID, tc.joined, tc.asPlayer)

			// then
			want := dto.GameRoomPresence{
				RoomID:       roomID,
				UserID:       tc.userID,
				Connected:    tc.wantConnected,
				AsPlayer:     tc.asPlayer,
				WatcherCount: 1,
			}
			if tc.wantDisconnectedAt != "" {
				want.DisconnectedAt = tc.wantDisconnectedAt
				want.GraceSeconds = int(disconnectGracePeriod.Seconds())
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestPlayersOutsideTopic(t *testing.T) {
	// given
	hub := ws.NewHub()
	roomID := uuid.New()
	watching, elsewhere := uuid.New(), uuid.New()
	hub.JoinTopic(gameRoomTopic(roomID), watching)
	players := []dto.GameRoomPlayer{{UserID: watching}, {UserID: elsewhere}}

	// when
	got := playersOutsideTopic(hub, gameRoomTopic(roomID), players)

	// then
	assert.Equal(t, []uuid.UUID{elsewhere}, got, "a player already on the room topic must not get the event twice")
}
