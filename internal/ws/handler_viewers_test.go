package ws

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTyping_ReachesOnlyViewers(t *testing.T) {
	tests := []struct {
		name         string
		relays       int
		typistInRoom bool
		wantViewer   []string
	}{
		{name: "one relay reaches the viewer", relays: 1, typistInRoom: true, wantViewer: []string{TypingMessageType}},
		{name: "a typist outside the room is ignored", relays: 1, typistInRoom: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hub := NewHub()
			roomID := uuid.New()
			typist := uuid.New()
			viewer := uuid.New()
			member := uuid.New()
			typistConn := registerTestClient(hub, typist)
			viewerConn := registerTestClient(hub, viewer)
			memberConn := registerTestClient(hub, member)
			if tc.typistInRoom {
				hub.JoinRoom(roomID, typist)
				hub.AddViewer(roomID, typist)
			}
			hub.JoinRoom(roomID, viewer)
			hub.AddViewer(roomID, viewer)
			hub.JoinRoom(roomID, member)
			payload, err := json.Marshal(typingData{RoomID: roomID.String()})
			require.NoError(t, err)

			// when
			for range tc.relays {
				handleWSMessage(typistConn, incomingMessage{Type: TypingMessageType, Data: payload}, hub, nil, nil, nil)
			}

			// then
			assert.Equal(t, tc.wantViewer, drainTypes(t, viewerConn))
			assert.Empty(t, drainTypes(t, memberConn), "a member without the room open never hears typing")
			assert.Empty(t, drainTypes(t, typistConn), "the typist never hears their own typing")
		})
	}
}

func TestPresence_ReachesOnlyViewers(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		data        func(roomID uuid.UUID) any
		arrange     func(hub *Hub, roomID, userID uuid.UUID)
	}{
		{
			name:        "join_room",
			messageType: "join_room",
			data:        func(roomID uuid.UUID) any { return roomActionData{RoomID: roomID.String()} },
			arrange:     func(hub *Hub, roomID, userID uuid.UUID) {},
		},
		{
			name:        "viewer_state",
			messageType: "viewer_state",
			data: func(roomID uuid.UUID) any {
				return viewerStateData{RoomID: roomID.String(), State: ViewerStateIdle}
			},
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.AddViewer(roomID, userID)
			},
		},
		{
			name:        "leave_room",
			messageType: "leave_room",
			data:        func(roomID uuid.UUID) any { return roomActionData{RoomID: roomID.String()} },
			arrange: func(hub *Hub, roomID, userID uuid.UUID) {
				hub.AddViewer(roomID, userID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			hub := NewHub()
			roomID := uuid.New()
			actor := uuid.New()
			viewer := uuid.New()
			member := uuid.New()
			actorConn := registerTestClient(hub, actor)
			viewerConn := registerTestClient(hub, viewer)
			memberConn := registerTestClient(hub, member)
			hub.JoinRooms([]uuid.UUID{roomID}, actor)
			hub.JoinRooms([]uuid.UUID{roomID}, viewer)
			hub.JoinRooms([]uuid.UUID{roomID}, member)
			hub.AddViewer(roomID, viewer)
			tc.arrange(hub, roomID, actor)
			payload, err := json.Marshal(tc.data(roomID))
			require.NoError(t, err)

			// when
			handleWSMessage(actorConn, incomingMessage{Type: tc.messageType, Data: payload}, hub, nil, nil, nil)

			// then
			assert.Equal(t, []string{"chat_presence_changed"}, drainTypes(t, viewerConn))
			assert.Empty(t, drainTypes(t, memberConn), "a member without the room open gets no presence")
		})
	}
}
