package ws

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func registerTestClient(hub *Hub, userID uuid.UUID) *Client {
	client := NewClient(userID, nil)
	hub.mu.Lock()
	hub.clients[userID] = append(hub.clients[userID], client)
	hub.mu.Unlock()

	return client
}

func drainTypes(t *testing.T, client *Client) []string {
	t.Helper()

	var types []string
	for {
		select {
		case data := <-client.send:
			var msg Message
			require.NoError(t, json.Unmarshal(data, &msg))
			types = append(types, msg.Type)
		default:
			return types
		}
	}
}

func TestHub_Unregister_TouchesOnlyTheDepartingUser(t *testing.T) {
	// given
	hub := NewHub()
	leaver := uuid.New()
	stayer := uuid.New()
	shared := uuid.New()
	solo := uuid.New()
	topic := "secret:witchHunter"
	leaverConn := registerTestClient(hub, leaver)
	registerTestClient(hub, stayer)

	hub.JoinRooms([]uuid.UUID{shared, solo}, leaver)
	hub.JoinRoom(shared, stayer)
	hub.JoinTopic(topic, leaver)
	hub.AddViewer(shared, leaver)
	hub.AddViewer(shared, stayer)
	hub.AddViewer(solo, leaver)

	// when
	cleared := hub.Unregister(leaverConn)

	// then
	assert.ElementsMatch(t, []uuid.UUID{shared, solo}, cleared)
	assert.False(t, hub.IsUserInRoom(shared, leaver))
	assert.False(t, hub.IsUserInRoom(solo, leaver))
	assert.False(t, hub.IsUserInRoom(TopicUUID(topic), leaver))
	assert.True(t, hub.IsUserInRoom(shared, stayer))
	assert.True(t, hub.IsUserViewing(shared, stayer))
	assert.NotContains(t, hub.rooms, solo, "an emptied room is dropped")
	assert.NotContains(t, hub.viewers, solo, "an emptied viewer set is dropped")
	assert.NotContains(t, hub.userRooms, leaver)
	assert.NotContains(t, hub.userViewing, leaver)
	assert.Contains(t, hub.userRooms, stayer)
	assert.Contains(t, hub.userViewing, stayer)
}

func TestHub_Unregister_ReverseIndexesFollowMembershipChanges(t *testing.T) {
	tests := []struct {
		name        string
		arrange     func(hub *Hub, roomID, userID uuid.UUID)
		secondConn  bool
		wantCleared bool
		wantInRoom  bool
	}{
		{
			name: "a room left before disconnecting is not touched again",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.LeaveRoom(roomID, userID)
			},
		},
		{
			name: "a closed tab is not reported as a cleared viewer",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.AddViewer(roomID, userID)
				hub.RemoveViewer(roomID, userID)
			},
		},
		{
			name: "a viewer still watching on disconnect is reported",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.AddViewer(roomID, userID)
			},
			wantCleared: true,
		},
		{
			name: "closing one of two connections keeps membership and viewing",
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.JoinRoom(roomID, userID)
				hub.AddViewer(roomID, userID)
			},
			secondConn: true,
			wantInRoom: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hub := NewHub()
			userID := uuid.New()
			roomID := uuid.New()
			conn := registerTestClient(hub, userID)
			if tc.secondConn {
				registerTestClient(hub, userID)
			}
			tc.arrange(hub, roomID, userID)

			// when
			cleared := hub.Unregister(conn)

			// then
			var want []uuid.UUID
			if tc.wantCleared {
				want = []uuid.UUID{roomID}
			}
			assert.Equal(t, want, cleared)
			assert.Equal(t, tc.wantInRoom, hub.IsUserInRoom(roomID, userID))
			assert.Equal(t, tc.wantInRoom, hub.IsUserViewing(roomID, userID))
		})
	}
}

func TestHub_JoinRooms_JoinsEveryRoom(t *testing.T) {
	tests := []struct {
		name  string
		rooms []uuid.UUID
	}{
		{name: "no rooms allocates nothing", rooms: nil},
		{name: "every listed room is joined", rooms: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hub := NewHub()
			userID := uuid.New()

			// when
			hub.JoinRooms(tc.rooms, userID)

			// then
			for _, roomID := range tc.rooms {
				assert.True(t, hub.IsUserInRoom(roomID, userID))
			}
			assert.Len(t, hub.rooms, len(tc.rooms))
			assert.Len(t, hub.userRooms[userID], len(tc.rooms))
		})
	}
}

func TestHub_BroadcastToViewers_ReachesOnlyCurrentViewers(t *testing.T) {
	// given
	hub := NewHub()
	roomID := uuid.New()
	sender := uuid.New()
	viewer := uuid.New()
	member := uuid.New()
	senderConn := registerTestClient(hub, sender)
	viewerConn := registerTestClient(hub, viewer)
	memberConn := registerTestClient(hub, member)
	hub.JoinRooms([]uuid.UUID{roomID}, sender)
	hub.JoinRooms([]uuid.UUID{roomID}, viewer)
	hub.JoinRooms([]uuid.UUID{roomID}, member)
	hub.AddViewer(roomID, sender)
	hub.AddViewer(roomID, viewer)

	// when
	recipients := hub.BroadcastToViewers(roomID, Message{Type: TypingMessageType}, sender)

	// then
	assert.Equal(t, 1, recipients)
	assert.Empty(t, drainTypes(t, senderConn), "the excluded sender is skipped")
	assert.Equal(t, []string{TypingMessageType}, drainTypes(t, viewerConn))
	assert.Empty(t, drainTypes(t, memberConn), "a member without the room open gets nothing")
}

func TestHub_BroadcastPublic_MarshalsOnceForEveryAudience(t *testing.T) {
	// given
	hub := NewHub()
	authed := registerTestClient(hub, uuid.New())
	anon := NewClient(uuid.Nil, nil)
	hub.anon[anon] = struct{}{}

	// when
	hub.BroadcastPublic(Message{Type: "stream_viewers", Data: map[string]any{"viewerCount": 3}})

	// then
	authedData := <-authed.send
	anonData := <-anon.send
	require.NotEmpty(t, authedData)
	assert.Same(t, &authedData[0], &anonData[0], "both audiences share one encoded payload")
}
